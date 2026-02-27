package builder

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"cuelang.org/go/cue"
	cueerrors "cuelang.org/go/cue/errors"
	"cuelang.org/go/cue/token"

	opmerrors "github.com/opmodel/cli/internal/errors"
)

// ValidateValues loads, validates, and unifies the given values files against
// the #config schema. It is the single entry point for all values handling in
// the build phase.
//
// The function:
//  1. Loads and compiles each file (preserving source positions via cue.Filename)
//  2. Extracts the top-level "values:" field from each file
//  3. Validates each file individually against config (type, constraint, closedness)
//  4. Unifies all files and catches cross-file conflicts
//
// Returns the unified values cue.Value on success.
// Returns *opmerrors.ValuesValidationError containing all errors on failure —
// never stops at the first error.
// Returns an error (not ValuesValidationError) for I/O or compile failures.
//
// When config is absent (zero cue.Value), schema validation is skipped and
// the files are only loaded and unified.
func ValidateValues(ctx *cue.Context, config cue.Value, filePaths []string) (cue.Value, error) {
	type loaded struct {
		path   string
		values cue.Value
	}

	files := make([]loaded, 0, len(filePaths))
	for _, p := range filePaths {
		v, err := loadValuesFile(ctx, p)
		if err != nil {
			return cue.Value{}, err
		}
		values := v.LookupPath(cue.ParsePath("values"))
		if !values.Exists() {
			return cue.Value{}, &opmerrors.ValidationError{
				Message: fmt.Sprintf("%s: no top-level 'values:' field found — each file must define 'values: { ... }'", filepath.Base(p)),
			}
		}
		files = append(files, loaded{path: p, values: values})
	}

	// Per-file schema validation. Collects all errors across all files before
	// returning, so the user sees the full picture at once.
	var fieldErrors []opmerrors.FieldError
	if config.Exists() {
		for _, f := range files {
			combined := validateFieldsRecursive(config, f.values, []string{"values"}, nil)
			if combined != nil {
				fieldErrors = append(fieldErrors, extractFieldErrors(combined)...)
			}
		}
	}
	fieldErrors = collapseFieldErrors(fieldErrors)

	// Unify all files into a single value. Only performed when more than one
	// file is provided; a single-file path skips the unification step entirely.
	var conflictErrors []opmerrors.ConflictError
	unified := files[0].values
	for _, f := range files[1:] {
		unified = unified.Unify(f.values)
		if err := unified.Err(); err != nil {
			conflictErrors = append(conflictErrors, extractConflictErrors(err)...)
		}
	}

	if len(fieldErrors) > 0 || len(conflictErrors) > 0 {
		return cue.Value{}, &opmerrors.ValuesValidationError{
			Errors:    fieldErrors,
			Conflicts: conflictErrors,
		}
	}

	return unified, nil
}

// extractFieldErrors converts a cueerrors.Error chain (from validateFieldsRecursive)
// into a slice of FieldError, pulling source positions from each error.
// Messages are derived from Msg() rather than Error() to avoid the schema path
// prefix that CUE injects into Error() output (e.g. "#config.port: ...").
func extractFieldErrors(err cueerrors.Error) []opmerrors.FieldError {
	var out []opmerrors.FieldError
	for _, e := range cueerrors.Errors(err) {
		pos := e.Position()
		file := ""
		if pos.IsValid() {
			file = filepath.Base(pos.Filename())
		}
		out = append(out, opmerrors.FieldError{
			File:    file,
			Line:    pos.Line(),
			Column:  pos.Column(),
			Path:    strings.Join(e.Path(), "."),
			Message: msgString(e),
		})
	}
	return out
}

// collapseFieldErrors deduplicates disjunction noise from CUE validation.
// CUE emits a triplet for type mismatches: a summary ("N errors in empty
// disjunction") followed by one entry per branch. We drop the summary and
// keep only the first branch per (File, Line, Column, Path) group, which is
// the most concrete conflict description.
func collapseFieldErrors(errs []opmerrors.FieldError) []opmerrors.FieldError {
	if len(errs) == 0 {
		return errs
	}

	// First pass: drop "N errors in empty disjunction" summary lines.
	filtered := errs[:0:len(errs)]
	for _, fe := range errs {
		if strings.Contains(fe.Message, "errors in empty disjunction") {
			continue
		}
		filtered = append(filtered, fe)
	}

	// Second pass: deduplicate by (File, Line, Column, Path), keeping first.
	type key struct {
		file, path string
		line, col  int
	}
	seen := make(map[key]bool, len(filtered))
	out := filtered[:0:len(filtered)]
	for _, fe := range filtered {
		k := key{fe.File, fe.Path, fe.Line, fe.Column}
		if seen[k] {
			continue
		}
		seen[k] = true
		out = append(out, fe)
	}
	return out
}

