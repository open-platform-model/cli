package errors

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// textMatchAllowed lists the only functions in non-test cli code that may
// test the text of an error, as "<path from the repo root>: <function>", each
// with the typed cause that is missing. None of them is a registry fetch or
// a dependency resolution failure: those are decided from the library's
// typed errors (0021:D8:R12). A fifth entry needs the same kind of reason.
var textMatchAllowed = map[string]string{
	"internal/cuemod/tidy.go: classify":                            "cmd/cue prints CUE's internal modload.ErrModuleNotTidy; no public type reaches it",
	"internal/publish/identity.go: conformIdentity":                "the public CUE API gives an incomplete-value error no type and no code",
	"pkg/errors/grouped_errors.go: groupCUEErrors":                 "CUE's disjunction summary line has no type; the filter only drops a display line",
	"internal/workflow/render/validation.go: printValidationError": "the library's DuplicateIdentitiesError exposes its header and rows only as one message; the type decides, the cut lays it out",
}

// stringPredicates are the strings functions that test or split a text.
var stringPredicates = map[string]bool{
	"Contains": true, "ContainsAny": true, "HasPrefix": true, "HasSuffix": true,
	"Index": true, "LastIndex": true, "EqualFold": true, "Cut": true,
	"CutPrefix": true, "CutSuffix": true, "Split": true, "SplitN": true, "Fields": true,
}

// regexpPredicates are the regexp methods that test or pick apart a text.
var regexpPredicates = map[string]bool{
	"MatchString": true, "FindString": true, "FindStringSubmatch": true,
	"FindAllString": true, "FindAllStringSubmatch": true, "FindStringIndex": true,
}

// isErrorText reports whether expr is the text of an error: a call of a
// method named Error, or of Msg (the message accessor of a CUE error).
func isErrorText(expr ast.Expr) bool {
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		return false
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	return ok && (sel.Sel.Name == "Error" || sel.Sel.Name == "Msg") && len(call.Args) == 0
}

// mentions reports whether expr holds an error text or names a variable in
// tainted.
func mentions(expr ast.Expr, tainted map[string]bool) bool {
	found := false
	ast.Inspect(expr, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.CallExpr:
			if isErrorText(x) {
				found = true
			}
		case *ast.Ident:
			if tainted[x.Name] {
				found = true
			}
		}
		return !found
	})
	return found
}

// errorTextVars returns the variables of fn that hold an error's text: one
// assigned from an Error or Msg call, or from an expression over such a
// variable (msg := err.Error(); text := strings.TrimSpace(msg)).
func errorTextVars(fn *ast.FuncDecl) map[string]bool {
	tainted := map[string]bool{}
	for changed := true; changed; {
		changed = false
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			assign, ok := n.(*ast.AssignStmt)
			if !ok {
				return true
			}
			hit := false
			for _, rhs := range assign.Rhs {
				if mentions(rhs, tainted) {
					hit = true
				}
			}
			if !hit {
				return true
			}
			for _, lhs := range assign.Lhs {
				if id, ok := lhs.(*ast.Ident); ok && id.Name != "_" && !tainted[id.Name] {
					tainted[id.Name] = true
					changed = true
				}
			}
			return true
		})
	}
	return tainted
}

// predicateOver reports whether call is a strings or regexp predicate with
// an argument holdsText accepts.
func predicateOver(call *ast.CallExpr, holdsText func(ast.Expr) bool) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	pkg, _ := sel.X.(*ast.Ident)
	stringsCall := pkg != nil && pkg.Name == "strings" && stringPredicates[sel.Sel.Name]
	if !stringsCall && !regexpPredicates[sel.Sel.Name] {
		return false
	}
	for _, arg := range call.Args {
		if holdsText(arg) {
			return true
		}
	}
	return false
}

// errorTextMatches reports every place in fn that tests an error's text: a
// strings or regexp predicate over an error text or a variable holding one,
// a comparison of an Error() call, and a switch over one. The text may be
// wrapped in other calls (strings.ToLower(err.Error())).
//
// Two forms are not seen, because the guard reads syntax and no types: a
// text made by formatting the error (fmt.Sprint(err), fmt.Sprintf("%v",
// err)), and a text handed to another function as a string, which is not
// followed there. Pass the error, not its text. TestErrorTextMatches_Detects
// holds both limits as rows.
func errorTextMatches(fset *token.FileSet, fn *ast.FuncDecl) []token.Position {
	if fn.Body == nil {
		return nil
	}
	tainted := errorTextVars(fn)
	// An argument holds the text when the text is anywhere inside it, so a
	// lower-cased or trimmed text counts: strings.ToLower(err.Error()).
	holdsText := func(expr ast.Expr) bool { return mentions(expr, tainted) }
	var found []token.Position
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.CallExpr:
			if predicateOver(x, holdsText) {
				found = append(found, fset.Position(x.Pos()))
			}
		case *ast.BinaryExpr:
			if (x.Op == token.EQL || x.Op == token.NEQ) && (isErrorText(x.X) || isErrorText(x.Y)) {
				found = append(found, fset.Position(x.Pos()))
			}
		case *ast.SwitchStmt:
			if x.Tag != nil && holdsText(x.Tag) {
				found = append(found, fset.Position(x.Pos()))
			}
		}
		return true
	})
	return found
}

