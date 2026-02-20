package valuesnapshot

// ---------------------------------------------------------------------------
// Core showcase: storing pre-evaluated and post-evaluated cue.Value together
//
// cue.Value is a value type — a small struct, not a pointer. Storing two
// cue.Value fields in a Go struct is free, safe, and the idiomatic approach.
// All CUE operations (FillPath, Unify, LookupPath) are pure: they return a
// new value without mutating the input.
//
// moduleSnapshot demonstrates this directly:
//
//   type moduleSnapshot struct {
//       Schema    cue.Value  // pre-fill: module as loaded, defaults visible
//       Evaluated cue.Value  // post-fill: user values applied
//   }
//
// These tests prove:
//   - Both fields can be populated from a single *cue.Context
//   - Schema carries the module author's defaults (image="nginx:latest")
//   - Evaluated carries the end-user overrides (image="nginx:1.28.2")
//   - The two values report different concrete results for the same path
//   - Both share the same *cue.Context — no cross-context panic risk
//
// Reference: docs/design/cue-in-go.md "Storing cue.Value" and
// "Divergent copies via operations"
// ---------------------------------------------------------------------------

import (
	"testing"

	"cuelang.org/go/cue"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSnapshot_BothFieldsPopulated proves that after buildSnapshot(), both
// moduleSnapshot fields are valid, existing, non-error cue.Value handles.
func TestSnapshot_BothFieldsPopulated(t *testing.T) {
	_, snap := buildSnapshot(t)

	assert.True(t, snap.Schema.Exists(), "Schema should exist")
	assert.NoError(t, snap.Schema.Err(), "Schema should not be errored")

	assert.True(t, snap.Evaluated.Exists(), "Evaluated should exist")
	assert.NoError(t, snap.Evaluated.Err(), "Evaluated should not be errored")
}

// TestSnapshot_SchemaHasModuleDefaults proves that Schema — the pre-fill value
// from BuildInstance — resolves component spec fields to the module author's
// defaults. No user values have been applied at this point.
//
// The test module defines:
//
//	#config: { image: string | *"nginx:latest", replicas: int & >=1 | *1 }
//
// CUE resolves these disjunctions to their defaults at evaluation time, so
// Schema is already concrete with author-defined values.
func TestSnapshot_SchemaHasModuleDefaults(t *testing.T) {
	_, snap := buildSnapshot(t)

	image, err := snap.Schema.LookupPath(cue.ParsePath("#components.web.spec.image")).String()
	require.NoError(t, err, "Schema.web.spec.image should be readable (module default)")
	assert.Equal(t, "nginx:latest", image, "Schema should carry module default image")

	replicas, err := snap.Schema.LookupPath(cue.ParsePath("#components.web.spec.replicas")).Int64()
	require.NoError(t, err, "Schema.web.spec.replicas should be readable (module default)")
	assert.Equal(t, int64(1), replicas, "Schema should carry module default replicas")
}

// TestSnapshot_EvaluatedHasUserValues proves that Evaluated — the post-fill
// value from FillPath — reflects the user-supplied overrides. The user values
// fixture sets image="nginx:1.28.2" and replicas=3, which replace the defaults.
func TestSnapshot_EvaluatedHasUserValues(t *testing.T) {
	_, snap := buildSnapshot(t)

	image, err := snap.Evaluated.LookupPath(cue.ParsePath("#components.web.spec.image")).String()
	require.NoError(t, err, "Evaluated.web.spec.image should be readable")
	assert.Equal(t, "nginx:1.28.2", image, "Evaluated should carry user-supplied image")

	replicas, err := snap.Evaluated.LookupPath(cue.ParsePath("#components.web.spec.replicas")).Int64()
	require.NoError(t, err, "Evaluated.web.spec.replicas should be readable")
	assert.Equal(t, int64(3), replicas, "Evaluated should carry user-supplied replicas")
}

// TestSnapshot_FieldsReportDifferentValues proves that the two fields in the
// same struct are genuinely independent handles pointing at different nodes in
// the evaluation graph. Reading the same path on each yields different results.
func TestSnapshot_FieldsReportDifferentValues(t *testing.T) {
	_, snap := buildSnapshot(t)

	schemaImage, err := snap.Schema.LookupPath(cue.ParsePath("#components.web.spec.image")).String()
	require.NoError(t, err)

	evaluatedImage, err := snap.Evaluated.LookupPath(cue.ParsePath("#components.web.spec.image")).String()
	require.NoError(t, err)

	assert.NotEqual(t, schemaImage, evaluatedImage,
		"Schema and Evaluated should report different values for the same path")
	assert.Equal(t, "nginx:latest", schemaImage)
	assert.Equal(t, "nginx:1.28.2", evaluatedImage)
}

// TestSnapshot_BothShareSameContext proves that Schema and Evaluated are from
// the same *cue.Context. This is a safety invariant: FillPath and Unify between
// values from different contexts panics with "values are not from the same runtime."
// Storing both in the snapshot guarantees they are always context-compatible.
func TestSnapshot_BothShareSameContext(t *testing.T) {
	_, snap := buildSnapshot(t)

	schemaCtx := snap.Schema.Context()
	evaluatedCtx := snap.Evaluated.Context()

	// Both values were created from the same ctx — the context pointer must match.
	assert.Equal(t, schemaCtx, evaluatedCtx,
		"Schema and Evaluated must share the same *cue.Context")
}

// TestSnapshot_OmittedUserFieldsRetainDefaults proves that fields not present
// in the user values (port, debug) retain their module defaults in Evaluated.
// Only image and replicas were overridden; port and debug are untouched.
func TestSnapshot_OmittedUserFieldsRetainDefaults(t *testing.T) {
	_, snap := buildSnapshot(t)

	// port was not in the user values — schema default *8080 should apply.
	port, err := snap.Evaluated.LookupPath(cue.ParsePath("#components.web.spec.port")).Int64()
	require.NoError(t, err, "Evaluated.web.spec.port should be readable")
	assert.Equal(t, int64(8080), port, "port should retain module default 8080")

	// Schema and Evaluated agree on the untouched port value.
	schemaPort, err := snap.Schema.LookupPath(cue.ParsePath("#components.web.spec.port")).Int64()
	require.NoError(t, err)
	assert.Equal(t, schemaPort, port, "port should be identical in Schema and Evaluated")
}

// TestSnapshot_BothAreConcrete proves that both Schema and Evaluated pass
// Validate(cue.Concrete(true)). Because every #config field carries a CUE
// default, the module is concrete at schema level — FillPath only overrides,
// it does not "unlock" concreteness. Both states are fully deployable values.
func TestSnapshot_BothAreConcrete(t *testing.T) {
	_, snap := buildSnapshot(t)

	assert.NoError(t, snap.Schema.Validate(cue.Concrete(true)),
		"Schema should be concrete (all #config fields have defaults)")
	assert.NoError(t, snap.Evaluated.Validate(cue.Concrete(true)),
		"Evaluated should be concrete (user values + remaining defaults)")
}
