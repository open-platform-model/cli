package loader

import (
	"strings"
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	cueerrors "cuelang.org/go/cue/errors"
	"cuelang.org/go/cue/load"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/opmodel/cli/internal/releaseprocess"
)

// TestValidateConfig_FieldNotAllowed_FileLoaded exercises the two-pass
// validation path against the real jellyfin example. The values.cue file
// deliberately contains two extra top-level fields ("test", "invalidField")
// and one type-conflicting nested field ("media.test").
//
// The test asserts that all three violations are reported — verifying that
// walkDisallowed catches "field not allowed" errors that schema.Unify()
// silently misses for BuildInstance-loaded schemas (close info is scoped to
// the source package's evaluation context).
func TestValidateConfig_FieldNotAllowed_FileLoaded(t *testing.T) {
	ctx := cuecontext.New()

	// Load release.cue alone (mirrors LoadReleaseFile).
	releaseInsts := load.Instances([]string{"release.cue"}, &load.Config{
		Dir: "../../examples/releases/jellyfin",
	})
	require.Len(t, releaseInsts, 1)
	require.NoError(t, releaseInsts[0].Err)
	releaseVal := ctx.BuildInstance(releaseInsts[0])
	require.NoError(t, releaseVal.Err())

	// Load values.cue alone and extract the "values" field (mirrors LoadValuesFile).
	valuesInsts := load.Instances([]string{"values.cue"}, &load.Config{
		Dir: "../../examples/releases/jellyfin",
	})
	require.Len(t, valuesInsts, 1)
	require.NoError(t, valuesInsts[0].Err)
	valuesVal := ctx.BuildInstance(valuesInsts[0])
	require.NoError(t, valuesVal.Err())
	if vf := valuesVal.LookupPath(cue.ParsePath("values")); vf.Exists() {
		valuesVal = vf
	}

	// Extract the config schema (mirrors release file validation flow).
	configSchema := releaseVal.LookupPath(cue.ParsePath("#module.#config"))
	require.True(t, configSchema.Exists(), "#module.#config must exist in release")
	require.True(t, configSchema.IsClosed(), "#module.#config must be a closed struct")

	// Call ValidateConfig and assert the result.
	_, cfgErr := releaseprocess.ValidateConfig(configSchema, []cue.Value{valuesVal}, "module", "jellyfin")
	require.NotNil(t, cfgErr, "values with extra fields should produce a ConfigError")

	// Collect all individual CUE errors.
	errs := cueerrors.Errors(cfgErr.RawError)
	require.NotEmpty(t, errs)

	// Build a string of all error messages for assertion.
	msgs := make([]string, 0, len(errs))
	for _, ce := range errs {
		f, args := ce.Msg()
		_ = args
		msgs = append(msgs, f)
	}
	combined := strings.Join(msgs, "\n")

	// Assert the three expected violations.
	assert.Contains(t, combined, "field not allowed",
		"extra top-level fields (test, invalidField) should produce 'field not allowed'")
	assert.Contains(t, combined, "conflicting values",
		"media.test type conflict should produce 'conflicting values'")

	// Assert exactly three errors: test, invalidField (field not allowed x2)
	// + media.test (conflicting values x1).
	fieldNotAllowedCount := 0
	conflictingCount := 0
	for _, ce := range errs {
		f, _ := ce.Msg()
		switch {
		case f == "field not allowed":
			fieldNotAllowedCount++
		case strings.HasPrefix(f, "conflicting values"):
			conflictingCount++
		}
	}
	assert.Equal(t, 2, fieldNotAllowedCount, "expected 2 'field not allowed' errors (test + invalidField)")
	assert.Equal(t, 1, conflictingCount, "expected 1 'conflicting values' error (media.test)")

	// Assert positions are present (file:line:col) for "field not allowed" errors.
	for _, ce := range errs {
		f, _ := ce.Msg()
		if f != "field not allowed" {
			continue
		}
		positions := cueerrors.Positions(ce)
		assert.NotEmpty(t, positions, "field not allowed errors should have source positions")
		if len(positions) > 0 {
			assert.True(t, positions[0].IsValid(), "position should be valid")
			assert.Contains(t, positions[0].Filename(), "values.cue",
				"position should point to values.cue")
		}
	}
}

// TestValidateConfig_FieldNotAllowed_Inline verifies that the two-pass approach
// also works for inline (CompileString) schemas — specifically that walkDisallowed
// does not double-count errors that schema.Unify() already catches.
func TestValidateConfig_FieldNotAllowed_Inline(t *testing.T) {
	ctx := cuecontext.New()

	schema := ctx.CompileString(`
#config: {
	port: int
	name: string
}
`, cue.Filename("schema.cue"))
	require.NoError(t, schema.Err())

	configSchema := schema.LookupPath(cue.ParsePath("#config"))
	require.True(t, configSchema.Exists())
	require.True(t, configSchema.IsClosed())

	values := ctx.CompileString(`{
	port:       8080
	name:       "hello"
	extraField: "bad"
}`, cue.Filename("values.cue"))
	require.NoError(t, values.Err())

	_, cfgErr := releaseprocess.ValidateConfig(configSchema, []cue.Value{values}, "module", "test")
	require.NotNil(t, cfgErr, "extra field should produce a ConfigError")

	errs := cueerrors.Errors(cfgErr.RawError)
	fieldNotAllowedCount := 0
	for _, ce := range errs {
		f, _ := ce.Msg()
		if f == "field not allowed" {
			fieldNotAllowedCount++
		}
	}
	// Should be exactly 1 — not 2 even if Unify also catches it.
	assert.Equal(t, 1, fieldNotAllowedCount, "should report exactly one 'field not allowed' for extraField")
}
