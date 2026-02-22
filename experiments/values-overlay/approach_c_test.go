package valuesoverlay

// ---------------------------------------------------------------------------
// Approach C: Go map merge + single FillPath recompile
//
// The core insight from Approach A: FillPath conflicts as soon as a field is
// set twice with differing concrete values. The root cause is that CUE
// unification is AND — there is no native "replace" or "override" operator.
//
// Approach C escapes the CUE evaluation layer entirely for the merge step:
//
//   1. Extract each value layer as a Go map (via JSON marshalling).
//   2. Deep-merge the maps at the Go level — later maps win on conflict.
//   3. Compile the merged map back to a CUE value.
//   4. FillPath the compiled value onto the abstract base ONCE.
//
// Because the merge happens in Go, there is no CUE unification conflict.
// The abstract base is only touched once, with an already-resolved concrete
// value — CUE narrows the constraint without conflict.
//
// Trade-off: the Go→JSON→CUE round-trip loses CUE-native type information
// (e.g. a CUE expression `replicas: 2 + 1` becomes `3` after marshalling).
// In practice this is acceptable because user values files contain concrete
// scalars, not expressions.
//
// TL;DR (expected outcome):
//   - Single file (full)     → all five fields concrete, user values present
//   - Two overlapping files  → last file wins cleanly, no conflict error
//   - Partial file           → only specified fields are in the merged map
//                              (missing fields remain abstract in the base)
// ---------------------------------------------------------------------------

