package loader

import (
	"fmt"
	"strings"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/errors"
)

// ConfigError is a structured validation error produced by the Bundle Gate or
// Module Gate when supplied values do not satisfy a #config schema.
//
// It is designed as the foundation for a future per-field diagnostic system:
// RawError carries the original CUE error tree, which can be walked to extract
// individual field paths and constraint violations for rich user-facing output.
//
// Current output is a formatted summary with one line per CUE error position.
// Future: FieldErrors []FieldError populated by parsing RawError.
type ConfigError struct {
	// Context is "bundle" or "module" — identifies which gate produced the error.
	Context string

	// Name is the release/bundle name for display (e.g. "my-game-stack", "server").
	Name string

	// RawError is the original CUE unification or concreteness error.
	// Preserved for future parsing — a diagnostic layer can walk cue/errors.Details
	// to extract per-field paths and constraint messages.
	RawError error
}

// Error implements the error interface.
// Produces a human-readable summary: one line per unique CUE error position.
func (e *ConfigError) Error() string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "%s %q: values do not satisfy #config:\n", e.Context, e.Name)

	// Walk the CUE error list for per-position messages.
	// errors.Errors unwraps a combined CUE error into its constituent list.
	for _, ce := range errors.Errors(e.RawError) {
		pos := ce.Position()
		msg := errors.Details(ce, nil)
		if pos.IsValid() {
			fmt.Fprintf(&sb, "  - %s: %s\n", pos, strings.TrimSpace(msg))
		} else {
			fmt.Fprintf(&sb, "  - %s\n", strings.TrimSpace(msg))
		}
	}
	return strings.TrimRight(sb.String(), "\n")
}

// Unwrap returns the underlying CUE error for errors.Is/As compatibility.
func (e *ConfigError) Unwrap() error { return e.RawError }

// validateConfig is the shared implementation for the Bundle Gate and Module Gate.
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
// Returns *ConfigError if validation fails, nil if values satisfy the schema.
//
// Design note: this function performs its own unification rather than relying
// on errors that propagated through the CUE comprehension. This gives us a
// clean, isolated error rooted at the values/schema boundary — not a deeply
// nested error from the comprehension output that is hard to attribute.
func validateConfig(schema cue.Value, values cue.Value, context string, name string) *ConfigError {
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
		return &ConfigError{
			Context:  context,
			Name:     name,
			RawError: err,
		}
	}

	// Concrete: verify all fields have concrete values (no open constraints remain).
	// This catches required fields (fields without defaults) that were not supplied.
	if err := unified.Validate(cue.Concrete(true)); err != nil {
		return &ConfigError{
			Context:  context,
			Name:     name,
			RawError: err,
		}
	}

	return nil
}
