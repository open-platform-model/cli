package cueedit

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"cuelang.org/go/cue/ast"
	"cuelang.org/go/cue/literal"
	"cuelang.org/go/cue/parser"
)

// RewriteSelfImports rewrites every import of the module at oldPath — the
// module root and any subpackage beneath it — to newPath, across every .cue
// file under dir (cue.mod excluded). Both paths carry their major suffix
// (e.g. "opmodel.dev/templates/standard@v1"); import paths usually spell the
// self-import without one (`import id ".../standard/identity"`), and a path
// that does carry the old major has it replaced with the new.
//
// This is the wholesale rewrite that is correct exactly once: on a freshly
// cloned tree the user does not own yet (init-from-template). Repair mode
// refuses it for the same reason this applies it.
//
// Splice-style like every writer here: only the import path literals change;
// every other byte — aliases, comments, grouping — stays as committed.
// Returns the dir-relative paths of the files it rewrote.
func RewriteSelfImports(dir, oldPath, newPath string) (changed []string, err error) {
	oldBase, oldMajor := splitMajor(oldPath)
	newBase, newMajor := splitMajor(newPath)
	if oldBase == "" || newBase == "" {
		return nil, errors.New("module paths must not be empty")
	}

	return rewriteTree(dir, func(path string, data []byte, f *ast.File) ([]byte, error) {
		type edit struct {
			start, end int
			repl       string
		}
		var edits []edit
		for _, decl := range f.Decls {
			imp, ok := decl.(*ast.ImportDecl)
			if !ok {
				continue
			}
			for _, spec := range imp.Specs {
				ipath, err := literal.Unquote(spec.Path.Value)
				if err != nil {
					return nil, fmt.Errorf("%s: import path %s is not a string: %w", path, spec.Path.Value, err)
				}
				rewritten, ok := rewriteImportPath(ipath, oldBase, oldMajor, newBase, newMajor)
				if !ok {
					continue
				}
				start, end, err := span(spec.Path)
				if err != nil {
					return nil, fmt.Errorf("%s: %w", path, err)
				}
				edits = append(edits, edit{start, end, strconv.Quote(rewritten)})
			}
		}
		if len(edits) == 0 {
			return data, nil
		}
		// Apply back-to-front so earlier offsets stay valid.
		sort.Slice(edits, func(i, j int) bool { return edits[i].start > edits[j].start })
		out := data
		for _, e := range edits {
			out = splice(out, e.start, e.end, e.repl)
		}
		return out, nil
	})
}

// RenamePackageClauses renames every `package oldName` clause under dir
// (cue.mod excluded) to newName, leaving every other byte — including
// comments and whitespace that follow the clause — untouched. Subpackages
// with their own names (identity/) keep them: only clauses binding oldName
// are renamed. Returns the dir-relative paths of the files it rewrote.
func RenamePackageClauses(dir, oldName, newName string) (changed []string, err error) {
	if oldName == "" || newName == "" {
		return nil, errors.New("package names must not be empty")
	}

	return rewriteTree(dir, func(path string, data []byte, f *ast.File) ([]byte, error) {
		for _, decl := range f.Decls {
			pkg, ok := decl.(*ast.Package)
			if !ok {
				continue
			}
			if pkg.Name == nil || pkg.Name.Name != oldName {
				return data, nil
			}
			start, end, err := span(pkg.Name)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", path, err)
			}
			return splice(data, start, end, newName), nil
		}
		return data, nil
	})
}

// rewriteTree walks every .cue file under dir (skipping cue.mod), hands each
// parsed file to rewrite, and verify-and-writes the ones whose bytes changed.
// The walk order is deterministic (WalkDir is lexical), so changed is too.
func rewriteTree(dir string, rewrite func(path string, data []byte, f *ast.File) ([]byte, error)) (changed []string, err error) {
	err = filepath.WalkDir(dir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			if d.Name() == "cue.mod" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(d.Name(), ".cue") {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("reading %s: %w", path, err)
		}
		f, err := parser.ParseFile(path, data, parser.ParseComments)
		if err != nil {
			return fmt.Errorf("%s does not parse: %w", path, err)
		}
		edited, err := rewrite(path, data, f)
		if err != nil {
			return err
		}
		if bytes.Equal(edited, data) {
			return nil
		}
		if err := verifyAndWrite(path, edited); err != nil {
			return err
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			rel = path
		}
		changed = append(changed, rel)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return changed, nil
}

// rewriteImportPath maps one import path from the old module to the new. An
// import path is `<path>[@major][:pkg]`; the qualifier survives verbatim, a
// major equal to the old module's follows the new module's, and any other
// major (unusual, but expressible) is preserved.
func rewriteImportPath(ipath, oldBase, oldMajor, newBase, newMajor string) (string, bool) {
	rest := ipath
	qual := ""
	if i := strings.LastIndex(rest, ":"); i >= 0 {
		qual = rest[i:]
		rest = rest[:i]
	}
	major := ""
	if i := strings.LastIndex(rest, "@"); i >= 0 {
		major = rest[i+1:]
		rest = rest[:i]
	}

	var out string
	switch {
	case rest == oldBase:
		out = newBase
	case strings.HasPrefix(rest, oldBase+"/"):
		out = newBase + strings.TrimPrefix(rest, oldBase)
	default:
		return "", false
	}

	if major != "" {
		if major == oldMajor {
			major = newMajor
		}
		out += "@" + major
	}
	return out + qual, true
}

// splitMajor splits a module path into its base and major suffix
// ("opmodel.dev/templates/standard@v1" → base, "v1"). A path without a major
// returns an empty major.
func splitMajor(modulePath string) (base, major string) {
	if before, after, ok := strings.Cut(modulePath, "@"); ok {
		return before, after
	}
	return modulePath, ""
}