import (
	"fmt"
	"testing"

	"cuelang.org/go/cue"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// approachCMerge merges zero or more concrete CUE value layers via Go map
// deep-merge and fills the result onto the abstract base via a single FillPath.
//
// layers[0] is lowest priority; layers[len-1] is highest priority (last wins).
// All layers must be concrete (marshalable); the abstract base must NOT be
// concrete (it is the constraint the merged values are validated against).
func approachCMerge(t *testing.T, ctx *cue.Context, base cue.Value, layers ...cue.Value) (cue.Value, error) {
	t.Helper()

	if len(layers) == 0 {
		return base, nil
	}

	// Merge all layers into a single Go map (last wins).
	merged := map[string]any{}
	for _, layer := range layers {
		m := cueValueToMap(t, layer)
		merged = deepMergeMap(merged, m)
	}

	// Compile the merged map back to a CUE value.
	mergedVal, err := mapToCUEValue(ctx, merged)
	if err != nil {
		return cue.Value{}, fmt.Errorf("compiling merged map: %w", err)
	}

	// Single FillPath onto the abstract base — no conflict possible.
	result := base.FillPath(cue.ParsePath("values"), mergedVal)
	return result, result.Err()
}

// TestApproachC_SingleFullFileWorks confirms the basic case: one complete
// user values file produces a fully concrete result.
func TestApproachC_SingleFullFileWorks(t *testing.T) {
	ctx, baseVal := loadModuleBase(t)
	fullVals := loadUserValues(t, ctx, "values_full.cue")

	result, err := approachCMerge(t, ctx, baseVal, fullVals)
	require.NoError(t, err)

	image, ierr := result.LookupPath(cue.ParsePath("values.image")).String()
	require.NoError(t, ierr)
	assert.Equal(t, "app:2.0.0", image)

	env, eerr := result.LookupPath(cue.ParsePath("values.env")).String()
	require.NoError(t, eerr)
	assert.Equal(t, "prod", env)
	t.Logf("Single full file (C): image=%q env=%q", image, env)
}

// TestApproachC_TwoOverlappingFilesLastWins is the primary correctness test.
// values_partial.cue and values_override.cue both set "image". The last file
// (values_override.cue) must win, with no conflict error.
func TestApproachC_TwoOverlappingFilesLastWins(t *testing.T) {
	ctx, baseVal := loadModuleBase(t)
	partial := loadUserValues(t, ctx, "values_partial.cue")   // image="app:1.2.3", replicas=5
	override := loadUserValues(t, ctx, "values_override.cue") // image="app:release", debug=true, env="staging"

	result, err := approachCMerge(t, ctx, baseVal, partial, override)
	require.NoError(t, err, "overlapping files must not conflict with Approach C")

	image, _ := result.LookupPath(cue.ParsePath("values.image")).String()
	assert.Equal(t, "app:release", image, "last file must win on image")

	replicas, _ := result.LookupPath(cue.ParsePath("values.replicas")).Int64()
	assert.Equal(t, int64(5), replicas, "replicas from partial (not overridden) must be present")

	debug, _ := result.LookupPath(cue.ParsePath("values.debug")).Bool()
	assert.Equal(t, true, debug, "debug from override must be present")

	env, _ := result.LookupPath(cue.ParsePath("values.env")).String()
	assert.Equal(t, "staging", env)

	t.Logf("Two overlapping files (C): image=%q replicas=%d debug=%v env=%q",
		image, replicas, debug, env)
}

// TestApproachC_PartialFileLeavesMissingFieldsAbstract shows that a partial
// user file (only image and replicas) leaves port, debug, and env abstract.
// Approach C by itself has no defaults — abstract fields will cause a
// Validate(Concrete(true)) failure if those fields are required by #config.
func TestApproachC_PartialFileLeavesMissingFieldsAbstract(t *testing.T) {
	ctx, baseVal := loadModuleBase(t)
	partial := loadUserValues(t, ctx, "values_partial.cue")

	result, err := approachCMerge(t, ctx, baseVal, partial)
	require.NoError(t, err, "partial file should not error at merge stage")

	// Fields present in partial are concrete.
	image, _ := result.LookupPath(cue.ParsePath("values.image")).String()
	assert.Equal(t, "app:1.2.3", image)

	// Fields absent from partial are still abstract — not fully concrete.
	portAbstract := result.LookupPath(cue.ParsePath("values.port")).Validate(cue.Concrete(true))
	assert.Error(t, portAbstract, "values.port must still be abstract (not in partial file)")
	t.Logf("values.port abstract after C partial merge: %v", portAbstract)

	// The overall result is not fully concrete — validate(Concrete) will fail.
	fullConcrete := result.Validate(cue.Concrete(true))
	assert.Error(t, fullConcrete, "result with partial file is not fully concrete — expected")
	t.Logf("Result not fully concrete (expected): %v", fullConcrete)
}

// TestApproachC_MergeOrderDeterminesWinner proves that the order of layers
// directly controls which value wins — the LAST layer always wins.
// Reversing the order reverses the winner.
func TestApproachC_MergeOrderDeterminesWinner(t *testing.T) {
	ctx, baseVal := loadModuleBase(t)
	partial := loadUserValues(t, ctx, "values_partial.cue")   // image="app:1.2.3"
	override := loadUserValues(t, ctx, "values_override.cue") // image="app:release"

	// Order 1: partial first → override last → override wins
	resultOverrideLast, err := approachCMerge(t, ctx, baseVal, partial, override)
	require.NoError(t, err)
	imgOverrideLast, _ := resultOverrideLast.LookupPath(cue.ParsePath("values.image")).String()
	assert.Equal(t, "app:release", imgOverrideLast, "last=override: override must win")

	// Order 2: override first → partial last → partial wins
	resultPartialLast, err := approachCMerge(t, ctx, baseVal, override, partial)
	require.NoError(t, err)
	imgPartialLast, _ := resultPartialLast.LookupPath(cue.ParsePath("values.image")).String()
	assert.Equal(t, "app:1.2.3", imgPartialLast, "last=partial: partial must win")

	t.Logf("Order matters: override-last=%q partial-last=%q", imgOverrideLast, imgPartialLast)
}

// TestApproachC_RoundTripPreservesTypes verifies that the JSON round-trip
// (CUE → JSON map → CUE) preserves the concrete scalar values correctly.
// This test documents any potential precision or type loss through marshalling.
func TestApproachC_RoundTripPreservesTypes(t *testing.T) {
	ctx, baseVal := loadModuleBase(t)
	fullVals := loadUserValues(t, ctx, "values_full.cue")

	result, err := approachCMerge(t, ctx, baseVal, fullVals)
	require.NoError(t, err)

	image, err := result.LookupPath(cue.ParsePath("values.image")).String()
	require.NoError(t, err)
	assert.Equal(t, "app:2.0.0", image, "string preserved through round-trip")

	replicas, err := result.LookupPath(cue.ParsePath("values.replicas")).Int64()
	require.NoError(t, err)
	assert.Equal(t, int64(3), replicas, "int preserved through round-trip")

	port, err := result.LookupPath(cue.ParsePath("values.port")).Int64()
	require.NoError(t, err)
	assert.Equal(t, int64(9090), port, "int preserved through round-trip")

	debug, err := result.LookupPath(cue.ParsePath("values.debug")).Bool()
	require.NoError(t, err)
	assert.Equal(t, false, debug, "bool preserved through round-trip")

	env, err := result.LookupPath(cue.ParsePath("values.env")).String()
	require.NoError(t, err)
	assert.Equal(t, "prod", env, "enum string preserved through round-trip")

	t.Logf("Round-trip: image=%q replicas=%d port=%d debug=%v env=%q",
		image, replicas, port, debug, env)
}
