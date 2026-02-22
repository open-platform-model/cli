package valuesflow

// ---------------------------------------------------------------------------
// Values selection: Layer 1 (defaults) and Layer 2 (user override)
//
// selectValues() is the single decision point for which values are used in
// a build. It mirrors what builder.Build() / selectValues() will do:
//
//   Layer 2 (highest): user-provided --values file → completely replaces defaults
//   Layer 1 (default): separate values.cue → used when no user file provided
//   Fallback:          inline values in module.cue → used when no values.cue exists
//
// The critical invariant: when user values are provided, module defaults are
// entirely ignored — they do not bleed through.
//
// Fixture: testdata/values_module/ (separate values.cue)
// Overrides: testdata/user_values.cue
// ---------------------------------------------------------------------------

import (
	"testing"

	"cuelang.org/go/cue"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSelectValues_DefaultsUsedWhenNoUserValues proves that when no --values
// file is provided, selectValues returns the module defaults from values.cue
// (Layer 1).
func TestSelectValues_DefaultsUsedWhenNoUserValues(t *testing.T) {
	ctx, _ := loadCatalog(t)
	moduleVal, defaultVals := loadModuleApproachA(t, ctx, fixturePath(t, "values_module"))

	selected, ok := selectValues(t, ctx, moduleVal, defaultVals, "" /* no user file */)

	require.True(t, ok, "selectValues must succeed when defaults are present")

	image, err := selected.LookupPath(cue.ParsePath("image")).String()
	require.NoError(t, err)
	assert.Equal(t, "nginx:latest", image,
		"Layer 1: defaults from values.cue must be used when no user values provided")

	replicas, err := selected.LookupPath(cue.ParsePath("replicas")).Int64()
	require.NoError(t, err)
	assert.Equal(t, int64(1), replicas)
}

// TestSelectValues_UserValuesReplaceDefaults proves that when a --values file
// is provided, it completely replaces the module defaults (Layer 2 wins).
func TestSelectValues_UserValuesReplaceDefaults(t *testing.T) {
	ctx, _ := loadCatalog(t)
	moduleVal, defaultVals := loadModuleApproachA(t, ctx, fixturePath(t, "values_module"))
	userFile := fixturePath(t, "user_values.cue")

	selected, ok := selectValues(t, ctx, moduleVal, defaultVals, userFile)

	require.True(t, ok)

	image, err := selected.LookupPath(cue.ParsePath("image")).String()
	require.NoError(t, err)
	assert.Equal(t, "custom:2.0", image,
		"Layer 2: user values must win over module defaults")

	replicas, err := selected.LookupPath(cue.ParsePath("replicas")).Int64()
	require.NoError(t, err)
	assert.Equal(t, int64(5), replicas,
		"Layer 2: user values must win over module defaults")
}

// TestSelectValues_DefaultsNeverBleedThrough proves that when user values are
// provided, the module defaults are entirely ignored — no field from the defaults
// appears in the selected values unless the user explicitly provided it.
func TestSelectValues_DefaultsNeverBleedThrough(t *testing.T) {
	ctx, _ := loadCatalog(t)
	moduleVal, defaultVals := loadModuleApproachA(t, ctx, fixturePath(t, "values_module"))
	userFile := fixturePath(t, "user_values.cue")

	selected, ok := selectValues(t, ctx, moduleVal, defaultVals, userFile)

	require.True(t, ok)

	image, err := selected.LookupPath(cue.ParsePath("image")).String()
	require.NoError(t, err)
	assert.NotEqual(t, "nginx:latest", image,
		"default image 'nginx:latest' must NOT appear when user values are provided — Layer 2 replaces Layer 1 entirely")
}
