package kubernetes

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// retiredHealthNames are the top-level names of the readiness evaluator the
// cli carried before it adopted the library's opm/k8s/health.
var retiredHealthNames = map[string]bool{
	"HealthStatus":        true,
	"EvaluateHealth":      true,
	"QuickInstanceHealth": true,
	"IsHealthy":           true,
}

// healthCopyDecls reports every top-level declaration in file that reuses a
// retired evaluator name, or a function named evaluate…Health, as
// "path:line: name".
func healthCopyDecls(fset *token.FileSet, file *ast.File) []string {
	var found []string
	report := func(name string, pos token.Pos) {
		found = append(found, fmt.Sprintf("%s: %s", fset.Position(pos), name))
	}
	for _, decl := range file.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			if d.Recv != nil {
				continue
			}
			name := d.Name.Name
			if retiredHealthNames[name] || (strings.HasPrefix(name, "evaluate") && strings.HasSuffix(name, "Health")) {
				report(name, d.Name.Pos())
			}
		case *ast.GenDecl:
			for _, spec := range d.Specs {
				switch s := spec.(type) {
				case *ast.TypeSpec:
					if retiredHealthNames[s.Name.Name] {
						report(s.Name.Name, s.Name.Pos())
					}
				case *ast.ValueSpec:
					for _, n := range s.Names {
						if retiredHealthNames[n.Name] {
							report(n.Name, n.Pos())
						}
					}
				}
			}
		}
	}
	return found
}

// libraryHealthPath is the import path of the package that judges readiness.
const libraryHealthPath = "github.com/open-platform-model/library/opm/k8s/health"

// healthAliasDecls reports every top-level type alias or value in file that is
// bound directly to a name of the library's health package, such as
// `type Status = health.Status` or `var Healthy = health.IsHealthy`, as
// "path:line: name". Such a re-export would let the copy's names come back
// as aliases (0012:D3:R6).
func healthAliasDecls(fset *token.FileSet, file *ast.File) []string {
	local := healthImportName(file)
	if local == "" || local == "_" || local == "." {
		return nil
	}
	boundToHealth := func(expr ast.Expr) bool {
		sel, ok := expr.(*ast.SelectorExpr)
		if !ok {
			return false
		}
		id, ok := sel.X.(*ast.Ident)
		return ok && id.Name == local
	}

	var found []string
	report := func(name string, pos token.Pos) {
		found = append(found, fmt.Sprintf("%s: %s", fset.Position(pos), name))
	}
	for _, decl := range file.Decls {
		d, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, spec := range d.Specs {
			switch s := spec.(type) {
			case *ast.TypeSpec:
				if s.Assign.IsValid() && boundToHealth(s.Type) {
					report(s.Name.Name, s.Name.Pos())
				}
			case *ast.ValueSpec:
				if valueBound(s, boundToHealth) {
					for _, n := range s.Names {
						report(n.Name, n.Pos())
					}
				}
			}
		}
	}
	return found
}

// healthImportName returns the name file imports the library's health package
// under, or "" when it does not import it.
func healthImportName(file *ast.File) string {
	for _, imp := range file.Imports {
		importPath, err := strconv.Unquote(imp.Path.Value)
		if err != nil || importPath != libraryHealthPath {
			continue
		}
		if imp.Name != nil {
			return imp.Name.Name
		}
		return path.Base(importPath)
	}
	return ""
}

// valueBound reports whether a var or const spec's type or one of its values
// is bound directly to a name of the health package.
func valueBound(s *ast.ValueSpec, boundToHealth func(ast.Expr) bool) bool {
	if s.Type != nil && boundToHealth(s.Type) {
		return true
	}
	for _, v := range s.Values {
		if boundToHealth(v) {
			return true
		}
	}
	return false
}

// TestNoLocalHealthEvaluator refuses a reintroduced copy of the readiness
// evaluator (0012:D3:R6). Readiness is the library's
// github.com/open-platform-model/library/opm/k8s/health: use health.Status,
// health.Evaluate, health.IsHealthy and health.Aggregate. The check is by
// name, so it catches the old copy coming back; a rewritten evaluator under
// new names is left to review. In internal/kubernetes, where the copy lived,
// it also refuses a type alias or value bound to the library package.
func TestNoLocalHealthEvaluator(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	require.NoError(t, err)

	var found []string
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			switch d.Name() {
			case ".git", ".claude", "testdata", "vendor", "node_modules":
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		fset := token.NewFileSet()
		file, parseErr := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
		if parseErr != nil {
			return parseErr
		}
		found = append(found, healthCopyDecls(fset, file)...)
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return relErr
		}
		if filepath.Dir(filepath.ToSlash(rel)) == "internal/kubernetes" {
			found = append(found, healthAliasDecls(fset, file)...)
		}
		return nil
	})
	require.NoError(t, err)
	assert.Empty(t, found, "the cli declares readiness evaluator names of its own; use github.com/open-platform-model/library/opm/k8s/health instead (0012:D3:R6)")
}

// The scan fires on the copy it exists to refuse.
func TestNoLocalHealthEvaluator_ReportsACopy(t *testing.T) {
	const src = `package kubernetes

import "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

type HealthStatus string

func EvaluateHealth(obj *unstructured.Unstructured) string { return "" }

func evaluatePodHealth() {}

func (c *Client) IsHealthy() bool { return true }
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "copy.go", src, parser.SkipObjectResolution)
	require.NoError(t, err)
	assert.Equal(t, []string{
		"copy.go:5:6: HealthStatus",
		"copy.go:7:6: EvaluateHealth",
		"copy.go:9:6: evaluatePodHealth",
	}, healthCopyDecls(fset, file))
}

// The alias scan fires on a re-export of the library package under any import
// name, and leaves values that only use it alone.
func TestNoLocalHealthEvaluator_ReportsAnAlias(t *testing.T) {
	const src = `package kubernetes

import h "github.com/open-platform-model/library/opm/k8s/health"

type Status = h.Status

var Healthy = h.IsHealthy

var Ready h.Status

const Missing = h.Missing

type statusRow struct{ status h.Status }

var verdicts = []h.Status{h.Ready}
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "alias.go", src, parser.SkipObjectResolution)
	require.NoError(t, err)
	assert.Equal(t, []string{
		"alias.go:5:6: Status",
		"alias.go:7:5: Healthy",
		"alias.go:9:5: Ready",
		"alias.go:11:7: Missing",
	}, healthAliasDecls(fset, file))
}
