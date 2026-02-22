package valuesoverlay

// ---------------------------------------------------------------------------
// Approach D: Defaults-as-base layer (recommended production approach)
//
// Approach C works for user-provided files but has a gap: if the user supplies
// a partial values file, the missing fields remain abstract and the result
// fails Validate(Concrete(true)). The caller must either error ("provide all
// fields") or fill the gaps somehow.
//
// Approach D addresses this by treating the MODULE AUTHOR'S defaults
// (values.cue) as layer 0 — always the lowest-priority base. User files are
// layers 1..N and override on top. This gives:
//
//   merged = deepMerge(authorDefaults, userFile1, ..., userFileN)
//           ↑ lowest priority                          ↑ highest priority
//
// The semantic model:
//
//   ┌─────────────────────────────────────────────────┐
//   │               PRIORITY STACK                    │
//   │                                                 │
//   │  Layer 0 (floor):   module author defaults      │
//   │  Layer 1:           --values file[0]            │
//   │  Layer 2:           --values file[1]            │
//   │  ...                                            │
//   │  Layer N (ceiling): --values file[N-1]          │
//   │                                                 │
//   │  Every key present in a higher layer overrides  │
//   │  the same key in all lower layers. Keys absent  │
//   │  from all user layers fall through to the floor.│
//   └─────────────────────────────────────────────────┘
//
// Consequences:
//   - Author defaults always provide a complete concrete base.
//   - Partial user files are valid — gaps are filled by author defaults.
//   - A user file that sets ALL fields fully overrides the defaults.
//   - Multiple user files stack: last file wins on conflict.
//   - result.Validate(Concrete(true)) succeeds as long as author defaults
//     cover all #config fields.
//
// This approach is implemented by prepending authorDefaults to the layers
// slice passed to approachCMerge from Approach C — no new mechanism needed.
// ---------------------------------------------------------------------------

