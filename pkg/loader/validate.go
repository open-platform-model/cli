package loader

import (
	"cuelang.org/go/cue"

	oerrors "github.com/opmodel/cli/pkg/errors"
)

// ValidateConfig is the shared implementation for the Bundle Gate and Module Gate.
//
// It performs two checks:
//
//  1. Unify: unifies values with schema and checks for CUE errors.
//     Catches type mismatches and constraint violations (e.g. string where int
//     is expected, value outside an enum, matchN violations).
//
//  2. Concrete: validates that the unified value has no remaining open fields.
//     Catches missing required fields — fields in #config that have no default
//     and were not provided in values.
//
// schema is the #config CUE value (from #bundle.#config or #module.#config).
// values is the consumer-supplied or bundle-wired values CUE value.
// context is "bundle" or "module" (used in error output).
// name is the release/bundle name (used in error output).
//
// Returns *oerrors.ConfigError if validation fails, nil if values satisfy the schema.
//
// Design note: this function performs its own unification rather than relying
// on errors that propagated through the CUE comprehension. This gives us a
// clean, isolated error rooted at the values/schema boundary — not a deeply
// nested error from the comprehension output that is hard to attribute.
func ValidateConfig(schema, values cue.Value, context, name string) *oerrors.ConfigError {
	// Check that both values are accessible before attempting unification.
	if !schema.Exists() || !values.Exists() {
		// Either the #config or values field is missing — not a user config error,
		// the caller is responsible for ensuring valid inputs. Return nil and let
		// downstream checks catch structural problems.
		return nil
	}

	// Unify: merge values into the schema. CUE unification is the same operation
	// that #BundleRelease / #ModuleRelease perform internally — we do it here
	// explicitly so we can intercept the error at the gate boundary.
	unified := schema.Unify(values)
	if err := unified.Err(); err != nil {
		return &oerrors.ConfigError{
			Context:  context,
			Name:     name,
			RawError: err,
		}
	}

	// Concrete: verify all fields have concrete values (no open constraints remain).
	// This catches required fields (fields without defaults) that were not supplied.
	if err := unified.Validate(cue.Concrete(true)); err != nil {
		return &oerrors.ConfigError{
			Context:  context,
			Name:     name,
			RawError: err,
		}
	}

	return nil
}
