// Package cueedit performs surgical, position-based rewrites of OPM authoring
// files. It implements enhancement 0011 D8's Version write: locate the field
// by its schema-fixed path in identity/identity.cue, splice the new value into
// the original bytes, and leave every other byte — comments, alignment, field
// order — exactly as committed. `opm ... publish --version` fills through this
// writer; `version set` reads and rewrites through it; the cue.mod `module:`
// writer ships here for cli-template-modules' repair and re-identification.
package cueedit

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"

	"cuelang.org/go/cue/ast"
	"cuelang.org/go/cue/literal"
	"cuelang.org/go/cue/parser"
	"cuelang.org/go/cue/token"
)

// ErrIdentityShape reports that identity/identity.cue does not match
// #IdentityPackage's shape at the point the writer needs it to: the file is
// absent, does not parse, or declares no Version field at the schema-fixed
// path. Callers branch with errors.Is; the wrapped message names the file and
// the specific mismatch.
var ErrIdentityShape = errors.New("identity file does not match #IdentityPackage's shape")

// ErrCueModShape reports that cue.mod/module.cue does not match the shape the
// module-path writer needs: the file is absent, does not parse, or declares
// no top-level module field carrying a string literal. Callers branch with
// errors.Is; the wrapped message names the file and the specific mismatch.
var ErrCueModShape = errors.New("cue.mod/module.cue does not match its expected shape")

// versionRE mirrors core.#VersionType (bare SemVer 2.0, no "v" prefix).
var versionRE = regexp.MustCompile(`^\d+\.\d+\.\d+(-[0-9A-Za-z-]+(\.[0-9A-Za-z-]+)*)?(\+[0-9A-Za-z-]+(\.[0-9A-Za-z-]+)*)?$`)

// CheckVersion reports whether version has core.#VersionType's shape: a bare
// SemVer 2.0 (MAJOR.MINOR.PATCH with optional pre-release and build metadata,
// no "v" prefix). Shared by the writers and by commands that validate the
// argument before touching any file.
func CheckVersion(version string) error {
	if !versionRE.MatchString(version) {
		return fmt.Errorf("version %q is not a bare SemVer (expected MAJOR.MINOR.PATCH, no \"v\" prefix)", version)
	}
	return nil
}

// SetIdentityVersion rewrites the Version field of dir/identity/identity.cue
// to the given bare SemVer, preserving all surrounding bytes:
//
//   - a string literal value ("1.2.0", or one inside an "&" chain) is replaced;
//   - a defaulted disjunction (#VersionType | *"1.2.0") has its default
//     replaced — the field stays a default, so release automation that owns the
//     committed value keeps owning it;
//   - an open field (no string literal anywhere in the value) gains a
//     unification with the version: `#VersionType` becomes
//     `#VersionType & "1.2.0"`.
//
// Setting the version the file already declares splices to identical bytes:
// the file is not written — no mtime change, nothing for a pre-commit hook to
// react to (D3's idempotent no-op) — and changed is false.
//
// An absent file, an unparseable file, or a missing Version field is a shape
// refusal (ErrIdentityShape) — a malformed artifact, not an unfinished one.
func SetIdentityVersion(dir, version string) (changed bool, err error) {
	if err := CheckVersion(version); err != nil {
		return false, err
	}

	path, data, field, err := loadIdentity(dir)
	if err != nil {
		return false, err
	}

	edited, err := spliceVersion(data, field.Value, version)
	if err != nil {
		return false, fmt.Errorf("%w: %s: %w", ErrIdentityShape, path, err)
	}
	if bytes.Equal(edited, data) {
		return false, nil
	}
	if err := verifyAndWrite(path, edited); err != nil {
		return false, err
	}
	return true, nil
}

