package release

import (
	"cuelang.org/go/cue"
	cueerrors "cuelang.org/go/cue/errors"
	"cuelang.org/go/cue/token"
)

// collectAllCUEErrors runs Validate() on the full value tree.
// Returns a combined error containing all discovered errors, or nil if clean.
func collectAllCUEErrors(v cue.Value) error {
	return v.Validate()
}

// validateValuesAgainstConfig validates user-provided values against the #config
// definition by recursively walking every field and checking each against the
// corresponding schema node.
func validateValuesAgainstConfig(configDef, valuesVal cue.Value) error {
	combined := validateFieldsRecursive(configDef, valuesVal, []string{"values"}, nil)
	if combined == nil {
		return nil
	}
	return combined
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
			fieldNotAllowed := cueerrors.Newf(pos, "field not allowed")
			errs = cueerrors.Append(errs, &pathRewrittenError{
				inner:   fieldNotAllowed,
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

		// Phase 4: Type validation for leaf values
		unified := schemaField.Unify(fieldVal)
		if fieldErr := unified.Validate(); fieldErr != nil {
			for _, e := range cueerrors.Errors(fieldErr) {
				errs = cueerrors.Append(errs, &pathRewrittenError{
					inner:   e,
					newPath: fieldPath,
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

// pathRewrittenError wraps a cueerrors.Error and overrides Path() to return
// a custom path.
type pathRewrittenError struct {
	inner   cueerrors.Error
	newPath []string
}

func (e *pathRewrittenError) Error() string {
	return e.inner.Error()
}

func (e *pathRewrittenError) Position() token.Pos {
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

// rewriteErrorPath wraps a CUE error with a new path.
func rewriteErrorPath(e cueerrors.Error, basePath []string) cueerrors.Error {
	errPath := e.Path()
	newPath := make([]string, 0, len(basePath)+len(errPath))
	newPath = append(newPath, basePath...)
	newPath = append(newPath, errPath...)

	return &pathRewrittenError{
		inner:   e,
		newPath: newPath,
	}
}
