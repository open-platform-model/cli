package kubernetes

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
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

// TestNoLocalHealthEvaluator refuses a reintroduced copy of the readiness
// evaluator (0012:D3:R6). Readiness is the library's
// github.com/open-platform-model/library/opm/k8s/health: use health.Status,
// health.Evaluate, health.IsHealthy and health.Aggregate. The check is by
// name, so it catches the old copy coming back; a rewritten evaluator under
// new names is left to review.
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