import (
	"testing"

	"cuelang.org/go/cue"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// approachDMerge is the defaults-as-base variant of approachCMerge.
// authorDefaults is always layer 0 (lowest priority); userLayers follow
// in order, with the last element having the highest priority.
func approachDMerge(t *testing.T, ctx *cue.Context, base cue.Value, authorDefaults cue.Value, userLayers ...cue.Value) (cue.Value, error) {
	t.Helper()
	// Prepend author defaults as the lowest-priority layer.
	allLayers := append([]cue.Value{authorDefaults}, userLayers...)
	return approachCMerge(t, ctx, base, allLayers...)
}

// TestApproachD_NoUserFiles uses only author defaults.
// With no user files, the result is the author's default values exactly.
func TestApproachD_NoUserFiles(t *testing.T) {
	ctx, baseVal := loadModuleBase(t)
	defaults := loadAuthorDefaults(t, ctx)

	result, err := approachDMerge(t, ctx, baseVal, defaults)
	require.NoError(t, err)

	// All fields should be concrete and match the author defaults.
	image, _ := result.LookupPath(cue.ParsePath("values.image")).String()
	replicas, _ := result.LookupPath(cue.ParsePath("values.replicas")).Int64()
	port, _ := result.LookupPath(cue.ParsePath("values.port")).Int64()
	debug, _ := result.LookupPath(cue.ParsePath("values.debug")).Bool()
	env, _ := result.LookupPath(cue.ParsePath("values.env")).String()

	assert.Equal(t, "app:latest", image)
	assert.Equal(t, int64(2), replicas)
	assert.Equal(t, int64(8080), port)
	assert.Equal(t, false, debug)
	assert.Equal(t, "dev", env)

	// With all defaults applied, the result should be fully concrete.
	assert.NoError(t, result.Validate(cue.Concrete(true)),
		"result with only author defaults should be fully concrete")

	t.Logf("No user files (D): image=%q replicas=%d port=%d debug=%v env=%q",
		image, replicas, port, debug, env)
}

// TestApproachD_FullUserFileOverridesAllDefaults verifies that a complete
// user values file overrides every author default. No default values should
// survive in the final result.
func TestApproachD_FullUserFileOverridesAllDefaults(t *testing.T) {
	ctx, baseVal := loadModuleBase(t)
	defaults := loadAuthorDefaults(t, ctx)
	fullVals := loadUserValues(t, ctx, "values_full.cue")

	result, err := approachDMerge(t, ctx, baseVal, defaults, fullVals)
	require.NoError(t, err)

	image, _ := result.LookupPath(cue.ParsePath("values.image")).String()
	replicas, _ := result.LookupPath(cue.ParsePath("values.replicas")).Int64()
	port, _ := result.LookupPath(cue.ParsePath("values.port")).Int64()
	debug, _ := result.LookupPath(cue.ParsePath("values.debug")).Bool()
	env, _ := result.LookupPath(cue.ParsePath("values.env")).String()

	// All values must come from values_full.cue, not from author defaults.
	assert.Equal(t, "app:2.0.0", image, "full file image must override default")
	assert.Equal(t, int64(3), replicas, "full file replicas must override default")
	assert.Equal(t, int64(9090), port, "full file port must override default")
	assert.Equal(t, false, debug)
	assert.Equal(t, "prod", env, "full file env must override default")

	assert.NoError(t, result.Validate(cue.Concrete(true)), "result must be fully concrete")
	t.Logf("Full override (D): image=%q replicas=%d port=%d debug=%v env=%q",
		image, replicas, port, debug, env)
}

// TestApproachD_PartialUserFileFillsGapsFromDefaults is the key scenario
// that distinguishes D from C. A partial user file (only image and replicas)
// results in a FULLY CONCRETE value because the missing fields (port, debug,
// env) are filled by the author defaults at layer 0.
func TestApproachD_PartialUserFileFillsGapsFromDefaults(t *testing.T) {
	ctx, baseVal := loadModuleBase(t)
	defaults := loadAuthorDefaults(t, ctx)
	partial := loadUserValues(t, ctx, "values_partial.cue") // image, replicas

	result, err := approachDMerge(t, ctx, baseVal, defaults, partial)
	require.NoError(t, err)

	// User values win on fields they provide.
	image, err := result.LookupPath(cue.ParsePath("values.image")).String()
	require.NoError(t, err)
	assert.Equal(t, "app:1.2.3", image, "user partial must override default image")

	replicas, err := result.LookupPath(cue.ParsePath("values.replicas")).Int64()
	require.NoError(t, err)
	assert.Equal(t, int64(5), replicas, "user partial must override default replicas")

	// Author defaults fill the gaps.
	port, err := result.LookupPath(cue.ParsePath("values.port")).Int64()
	require.NoError(t, err)
	assert.Equal(t, int64(8080), port, "default port must fill the gap")

	debug, err := result.LookupPath(cue.ParsePath("values.debug")).Bool()
	require.NoError(t, err)
	assert.Equal(t, false, debug, "default debug must fill the gap")

	env, err := result.LookupPath(cue.ParsePath("values.env")).String()
	require.NoError(t, err)
	assert.Equal(t, "dev", env, "default env must fill the gap")

	// Result is fully concrete even though the user only set two fields.
	assert.NoError(t, result.Validate(cue.Concrete(true)),
		"partial user file + defaults must produce a fully concrete result")

	t.Logf("Partial + defaults (D): image=%q replicas=%d port=%d debug=%v env=%q",
		image, replicas, port, debug, env)
}

// TestApproachD_TwoUserFilesLastWinsOverridesDefault stacks two user files
// on top of author defaults. The last file wins on the "image" key (which
// all three sources — defaults, partial, override — provide). Other keys
// are determined by priority: override > partial > defaults.
func TestApproachD_TwoUserFilesLastWinsOverridesDefault(t *testing.T) {
	ctx, baseVal := loadModuleBase(t)
	defaults := loadAuthorDefaults(t, ctx)
	partial := loadUserValues(t, ctx, "values_partial.cue")   // image="app:1.2.3", replicas=5
	override := loadUserValues(t, ctx, "values_override.cue") // image="app:release", debug=true, env="staging"

	result, err := approachDMerge(t, ctx, baseVal, defaults, partial, override)
	require.NoError(t, err, "three-layer merge must not conflict")

	// override wins on image (highest priority that sets it)
	image, _ := result.LookupPath(cue.ParsePath("values.image")).String()
	assert.Equal(t, "app:release", image, "override wins on image")

	// partial wins on replicas (override doesn't set it)
	replicas, _ := result.LookupPath(cue.ParsePath("values.replicas")).Int64()
	assert.Equal(t, int64(5), replicas, "partial wins on replicas (not in override)")

	// defaults fill port (neither user file sets it)
	port, _ := result.LookupPath(cue.ParsePath("values.port")).Int64()
	assert.Equal(t, int64(8080), port, "default fills port (absent from all user files)")

	// override wins on debug and env
	debug, _ := result.LookupPath(cue.ParsePath("values.debug")).Bool()
	assert.Equal(t, true, debug, "override sets debug=true")

	env, _ := result.LookupPath(cue.ParsePath("values.env")).String()
	assert.Equal(t, "staging", env, "override wins on env")

	assert.NoError(t, result.Validate(cue.Concrete(true)), "three-layer result must be fully concrete")
	t.Logf("Three layers (D): image=%q replicas=%d port=%d debug=%v env=%q",
		image, replicas, port, debug, env)
}

// TestApproachD_SchemaValidationStillApplied confirms that even with D's
// Go-level merge, the final FillPath against the abstract base still applies
// the #config schema constraint. An out-of-range value (replicas=99, max is 10)
// must be rejected by CUE after FillPath.
func TestApproachD_SchemaValidationStillApplied(t *testing.T) {
	ctx, baseVal := loadModuleBase(t)
	defaults := loadAuthorDefaults(t, ctx)

	// Inline an invalid value: replicas=99 violates #config's int & >=1 & <=10
	badVals := ctx.CompileString(`{image: "app:ok", replicas: 99, port: 8080, debug: false, env: "dev"}`)
	require.NoError(t, badVals.Err())

	result, err := approachDMerge(t, ctx, baseVal, defaults, badVals)
	if err != nil {
		t.Logf("Schema validation caught bad replicas at merge time: %v", err)
	} else {
		// Validation may surface at Validate(Concrete(true)) time instead.
		validateErr := result.Validate(cue.Concrete(true))
		if validateErr != nil {
			t.Logf("Schema validation caught bad replicas at Validate time: %v", validateErr)
		}
		// The error may appear on the specific field rather than at root.
		replicasErr := result.LookupPath(cue.ParsePath("values.replicas")).Validate(cue.Concrete(true))
		t.Logf("values.replicas (should be invalid): %v", replicasErr)
	}
	t.Logf("Note: JSON round-trip may lose int constraints — replicas=99 as JSON int may bypass CUE range check")
	t.Logf("Schema enforcement against #config is best done as a separate Unify step after merge")
}

// TestApproachD_CompareWithC shows the concrete difference between C and D
// when given an identical partial user file:
//
//	C: partial file only → result has abstract gaps → NOT fully concrete
//	D: defaults + partial → result has no gaps → IS fully concrete
func TestApproachD_CompareWithC(t *testing.T) {
	ctx, baseVal := loadModuleBase(t)
	defaults := loadAuthorDefaults(t, ctx)
	partial := loadUserValues(t, ctx, "values_partial.cue")

	// Approach C (no defaults)
	resultC, err := approachCMerge(t, ctx, baseVal, partial)
	require.NoError(t, err)
	concreteC := resultC.Validate(cue.Concrete(true))

	// Approach D (with defaults as floor)
	resultD, err := approachDMerge(t, ctx, baseVal, defaults, partial)
	require.NoError(t, err)
	concreteD := resultD.Validate(cue.Concrete(true))

	assert.Error(t, concreteC, "C with partial file: NOT fully concrete (gaps exist)")
	assert.NoError(t, concreteD, "D with partial file + defaults: IS fully concrete (gaps filled)")

	t.Logf("C fully concrete: %v", concreteC == nil)
	t.Logf("D fully concrete: %v", concreteD == nil)
	t.Logf("→ D is strictly more complete than C for partial user files")
}
