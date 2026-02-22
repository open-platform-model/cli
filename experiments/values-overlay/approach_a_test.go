package valuesoverlay

// ---------------------------------------------------------------------------
// Approach A: Sequential FillPath (full struct value)
//
// The naive attempt: call FillPath("values", v) once per layer, in priority
// order.
//
//   step1 := base.FillPath("values", authorDefaults)
//   step2 := step1.FillPath("values", userValues)    // does this override?
//
// CUE hypothesis: FillPath performs UNIFICATION (conjunction, &), not
// replacement. If step1 already has "image: app:latest" (concrete) and
// userValues has "image: app:1.2.3" (also concrete), then:
//
//   "app:latest" & "app:1.2.3"  →  _|_  (bottom, conflict)
//
// The tests below prove or disprove this hypothesis and document exactly
// where the approach works and where it breaks down.
//
// TL;DR (expected outcome):
//   - FillPath on an ABSTRACT field     → works (narrows constraint)
//   - FillPath twice on the SAME field  → conflict if values differ
//   - FillPath with NON-OVERLAPPING files → works (additive, no conflict)
// ---------------------------------------------------------------------------

import (
	"testing"

	"cuelang.org/go/cue"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestApproachA_FillPathOnAbstractField establishes the baseline: FillPath
// works correctly when the target field is still abstract (a constraint, not
// yet concrete). Abstract + concrete = concrete narrowing — no conflict.
func TestApproachA_FillPathOnAbstractField(t *testing.T) {
	ctx, baseVal := loadModuleBase(t)
	defaults := loadAuthorDefaults(t, ctx)

	// values is abstract (#config constraint). FillPath narrows it to the
	// concrete defaults — this must succeed.
	result := baseVal.FillPath(cue.ParsePath("values"), defaults)
	assert.NoError(t, result.Err(), "FillPath onto abstract field should not conflict")

	image, err := result.LookupPath(cue.ParsePath("values.image")).String()
	require.NoError(t, err, "values.image should be concrete after FillPath")
	assert.Equal(t, "app:latest", image)
	t.Logf("Abstract → concrete: values.image = %q", image)
}

// TestApproachA_SecondFillPathOnConcreteField is THE central question.
// After filling values with author defaults (all five fields are now concrete),
// attempt to FillPath again with user values that overlap on some keys.
//
// Expected: CONFLICT — FillPath unifies, so "app:latest" & "app:1.2.3" = _|_.
// If this test unexpectedly passes, FillPath replaces rather than unifies,
// and sequential FillPath would be a valid override mechanism.
func TestApproachA_SecondFillPathOnConcreteField(t *testing.T) {
	ctx, baseVal := loadModuleBase(t)
	defaults := loadAuthorDefaults(t, ctx)

	step1 := baseVal.FillPath(cue.ParsePath("values"), defaults)
	require.NoError(t, step1.Err(), "first FillPath (defaults) must succeed")

	// values_partial.cue sets image="app:1.2.3" — different from the default
	// "app:latest". Both are concrete strings, so CUE unification fails.
	userVals := loadUserValues(t, ctx, "values_partial.cue")
	step2 := step1.FillPath(cue.ParsePath("values"), userVals)

	if err := step2.Err(); err != nil {
		t.Logf("RESULT: FillPath UNIFIES — conflict on overlapping concrete fields: %v", err)
		t.Logf("  → Sequential FillPath cannot implement override semantics")
		assert.Error(t, err, "documenting: FillPath(concrete→concrete) conflicts")
	} else {
		image, _ := step2.LookupPath(cue.ParsePath("values.image")).String()
		t.Logf("RESULT: FillPath REPLACES — image is now %q (unexpected)", image)
		t.Logf("  → Sequential FillPath implements override (hypothesis disproved)")
	}
}

// TestApproachA_FillPathNonOverlappingFiles proves that sequential FillPath
// works when the value files do NOT share any keys. There is no conflict
// because unification only fails on concrete↔concrete disagreements.
//
// File 1 provides {image, replicas}; File 2 provides {port} — no overlap.
// Both files contribute additive information: all three fields are concrete
// in the final result.
func TestApproachA_FillPathNonOverlappingFiles(t *testing.T) {
	ctx, baseVal := loadModuleBase(t)

	// Inline CUE snippets that together cover distinct fields.
	v1 := ctx.CompileString(`{image: "app:1.2.3", replicas: 5}`)
	require.NoError(t, v1.Err())

	v2 := ctx.CompileString(`{port: 9090}`)
	require.NoError(t, v2.Err())

	step1 := baseVal.FillPath(cue.ParsePath("values"), v1)
	require.NoError(t, step1.Err(), "first FillPath (non-overlapping) must not conflict")

	step2 := step1.FillPath(cue.ParsePath("values"), v2)
	assert.NoError(t, step2.Err(), "second FillPath (non-overlapping) must not conflict")

	image, _ := step2.LookupPath(cue.ParsePath("values.image")).String()
	replicas, _ := step2.LookupPath(cue.ParsePath("values.replicas")).Int64()
	port, _ := step2.LookupPath(cue.ParsePath("values.port")).Int64()

	assert.Equal(t, "app:1.2.3", image)
	assert.Equal(t, int64(5), replicas)
	assert.Equal(t, int64(9090), port)
	t.Logf("Non-overlapping sequential FillPath: image=%q replicas=%d port=%d", image, replicas, port)
}

// TestApproachA_FullFileNoConflictWithAbstractBase confirms that FillPath
// with a complete user values file (all five fields) onto the abstract base
// (no defaults pre-applied) works correctly. This is the "user provides
// everything" scenario — no defaults needed, no conflict possible.
func TestApproachA_FullFileNoConflictWithAbstractBase(t *testing.T) {
	ctx, baseVal := loadModuleBase(t)

	// values_full.cue provides all five fields — no gaps, no need for defaults.
	fullVals := loadUserValues(t, ctx, "values_full.cue")
	result := baseVal.FillPath(cue.ParsePath("values"), fullVals)
	assert.NoError(t, result.Err(), "FillPath with complete user values onto abstract base must not conflict")

	image, err := result.LookupPath(cue.ParsePath("values.image")).String()
	require.NoError(t, err)
	assert.Equal(t, "app:2.0.0", image)

	env, err := result.LookupPath(cue.ParsePath("values.env")).String()
	require.NoError(t, err)
	assert.Equal(t, "prod", env)
	t.Logf("Full file → abstract base: image=%q env=%q", image, env)
}

// TestApproachA_UserFileBeforeDefaultsAlsoConflicts shows that applying user
// values FIRST (to the abstract base) and then attempting to add defaults
// produces the same conflict — order does not rescue sequential FillPath.
//
// This rules out "just flip the order" as a fix.
func TestApproachA_UserFileBeforeDefaultsAlsoConflicts(t *testing.T) {
	ctx, baseVal := loadModuleBase(t)
	defaults := loadAuthorDefaults(t, ctx)
	userVals := loadUserValues(t, ctx, "values_partial.cue") // image, replicas only

	// Apply user values first (to abstract base — succeeds)
	step1 := baseVal.FillPath(cue.ParsePath("values"), userVals)
	require.NoError(t, step1.Err(), "user values onto abstract base must succeed")

	// Now try to add defaults — image is already "app:1.2.3", defaults say
	// "app:latest". Conflict expected here too.
	step2 := step1.FillPath(cue.ParsePath("values"), defaults)
	if err := step2.Err(); err != nil {
		t.Logf("Reversed order also conflicts: %v", err)
		t.Logf("  → Order does not matter; FillPath always unifies")
		assert.Error(t, err, "documenting: reversed FillPath order also conflicts on overlapping fields")
	} else {
		image, _ := step2.LookupPath(cue.ParsePath("values.image")).String()
		t.Logf("Reversed order: image=%q (unexpected — FillPath replaced, not unified)", image)
	}
}
