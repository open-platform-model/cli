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

// allowedDeleteSites are the only functions of the CLI that may send a
// DELETE to the cluster, as "<file relative to the repo root>:<function>".
// Every object of an instance is deleted through JudgedDelete, which asks the
// library's delete verdict first. The two others are named exceptions: the
// operator install's migration deletes run under a verdict taken when the
// plan was made, and the ModuleInstance record is the CLI's own.
var allowedDeleteSites = []string{
	"internal/inventory/store.go:DeleteCR",
	"internal/kubernetes/delete.go:JudgedDelete",
	"internal/operator/migration_execute.go:deleteProven",
}

// cliKubernetesPackage is this package's import path. Its own
// Delete(ctx, client, opts) is the instance delete loop, which sends its
// deletes through JudgedDelete, so a call of it is not a send.
const cliKubernetesPackage = "github.com/open-platform-model/cli/internal/kubernetes"

// TestDeleteCallSites fails when a DELETE or a DELETE of a collection is
// sent from a function outside the allowed list, so a new delete cannot
// bypass the ownership verdict unseen. It reads the non-test Go files under
// internal/, cmd/ and pkg/ and reports every call of a method named Delete
// or DeleteCollection with three or more arguments, the shape of the dynamic
// and typed client calls, whatever the arguments are and wherever the call
// stands: in a function, a method, or a function literal of a package-level
// variable.
func TestDeleteCallSites(t *testing.T) {
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
			for _, site := range deleteSites(file) {
				got = append(got, filepath.ToSlash(rel)+":"+site)
			}
			return nil
		})
		require.NoError(t, err)
	}
	sort.Strings(got)
	assert.Equal(t, allowedDeleteSites, got,
		"a delete of an instance's object goes through JudgedDelete; a new exception needs a reason in allowedDeleteSites")
}

// TestDeleteSites_Matcher pins what the call-site matcher sees, so the guard
// above cannot go blind: a send is found whatever its context argument is
// called and inside a function literal, and this package's own Delete is not
// one, under any import name.
func TestDeleteSites_Matcher(t *testing.T) {
	const src = `package x

import (
	"context"

	k8s "github.com/open-platform-model/cli/internal/kubernetes"
	"github.com/open-platform-model/cli/internal/kubernetes"
)

func named(ctx context.Context) { _ = r.Delete(ctx, "a", o) }

func background() { _ = r.Delete(context.Background(), "a", o) }

func otherName(waitCtx context.Context) { _ = c.Resource(g).Namespace("n").Delete(waitCtx, "a", o) }

func collection(ctx context.Context) { _ = r.DeleteCollection(ctx, o, l) }

func inLiteral() { f := func() { _ = r.Delete(context.TODO(), "a", o) }; f() }

var packageLevel = func() error { return r.Delete(context.Background(), "a", o) }

func instanceLoop(ctx context.Context) { _, _ = kubernetes.Delete(ctx, c, o) }

func instanceLoopRenamed(ctx context.Context) { _, _ = k8s.Delete(ctx, c, o) }

func notASend() { m.Delete("key"); _ = os.Remove("f") }
`
	file, err := parser.ParseFile(token.NewFileSet(), "x.go", src, parser.SkipObjectResolution)
	require.NoError(t, err)
	assert.Equal(t,
		[]string{"named", "background", "otherName", "collection", "inLiteral", "package-level declaration"},
		deleteSites(file))
}

// deleteSites returns, in source order, the top-level declarations of file
// that send a delete: a function or method by its name, anything else as
// "package-level declaration".
func deleteSites(file *ast.File) []string {
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
		if sendsDelete(decl, ownPackage) {
			sites = append(sites, name)
		}
	}
	return sites
}

// sendsDelete reports whether node holds a call of a method named Delete or
// DeleteCollection with three or more arguments, other than a call of the
// CLI's own kubernetes.Delete.
func sendsDelete(node ast.Node, ownPackage map[string]bool) bool {
	found := false
	ast.Inspect(node, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok || len(call.Args) < 3 {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || (sel.Sel.Name != "Delete" && sel.Sel.Name != "DeleteCollection") {
			return true
		}
		if pkg, ok := sel.X.(*ast.Ident); ok && ownPackage[pkg.Name] {
			return true
		}
		found = true
		return true
	})
	return found
}