// ReadIdentityVersion reports the version dir/identity/identity.cue declares:
// the same string literal SetIdentityVersion would rewrite — the value
// itself, the literal in an "&" chain, or a disjunction's default arm (with
// defaulted true). An open field (no version-carrying literal) reads as ""
// with no error. Shape failures are ErrIdentityShape, exactly as
// SetIdentityVersion refuses them.
func ReadIdentityVersion(dir string) (value string, defaulted bool, err error) {
	path, _, field, err := loadIdentity(dir)
	if err != nil {
		return "", false, err
	}
	lit, defaulted := versionLiteral(field.Value)
	if lit == nil {
		return "", false, nil
	}
	value, err = literal.Unquote(lit.Value)
	if err != nil {
		return "", false, fmt.Errorf("%w: %s: Version literal %s is not a string: %w", ErrIdentityShape, path, lit.Value, err)
	}
	return value, defaulted, nil
}

// SetCueModModule rewrites the module field of dir/cue.mod/module.cue to the
// given module path (major suffix included), preserving all surrounding bytes
// — comments, field order, the language stanza. Declaring the path the file
// already declares writes nothing and returns changed false. The pipeline's
// own reader stays read-only over this file (D16); consumers of this writer
// land with cli-template-modules (repair and re-identification).
func SetCueModModule(dir, modulePath string) (changed bool, err error) {
	if modulePath == "" {
		return false, errors.New("module path must not be empty")
	}

	path := filepath.Join(dir, "cue.mod", "module.cue")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, fmt.Errorf("%w: %s does not exist", ErrCueModShape, path)
		}
		return false, fmt.Errorf("reading %s: %w", path, err)
	}

	f, err := parser.ParseFile(path, data, parser.ParseComments)
	if err != nil {
		return false, fmt.Errorf("%w: %s does not parse: %w", ErrCueModShape, path, err)
	}

	field := topLevelField(f, "module")
	if field == nil {
		return false, fmt.Errorf("%w: %s declares no top-level module field", ErrCueModShape, path)
	}
	lit, ok := field.Value.(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		return false, fmt.Errorf("%w: %s module field does not carry a string literal", ErrCueModShape, path)
	}

	start, end, err := span(lit)
	if err != nil {
		return false, fmt.Errorf("%w: %s: %w", ErrCueModShape, path, err)
	}
	edited := splice(data, start, end, strconv.Quote(modulePath))
	if bytes.Equal(edited, data) {
		return false, nil
	}
	if err := verifyAndWrite(path, edited); err != nil {
		return false, err
	}
	return true, nil
}

// verifyAndWrite is the shared tail of every writer: the splice is byte
// surgery, so prove the result still parses before it atomically replaces
// the committed file. Callers have already established the edit is a real
// change.
func verifyAndWrite(path string, edited []byte) error {
	if _, err := parser.ParseFile(path, edited); err != nil {
		return fmt.Errorf("internal error: rewritten %s does not parse: %w", path, err)
	}
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("stat %s: %w", path, err)
	}
	return writeFileAtomic(path, edited, info.Mode().Perm())
}

// loadIdentity reads and parses dir/identity/identity.cue and locates its
// top-level Version field, mapping each failure to the shape refusal shared
// by the read and write paths.
func loadIdentity(dir string) (path string, data []byte, field *ast.Field, err error) {
	path = filepath.Join(dir, "identity", "identity.cue")
	data, err = os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil, nil, fmt.Errorf("%w: %s does not exist", ErrIdentityShape, path)
		}
		return "", nil, nil, fmt.Errorf("reading %s: %w", path, err)
	}

	f, err := parser.ParseFile(path, data, parser.ParseComments)
	if err != nil {
		return "", nil, nil, fmt.Errorf("%w: %s does not parse: %w", ErrIdentityShape, path, err)
	}

	field = topLevelField(f, "Version")
	if field == nil {
		return "", nil, nil, fmt.Errorf("%w: %s declares no top-level Version field", ErrIdentityShape, path)
	}
	return path, data, field, nil
}

