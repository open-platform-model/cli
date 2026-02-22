package valuesoverlay

// ---------------------------------------------------------------------------
// Approach B: CUE-native priority stack — reverse-order abstract detection
//
// Approach A showed that FillPath unifies, making it impossible to override
// a field once it has been set to a concrete value. This approach works
// around that constraint by staying in CUE-land but changing the order of
// operations:
//
//   1. Process value layers in REVERSE priority order (lowest priority first
//      … highest priority last, so high-priority layers are applied FIRST
//      to the still-abstract base).
//   2. Before filling a field, check whether it is already concrete.
//      If concrete → skip (a higher-priority layer already set it).
//      If abstract → fill (this is the first/only layer to set it).
//
// Concretely, for user files [file1, file2] where file2 has higher priority:
//   pass 1: iterate file2 fields → all abstract → fill all
//   pass 2: iterate file1 fields → fields set by file2 are concrete → skip
//   pass 3: iterate defaults    → fields set by user files are concrete → skip
//
// This gives last-wins semantics without leaving the CUE evaluation layer.
//
// Limitation: it only handles FLAT (non-nested) values structs cleanly.
// Nested structs require recursive field walking.
//
// TL;DR (expected outcome):
//   - Single file (full)     → works, user values concrete
//   - Single file (partial)  → works, missing fields remain abstract
//                              (defaults must be applied as a follow-up pass)
//   - Two overlapping files  → works, last file wins on each field
//   - With defaults pass     → full coverage: user wins, gaps filled by defaults
// ---------------------------------------------------------------------------