// extractConflictErrors converts a CUE unification conflict error into
// ConflictErrors. CUE's InputPositions() returns positions for both sides of
// a conflict; we group all positions for the same path into a single
// ConflictError so the user sees both sides together.
func extractConflictErrors(err error) []opmerrors.ConflictError {
	var out []opmerrors.ConflictError
	for _, e := range cueerrors.Errors(err) {
		path := strings.Join(e.Path(), ".")

		// Skip disjunction summary lines — they add noise without location info.
		msg := msgString(e)
		if strings.Contains(msg, "errors in empty disjunction") {
			continue
		}

		// Prefer InputPositions (both sides of a conflict) over Position.
		positions := e.InputPositions()
		if len(positions) == 0 {
			positions = []token.Pos{e.Position()}
		}

		seen := make(map[string]bool)
		var locs []opmerrors.ConflictLocation
		for _, pos := range positions {
			if !pos.IsValid() {
				continue
			}
			// Deduplicate: same file+line should not produce two entries.
			key := fmt.Sprintf("%s:%d", pos.Filename(), pos.Line())
			if seen[key] {
				continue
			}
			seen[key] = true
			locs = append(locs, opmerrors.ConflictLocation{
				File:   filepath.Base(pos.Filename()),
				Line:   pos.Line(),
				Column: pos.Column(),
			})
		}

		out = append(out, opmerrors.ConflictError{
			Path:      path,
			Message:   msg,
			Locations: locs,
		})
	}
	return out
}

// msgString formats a cueerrors.Error message using Msg() to avoid the schema
// path prefix that CUE injects into Error() output (e.g. "#config.port: ...").
func msgString(e cueerrors.Error) string {
	format, args := e.Msg()
	if len(args) == 0 {
		return format
	}
	return fmt.Sprintf(format, args...)
}

// loadValuesFile reads and compiles a single CUE values file.
// The absolute path is embedded via cue.Filename so that all cue.Value
// positions point back to the source file through unification.
func loadValuesFile(ctx *cue.Context, path string) (cue.Value, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return cue.Value{}, fmt.Errorf("resolving path: %w", err)
	}

	if _, err := os.Stat(absPath); os.IsNotExist(err) { //nolint:gosec // absPath resolved from caller-provided paths validated upstream
		return cue.Value{}, fmt.Errorf("values file not found: %s", absPath)
	}

	content, err := os.ReadFile(absPath) //nolint:gosec // absPath resolved from caller-provided paths validated upstream
	if err != nil {
		return cue.Value{}, fmt.Errorf("reading values file %s: %w", absPath, err)
	}

	v := ctx.CompileBytes(content, cue.Filename(absPath))
	if v.Err() != nil {
		return cue.Value{}, fmt.Errorf("compiling values file %s: %w", absPath, v.Err())
	}

	return v, nil
}

// validateFieldsRecursive walks every field in data recursively and validates
// each against the corresponding schema node.
func validateFieldsRecursive(schema, data cue.Value, path []string, errs cueerrors.Error) cueerrors.Error {
	iter, err := data.Fields()
	if err != nil {
		return errs
	}

	for iter.Next() {
		sel := iter.Selector()
		fieldVal := iter.Value()
		fieldName := sel.Unquoted()

		fieldPath := make([]string, len(path), len(path)+1)
		copy(fieldPath, path)
		fieldPath = append(fieldPath, sel.String())

		// Phase 1: Closedness check
		if !schema.Allows(cue.Str(fieldName)) {
			pos := findSourcePosition(fieldVal)
			errs = cueerrors.Append(errs, &pathRewrittenError{
				inner:   cueerrors.Newf(pos, "field not allowed"),
				newPath: fieldPath,
			})
			continue
		}

		// Phase 2: Resolve the schema field
		schemaField := schema.LookupPath(cue.MakePath(sel))
		if !schemaField.Exists() {
			schemaField = schema.LookupPath(cue.MakePath(cue.Str(fieldName).Optional()))
		}
		if !schemaField.Exists() {
			continue
		}

		// Phase 3: Recurse into struct children
		if fieldVal.IncompleteKind() == cue.StructKind {
			errs = validateFieldsRecursive(schemaField, fieldVal, fieldPath, errs)
			continue
		}

		// Phase 4: Type validation for leaf values.
		// Use findSourcePosition to anchor errors to the values file position,
		// not the schema side — CUE's unification errors often point at schema
		// nodes, which is unhelpful when the user needs to know which line in
		// their values file is wrong.
		pos := findSourcePosition(fieldVal)
		unified := schemaField.Unify(fieldVal)
		if fieldErr := unified.Validate(); fieldErr != nil {
			for _, e := range cueerrors.Errors(fieldErr) {
				errs = cueerrors.Append(errs, &pathRewrittenError{
					inner:       e,
					newPath:     fieldPath,
					posOverride: pos,
				})
			}
		}
	}
	return errs
}

// findSourcePosition extracts a source file position from a CUE value.
func findSourcePosition(v cue.Value) token.Pos {
	if pos := v.Pos(); pos.IsValid() {
		return pos
	}

	op, parts := v.Expr()
	if op == cue.AndOp {
		for _, part := range parts {
			if pos := part.Pos(); pos.IsValid() {
				return pos
			}
		}
	}

	return token.NoPos
}

// pathRewrittenError wraps a cueerrors.Error and overrides Path() and
// optionally Position() to anchor errors to values-file source locations.
type pathRewrittenError struct {
	inner       cueerrors.Error
	newPath     []string
	posOverride token.Pos // when valid, replaces inner.Position()
}

func (e *pathRewrittenError) Error() string {
	return e.inner.Error()
}

func (e *pathRewrittenError) Position() token.Pos {
	if e.posOverride.IsValid() {
		return e.posOverride
	}
	return e.inner.Position()
}

func (e *pathRewrittenError) InputPositions() []token.Pos {
	return e.inner.InputPositions()
}

func (e *pathRewrittenError) Path() []string {
	return e.newPath
}

func (e *pathRewrittenError) Msg() (format string, args []interface{}) {
	return e.inner.Msg()
}