// writeFileAtomic replaces path's content via a same-directory temp file and
// rename, so a crash mid-write can never leave the committed file truncated —
// this rewrites a file the author did not touch themselves.
func writeFileAtomic(path string, data []byte, perm os.FileMode) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), ".cueedit-*.tmp")
	if err != nil {
		return fmt.Errorf("staging rewrite of %s: %w", path, err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName) // no-op after a successful rename

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("staging rewrite of %s: %w", path, err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("staging rewrite of %s: %w", path, err)
	}
	if err := os.Chmod(tmpName, perm); err != nil {
		return fmt.Errorf("staging rewrite of %s: %w", path, err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}
	return nil
}

// topLevelField returns the top-level field with the given identifier label,
// or nil.
func topLevelField(f *ast.File, name string) *ast.Field {
	for _, decl := range f.Decls {
		field, ok := decl.(*ast.Field)
		if !ok {
			continue
		}
		if ident, ok := field.Label.(*ast.Ident); ok && ident.Name == name {
			return field
		}
	}
	return nil
}

// spliceVersion returns data with the Version field's value expression edited
// to carry version, per the SetIdentityVersion contract.
func spliceVersion(data []byte, value ast.Expr, version string) ([]byte, error) {
	quoted := strconv.Quote(version)

	if lit, _ := versionLiteral(value); lit != nil {
		start, end, err := span(lit)
		if err != nil {
			return nil, err
		}
		return splice(data, start, end, quoted), nil
	}

	// Open field: unify the existing expression with the version. A top-level
	// disjunction needs parentheses so the "&" binds the whole expression.
	start, end, err := span(value)
	if err != nil {
		return nil, err
	}
	if bin, ok := value.(*ast.BinaryExpr); ok && bin.Op == token.OR {
		return splice(data, start, end, "("+string(data[start:end])+") & "+quoted), nil
	}
	return splice(data, start, end, string(data[start:end])+" & "+quoted), nil
}

// versionLiteral finds the string literal that carries the field's version:
// the value itself, an operand of an "&" chain, or the defaulted arm of a
// disjunction (preferred over non-default arms). defaulted reports whether
// the literal is a disjunction's default arm. Returns nil when the value
// holds no version-carrying literal — the open-field case.
func versionLiteral(expr ast.Expr) (lit *ast.BasicLit, defaulted bool) {
	switch e := expr.(type) {
	case *ast.BasicLit:
		if e.Kind == token.STRING {
			return e, false
		}
	case *ast.ParenExpr:
		return versionLiteral(e.X)
	case *ast.UnaryExpr:
		if e.Op == token.MUL {
			if lit, _ := versionLiteral(e.X); lit != nil {
				return lit, true
			}
		}
	case *ast.BinaryExpr:
		return binaryVersionLiteral(e)
	}
	return nil, false
}

// binaryVersionLiteral resolves versionLiteral over a binary expression: only
// "|" and "&" chains can hold the version literal — any other operator
// (comparison, arithmetic, pattern match) does not.
func binaryVersionLiteral(e *ast.BinaryExpr) (lit *ast.BasicLit, defaulted bool) {
	if e.Op != token.OR && e.Op != token.AND {
		return nil, false
	}
	if e.Op == token.OR {
		// Prefer the defaulted arm — that is the committed, release-owned
		// declaration.
		for _, side := range []ast.Expr{e.X, e.Y} {
			if u, ok := side.(*ast.UnaryExpr); ok && u.Op == token.MUL {
				if lit, _ := versionLiteral(u.X); lit != nil {
					return lit, true
				}
			}
		}
	}
	if lit, defaulted := versionLiteral(e.X); lit != nil {
		return lit, defaulted
	}
	return versionLiteral(e.Y)
}

// span returns the byte offsets [start, end) of an expression in its file.
func span(expr ast.Expr) (start, end int, err error) {
	startPos, endPos := expr.Pos(), expr.End()
	if !startPos.HasAbsPos() || !endPos.HasAbsPos() {
		return 0, 0, fmt.Errorf("expression carries no absolute position")
	}
	return startPos.Position().Offset, endPos.Position().Offset, nil
}

// splice returns data with [start, end) replaced by repl.
func splice(data []byte, start, end int, repl string) []byte {
	out := make([]byte, 0, len(data)-(end-start)+len(repl))
	out = append(out, data[:start]...)
	out = append(out, repl...)
	out = append(out, data[end:]...)
	return out
}
