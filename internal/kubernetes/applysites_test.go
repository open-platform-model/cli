package kubernetes

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// allowedApplySites are the only functions of the CLI that may send a PATCH
// to the cluster or call this package's Apply or ApplyOne, as "<file relative
// to the repo root>:<function>". Every object of an instance is applied
// through Apply, which the apply workflow (Execute) calls only after the
// ownership guard (inventory.Guard) allowed it; Apply (in applyStage) and
// ApplyOne send through applyOne. Each other entry is a named exception:
//
//   - internal/inventory/store.go: the ModuleInstance record, which is the
//     CLI's own.
//   - internal/operator/install.go:Install: the CRD step, whose objects the
//     guard allowed in the check phase of the same command, and the --rbac
//     objects, which belong to no instance and stay outside the guard.
//   - internal/operator/migration_execute.go:MoveOwnership: the field-ownership
//     moves, on objects the guard allowed in the check phase.
//   - internal/operator/uninstall.go: the finalizer removal on a
//     ModuleInstance record during uninstall.
var allowedApplySites = []string{
	"internal/inventory/store.go:ssaApplyReturning",
	"internal/kubernetes/apply.go:ApplyOne",
	"internal/kubernetes/apply.go:applyOne",
	"internal/kubernetes/apply.go:applyStage",
	"internal/operator/install.go:Install",
	"internal/operator/migration_execute.go:MoveOwnership",
	"internal/operator/uninstall.go:removeOneCleanupFinalizer",
	"internal/workflow/apply/apply.go:Execute",
}

// TestApplyCallSites fails when a PATCH is sent, or this package's Apply or
// ApplyOne is called, from a function outside the allowed list, so a new
// apply cannot bypass the ownership guard unseen. It reads the non-test Go
// files under internal/, cmd/ and pkg/ and reports every call of a method
// named Patch with four or more arguments, the shape of the dynamic and typed
// client calls, every call of a function or method named ApplyOne or
// applyOne, and every call of Apply on this package under any import name,
// wherever the call stands.
func TestApplyCallSites(t *testing.T) {
	root := filepath.Join("..", "..")
	var got []string
	fset := token.NewFileSet()
	for _, dir := range []string{"internal", "cmd", "pkg"} {
		err := filepath.WalkDir(filepath.Join(root, dir), func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			file, err := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
			if err != nil {
				return err
			}
			rel, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			for _, site := range applySites(file) {
				got = append(got, filepath.ToSlash(rel)+":"+site)
			}
			return nil
		})
		require.NoError(t, err)
	}
	sort.Strings(got)
	assert.Equal(t, allowedApplySites, got,
		"an apply of an instance's object goes through Apply, behind the ownership guard; a new exception needs a reason in allowedApplySites")
}

// TestApplySites_Matcher pins what the call-site matcher sees, so the guard
// above cannot go blind.
func TestApplySites_Matcher(t *testing.T) {
	const src = `package x

import (
	k8s "github.com/open-platform-model/cli/internal/kubernetes"
	"github.com/open-platform-model/cli/internal/kubernetes"
)

func patch(ctx context.Context) { _, _ = r.Patch(ctx, "a", types.ApplyPatchType, data, o) }

func patchSubresource(c context.Context) { _, _ = c.Resource(g).Namespace("n").Patch(c, "a", pt, data, o, "status") }

func one(ctx context.Context) { _, _ = kubernetes.ApplyOne(ctx, c, obj, opts) }

func oneLocal(ctx context.Context) { _, _ = ApplyOne(ctx, c, obj, opts) }

func oneUnexported(ctx context.Context) { _, _ = applyOne(ctx, c, obj, opts) }

func all(ctx context.Context) { _, _ = kubernetes.Apply(ctx, c, objs, "name", opts) }

func allRenamed(ctx context.Context) { _, _ = k8s.Apply(ctx, c, objs, "name", opts) }

func otherApply(ctx context.Context) { _ = cfg.Apply(ctx); _, _ = inventory.Apply(ctx, c) }

func inLiteral() { f := func() { _, _ = r.Patch(context.TODO(), "a", pt, data, o) }; f() }

var packageLevel = func() error { _, err := r.Patch(context.Background(), "a", pt, data, o); return err }

func notASend() { p.Patch("key"); _ = jsonpatch.Patch(a, b) }
`
	file, err := parser.ParseFile(token.NewFileSet(), "x.go", src, parser.SkipObjectResolution)
	require.NoError(t, err)
	assert.Equal(t,
		[]string{"patch", "patchSubresource", "one", "oneLocal", "oneUnexported", "all", "allRenamed", "inLiteral", "package-level declaration"},
		applySites(file))
}

// applySites returns, in source order, the top-level declarations of file
// that send a patch or call Apply or ApplyOne: a function or method by its
// name, anything else as "package-level declaration".
func applySites(file *ast.File) []string {
	// The names this file imports the CLI's kubernetes package under.
	ownPackage := map[string]bool{}
	for _, imp := range file.Imports {
		if strings.Trim(imp.Path.Value, `"`) != cliKubernetesPackage {
			continue
		}
		name := "kubernetes"
		if imp.Name != nil {
			name = imp.Name.Name
		}
		ownPackage[name] = true
	}

	var sites []string
	for _, decl := range file.Decls {
		name := "package-level declaration"
		if fn, ok := decl.(*ast.FuncDecl); ok {
			name = fn.Name.Name
		}
		if sendsApply(decl, ownPackage) {
			sites = append(sites, name)
		}
	}
	return sites
}

// sendsApply reports whether node holds a call of a method named Patch with
// four or more arguments, a call of ApplyOne or applyOne, or a call of the
// CLI's own kubernetes.Apply.
func sendsApply(node ast.Node, ownPackage map[string]bool) bool {
	found := false
	ast.Inspect(node, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		switch fun := call.Fun.(type) {
		case *ast.Ident:
			if fun.Name == "ApplyOne" || fun.Name == "applyOne" {
				found = true
			}
		case *ast.SelectorExpr:
			pkg, isIdent := fun.X.(*ast.Ident)
			switch {
			case fun.Sel.Name == "ApplyOne":
				found = true
			case fun.Sel.Name == "Patch" && len(call.Args) >= 4:
				found = true
			case fun.Sel.Name == "Apply" && isIdent && ownPackage[pkg.Name]:
				found = true
			}
		}
		return true
	})
	return found
}