import (
	"testing"

	"cuelang.org/go/cue"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// approachBFill applies a slice of value layers to the abstract base using
// the reverse-order abstract-detection strategy.
//
// layers[0] is the LOWEST priority; layers[len-1] is the HIGHEST priority.
// Fields are applied highest-priority-first; if a field is already concrete
// when a lower-priority layer is processed, it is skipped.
//
// Returns the result value. The caller should check result.Err().
func approachBFill(t *testing.T, base cue.Value, layers ...cue.Value) cue.Value {
	t.Helper()
	result := base

	// Iterate layers in REVERSE order (highest priority first → fills the
	// abstract fields first; lower-priority layers see concrete fields and skip).
	for i := len(layers) - 1; i >= 0; i-- {
		layer := layers[i]

		iter, err := layer.Fields()
		require.NoError(t, err, "layer %d must be iterable", i)

		for iter.Next() {
			fieldPath := cue.MakePath(cue.Str("values"), iter.Selector())
			existing := result.LookupPath(fieldPath)

			// Only fill if the field is not yet concrete.
			// Validate(cue.Concrete(true)) returns nil iff the value is concrete.
			if existing.Validate(cue.Concrete(true)) == nil {
				// Already concrete — a higher-priority layer set this field. Skip.
				continue
			}
			result = result.FillPath(fieldPath, iter.Value())
		}
	}

	return result
}

// TestApproachB_SingleFullFileWorks confirms the common case: one user file
// containing all five fields, applied to the abstract base. All fields should
// be concrete with the user's values.
func TestApproachB_SingleFullFileWorks(t *testing.T) {
	ctx, baseVal := loadModuleBase(t)
	fullVals := loadUserValues(t, ctx, "values_full.cue")

	result := approachBFill(t, baseVal, fullVals)
	assert.NoError(t, result.Err(), "single full file should produce no error")

	image, err := result.LookupPath(cue.ParsePath("values.image")).String()
	require.NoError(t, err)
	assert.Equal(t, "app:2.0.0", image)

	env, err := result.LookupPath(cue.ParsePath("values.env")).String()
	require.NoError(t, err)
	assert.Equal(t, "prod", env)
	t.Logf("Single full file: image=%q env=%q", image, env)
}

// TestApproachB_TwoOverlappingFilesLastWins verifies that when two user
// files share a key (image), the LAST file in the slice wins. The reverse
// iteration processes the last file first, setting the field concrete; the
// first file then sees the field already concrete and skips it.
func TestApproachB_TwoOverlappingFilesLastWins(t *testing.T) {
	ctx, baseVal := loadModuleBase(t)

	// values_partial.cue: image="app:1.2.3", replicas=5
	// values_override.cue: image="app:release", debug=true, env="staging"
	// overlap on "image" → values_override.cue (last) should win
	partial := loadUserValues(t, ctx, "values_partial.cue")
	override := loadUserValues(t, ctx, "values_override.cue")

	// layers[0]=partial (low priority), layers[1]=override (high priority)
	result := approachBFill(t, baseVal, partial, override)
	assert.NoError(t, result.Err(), "two overlapping files should not conflict with Approach B")

	image, err := result.LookupPath(cue.ParsePath("values.image")).String()
	require.NoError(t, err, "values.image should be concrete")
	assert.Equal(t, "app:release", image, "last file (override) must win on image")

	replicas, err := result.LookupPath(cue.ParsePath("values.replicas")).Int64()
	require.NoError(t, err, "values.replicas should be concrete (from partial, no conflict)")
	assert.Equal(t, int64(5), replicas, "replicas from partial (not in override) should be present")

	env, err := result.LookupPath(cue.ParsePath("values.env")).String()
	require.NoError(t, err, "values.env should be concrete (from override)")
	assert.Equal(t, "staging", env)

	t.Logf("Two overlapping files: image=%q (override wins), replicas=%d (from partial), env=%q",
		image, replicas, env)
}

// TestApproachB_PartialFileLeavesMissingFieldsAbstract shows that a partial
// user file (only image and replicas) leaves port, debug, env still abstract
// after approachBFill. This is expected — the caller must decide how to fill
// the remaining abstract fields (typically via author defaults or an error).
func TestApproachB_PartialFileLeavesMissingFieldsAbstract(t *testing.T) {
	ctx, baseVal := loadModuleBase(t)
	partial := loadUserValues(t, ctx, "values_partial.cue")

	result := approachBFill(t, baseVal, partial)

	// image and replicas are set by the partial file — should be concrete.
	image, err := result.LookupPath(cue.ParsePath("values.image")).String()
	require.NoError(t, err, "values.image should be concrete")
	assert.Equal(t, "app:1.2.3", image)

	replicas, err := result.LookupPath(cue.ParsePath("values.replicas")).Int64()
	require.NoError(t, err, "values.replicas should be concrete")
	assert.Equal(t, int64(5), replicas)

	// port, debug, env are not in the partial file — should be abstract.
	portErr := result.LookupPath(cue.ParsePath("values.port")).Validate(cue.Concrete(true))
	assert.Error(t, portErr, "values.port should still be abstract (not in partial file)")
	t.Logf("values.port abstract after partial file: %v", portErr)
}

// TestApproachB_DefaultsAsLowestPriorityLayer demonstrates the complete
// defaults-fallthrough pattern using Approach B:
//
//	layers = [authorDefaults, userPartialFile]
//
// authorDefaults is layer 0 (lowest priority).
// userPartialFile is layer 1 (highest priority — processed first).
//
// Fields present in userPartialFile are set first (concrete). When the
// defaults layer runs, those fields are already concrete and are skipped.
// Fields absent from userPartialFile are still abstract when the defaults
// layer runs, so defaults fill the gaps.
func TestApproachB_DefaultsAsLowestPriorityLayer(t *testing.T) {
	ctx, baseVal := loadModuleBase(t)
	defaults := loadAuthorDefaults(t, ctx)
	partial := loadUserValues(t, ctx, "values_partial.cue")

	// layers: [defaults (lowest), partial (highest)]
	result := approachBFill(t, baseVal, defaults, partial)
	assert.NoError(t, result.Err(), "defaults + partial should produce no error")

	// User values win on fields they provide.
	image, err := result.LookupPath(cue.ParsePath("values.image")).String()
	require.NoError(t, err)
	assert.Equal(t, "app:1.2.3", image, "user partial image must override default")

	replicas, err := result.LookupPath(cue.ParsePath("values.replicas")).Int64()
	require.NoError(t, err)
	assert.Equal(t, int64(5), replicas, "user partial replicas must override default")

	// Author defaults fill the gaps for fields absent from the user file.
	port, err := result.LookupPath(cue.ParsePath("values.port")).Int64()
	require.NoError(t, err, "values.port should be filled by author defaults")
	assert.Equal(t, int64(8080), port, "port should come from author defaults")

	debug, err := result.LookupPath(cue.ParsePath("values.debug")).Bool()
	require.NoError(t, err, "values.debug should be filled by author defaults")
	assert.Equal(t, false, debug, "debug should come from author defaults")

	env, err := result.LookupPath(cue.ParsePath("values.env")).String()
	require.NoError(t, err, "values.env should be filled by author defaults")
	assert.Equal(t, "dev", env, "env should come from author defaults")

	t.Logf("Defaults + partial (B): image=%q replicas=%d port=%d debug=%v env=%q",
		image, replicas, port, debug, env)
}

// TestApproachB_ThreeLayersCorrectPriority verifies three-layer precedence:
// defaults < partial < override, with override winning on image (set in all
// three layers), and lower layers filling fields not present in higher layers.
func TestApproachB_ThreeLayersCorrectPriority(t *testing.T) {
	ctx, baseVal := loadModuleBase(t)
	defaults := loadAuthorDefaults(t, ctx)
	partial := loadUserValues(t, ctx, "values_partial.cue")   // image, replicas
	override := loadUserValues(t, ctx, "values_override.cue") // image, debug, env

	// layers[0]=defaults, layers[1]=partial, layers[2]=override
	result := approachBFill(t, baseVal, defaults, partial, override)
	assert.NoError(t, result.Err(), "three-layer merge should produce no error")

	// override wins on image (it is the highest-priority layer that sets it)
	image, err := result.LookupPath(cue.ParsePath("values.image")).String()
	require.NoError(t, err)
	assert.Equal(t, "app:release", image, "override (highest priority) wins on image")

	// partial wins on replicas (override doesn't set it; partial is next)
	replicas, err := result.LookupPath(cue.ParsePath("values.replicas")).Int64()
	require.NoError(t, err)
	assert.Equal(t, int64(5), replicas, "partial wins on replicas (not in override)")

	// defaults fill port (neither user file sets it)
	port, err := result.LookupPath(cue.ParsePath("values.port")).Int64()
	require.NoError(t, err)
	assert.Equal(t, int64(8080), port, "defaults fill port (absent from both user files)")

	// override wins on debug and env
	debug, err := result.LookupPath(cue.ParsePath("values.debug")).Bool()
	require.NoError(t, err)
	assert.Equal(t, true, debug, "override sets debug=true")

	env, err := result.LookupPath(cue.ParsePath("values.env")).String()
	require.NoError(t, err)
	assert.Equal(t, "staging", env, "override wins on env")

	t.Logf("Three layers: image=%q replicas=%d port=%d debug=%v env=%q",
		image, replicas, port, debug, env)
}
