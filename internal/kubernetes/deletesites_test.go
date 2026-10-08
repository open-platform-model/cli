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

// TestDeleteCallSites fails when a DELETE or a DELETE of a collection is
// sent from a function outside the allowed list, so a new delete cannot
// bypass the ownership verdict unseen. It reads the non-test Go files under
// internal/, cmd/ and pkg/ and reports every call of a method named Delete
// or DeleteCollection that takes a context first, the shape of the dynamic
// and typed client calls.
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
			for _, decl := range file.Decls {
				fn, ok := decl.(*ast.FuncDecl)
				if !ok || fn.Body == nil {
					continue
				}
				if sendsDelete(fn.Body) {
					got = append(got, filepath.ToSlash(rel)+":"+fn.Name.Name)
				}
			}
			return nil
		})
		require.NoError(t, err)
	}
	sort.Strings(got)
	assert.Equal(t, allowedDeleteSites, got,
		"a delete of an instance's object goes through JudgedDelete; a new exception needs a reason in allowedDeleteSites")
}

// sendsDelete reports whether body calls a method named Delete or
// DeleteCollection whose first argument is named ctx.
func sendsDelete(body *ast.BlockStmt) bool {
	found := false
	ast.Inspect(body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok || len(call.Args) < 2 {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || (sel.Sel.Name != "Delete" && sel.Sel.Name != "DeleteCollection") {
			return true
		}
		// This package's own Delete(ctx, client, opts) is the instance delete
		// loop, which sends its deletes through JudgedDelete.
		if pkg, ok := sel.X.(*ast.Ident); ok && pkg.Name == "kubernetes" {
			return true
		}
		if first, ok := call.Args[0].(*ast.Ident); ok && first.Name == "ctx" {
			found = true
		}
		return true
	})
	return found
}
