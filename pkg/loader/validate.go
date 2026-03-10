package loader

import (
	"cuelang.org/go/cue"
	cueerrors "cuelang.org/go/cue/errors"
	"cuelang.org/go/cue/token"

	oerrors "github.com/opmodel/cli/pkg/errors"
)

// ValidateConfig is the shared implementation for the Bundle Gate and Module Gate.
//
// It performs two passes to collect all validation errors:
//
//  1. walkDisallowed: walks the values tree recursively, calling schema.Allows()
//     at each level. Fields not allowed by the closed schema are collected as
//     "field not allowed" errors with source positions from the values value.
//     This is necessary because schema.Unify(values) does not reliably produce
//     "field not allowed" for file-loaded schemas — the close info is scoped to
//     the source package's evaluation context and does not transfer cleanly to
//     cross-package unification.
//
//  2. Unify + Validate(Concrete(true)): unifies values into the schema and walks
//     the entire arc tree (AllErrors hardcoded true internally), collecting type
//     conflicts, constraint violations, and missing required fields. "Field not
//     allowed" arcs silently become concrete string/int arcs in the unified
//     value (no bottom), so they are NOT double-counted from pass 1.
//
// schema is the #config CUE value (from #bundle.#config or #module.#config).
// values is the consumer-supplied or bundle-wired values CUE value.
// context is "bundle" or "module" (used in error output).
// name is the release/bundle name (used in error output).
//
// Returns *oerrors.ConfigError if validation fails, nil if values satisfy the schema.
func ValidateConfig(schema, values cue.Value, context, name string) *oerrors.ConfigError {
	// Check that both values are accessible before attempting unification.
	if !schema.Exists() || !values.Exists() {
		// Either the #config or values field is missing — not a user config error,
		// the caller is responsible for ensuring valid inputs. Return nil and let
		// downstream checks catch structural problems.
		return nil
	}

	var combined cueerrors.Error

	// Pass 1: detect fields not allowed by the closed schema.
	//
	// schema.Unify(values) does not produce "field not allowed" errors for schemas
	// loaded via BuildInstance — extra fields from an open values struct are added
	// to the unified vertex without a bottom value, because the close info is
	// scoped to the original package evaluation context. schema.Allows() is the
	// reliable API: it calls Vertex.Accept() which checks closedness correctly
	// regardless of how the schema was constructed.
	combined = walkDisallowed(schema, values, nil, combined)

	// Pass 2: unify and validate to catch type/constraint/concrete errors.
	//
	// Validate(Concrete(true)) hardcodes AllErrors:true internally, so it walks
	// the entire arc tree without breaking early on sibling errors. It collects:
	//   - conflicting values  (EvalError, e.g. string where struct expected)
	//   - constraint violations (e.g. value outside enum)
	//   - missing required fields (IncompleteError, gated by Concrete:true)
	//
	// For BuildInstance-loaded schemas, "field not allowed" arcs appear as plain
	// string/int arcs in the unified value — no bottom — so Validate doesn't
	// produce them. For CompileString schemas, Unify DOES produce "field not
	// allowed" — we skip those here because pass 1 is the authoritative source
	// for that error category (with correct positions from the values file).
	unified := schema.Unify(values)
	if err := unified.Validate(cue.Concrete(true)); err != nil {
		for _, ce := range cueerrors.Errors(err) {
			f, _ := ce.Msg()
			if f == "field not allowed" {
				continue // pass 1 owns this category; skip to avoid duplicates
			}
			combined = cueerrors.Append(combined, ce)
		}
	}

	if combined == nil {
		return nil
	}
	return &oerrors.ConfigError{
		Context:  context,
		Name:     name,
		RawError: combined,
	}
}

// walkDisallowed walks val's fields and uses schema.Allows() to detect fields
// not permitted by the closed schema. For each disallowed field a
// "field not allowed" cueerrors.Error is appended to acc. The walk recurses
// into struct-typed fields that ARE allowed, passing the corresponding
// sub-schema so nested closedness constraints are checked too.
//
// pathPrefix is the dot-joined path of ancestor selectors, used to populate
// the Path() on each emitted error (e.g. "media.extra"). Pass nil initially.
//
// Pattern constraints in the schema ([Name=string]: {...}) cause Allows() to
// return true for any string key, so the walk recurses into them using the
// pattern's value schema — only the field's value type is then checked by
// pass 2 (Unify+Validate).
func walkDisallowed(schema, val cue.Value, pathPrefix []string, acc cueerrors.Error) cueerrors.Error {
	iter, err := val.Fields(cue.Optional(true))
	if err != nil {
		return acc
	}
	for iter.Next() {
		sel := iter.Selector()
		child := iter.Value()
		fieldPath := append(pathPrefix, sel.String()) //nolint:gocritic // new slice per loop

		if !schema.Allows(sel) {
			acc = cueerrors.Append(acc, &fieldNotAllowedError{
				pos:  child.Pos(),
				path: fieldPath,
			})
			continue // do not recurse into a disallowed subtree
		}

		// Recurse into struct-typed fields using the sub-schema at that selector.
		// If the sub-schema does not exist (e.g. a pattern-constrained field
		// where the concrete key has no direct arc in the schema), skip
		// recursion — pass 2 (Unify+Validate) handles value-type errors there.
		if child.IncompleteKind() == cue.StructKind {
			childSchema := schema.LookupPath(cue.MakePath(sel))
			if !childSchema.Exists() {
				// Pattern-constrained field: schema accepts the key but has no
				// direct arc for it (the constraint is a pattern, not a named
				// field). Value type checking is done by Unify+Validate.
				continue
			}
			acc = walkDisallowed(childSchema, child, fieldPath, acc)
		}
	}
	return acc
}

// fieldNotAllowedError is a minimal cueerrors.Error implementation used to
// represent a "field not allowed" violation detected by walkDisallowed.
// It carries the source position of the disallowed value (from cue.Value.Pos())
// and the dot-joined path from the walk root, so the display layer can show
// file:line:col -> path.
type fieldNotAllowedError struct {
	pos  token.Pos
	path []string // selector path from the walk root, e.g. ["media", "extra"]
}

func (e *fieldNotAllowedError) Position() token.Pos         { return e.pos }
func (e *fieldNotAllowedError) InputPositions() []token.Pos { return nil }
func (e *fieldNotAllowedError) Error() string               { return "field not allowed" }
func (e *fieldNotAllowedError) Path() []string              { return e.path }
func (e *fieldNotAllowedError) Msg() (string, []interface{}) {
	return "field not allowed", nil
}