// errorTextMatchesIn parses src as the file at rel and returns every match as
// "<rel>: <function>" with its positions.
func errorTextMatchesIn(t *testing.T, rel string, src []byte) map[string][]string {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, rel, src, parser.SkipObjectResolution)
	require.NoError(t, err)
	out := map[string][]string{}
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}
		for _, pos := range errorTextMatches(fset, fn) {
			key := rel + ": " + fn.Name.Name
			out[key] = append(out[key], fmt.Sprintf("%s:%d", rel, pos.Line))
		}
	}
	return out
}

// TestNoErrorTextMatch refuses a new decision on the text of an error in
// non-test cli code. The cli decides from error types and CUE error paths;
// the allowlist holds the four places where no type exists, and each entry
// must still be in the code, so a removed match leaves the list too.
func TestNoErrorTextMatch(t *testing.T) {
	const root = "../.."
	seen := map[string]bool{}
	for _, tree := range []string{"cmd", "internal", "pkg"} {
		require.NoError(t, filepath.WalkDir(filepath.Join(root, tree), func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				if d.Name() == "testdata" {
					return filepath.SkipDir
				}
				return nil
			}
			if !strings.HasSuffix(p, ".go") || strings.HasSuffix(p, "_test.go") {
				return nil
			}
			rel, err := filepath.Rel(root, p)
			if err != nil {
				return err
			}
			src, err := os.ReadFile(p)
			if err != nil {
				return err
			}
			for key, positions := range errorTextMatchesIn(t, filepath.ToSlash(rel), src) {
				seen[key] = true
				if _, ok := textMatchAllowed[key]; !ok {
					assert.Fail(t, "an error's text decides here",
						"%s (%s): decide from the error's type (errors.Is, errors.As, the library's Classify) or from a CUE error path, not from its message",
						key, strings.Join(positions, ", "))
				}
			}
			return nil
		}))
	}
	var stale []string
	for key := range textMatchAllowed {
		if !seen[key] {
			stale = append(stale, key)
		}
	}
	sort.Strings(stale)
	assert.Empty(t, stale, "allowlisted text matches that are no longer in the code: remove them from textMatchAllowed")
}

// TestErrorTextMatches_Detects holds what the guard sees and what it leaves
// alone.
func TestErrorTextMatches_Detects(t *testing.T) {
	for _, tc := range []struct {
		name string
		body string
		want bool
	}{
		{"predicate on Error()", `return strings.Contains(err.Error(), "x")`, true},
		{"predicate on a variable holding the text", `msg := err.Error(); return strings.HasPrefix(msg, "x")`, true},
		{"predicate on a text derived from it", `msg := err.Error(); text := strings.TrimSpace(msg); return strings.Contains(text, "x")`, true},
		{"regexp on the text", `return re.MatchString(err.Error())`, true},
		{"comparison of the text", `return err.Error() == "x"`, true},
		{"switch over the text", `switch err.Error() { case "x": return true }; return false`, true},
		{"CUE message accessor", `format, _ := ce.Msg(); return strings.Contains(format, "x")`, true},
		{"predicate on a lower-cased text", `return strings.Contains(strings.ToLower(err.Error()), "x")`, true},
		{"predicate on a lower-cased variable", `msg := err.Error(); return strings.HasSuffix(strings.TrimSpace(msg), "x")`, true},
		{"switch over a trimmed text", `switch strings.TrimSpace(err.Error()) { case "x": return true }; return false`, true},
		{"typed check", `return errors.Is(err, errX)`, false},
		// Known limits: the guard reads syntax, not types.
		{"limit: a formatted error is not seen", `msg := fmt.Sprint(err); return strings.Contains(msg, "x")`, false},
		{"limit: a text handed to a helper is not followed", `return helper(err.Error())`, false},
		{"predicate on another string", `return strings.HasPrefix(name, "x")`, false},
		{"text only printed", `msg := err.Error(); fmt.Println(msg); return false`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			src := "package p\n\nfunc f(err error, name string) bool {\n" + tc.body + "\n}\n"
			got := errorTextMatchesIn(t, "p.go", []byte(src))
			assert.Equal(t, tc.want, len(got) > 0, "%v", got)
		})
	}
}
