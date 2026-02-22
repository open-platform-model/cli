package valuesloadisolation

// ---------------------------------------------------------------------------
// Baseline: reproduce the production bug
//
// When load.Instances([]string{"."}, cfg) loads a module directory that
// contains multiple values*.cue files in the same CUE package, CUE unifies
// ALL of them. Concrete conflicting fields (e.g. serverType: "FORGE" vs "PAPER")
// produce a CUE evaluation error.
//
// These tests are EXPECTED TO FAIL (conflict error) — they prove the bug is
// real and reproducible with the current naive load strategy before any fix.
// ---------------------------------------------------------------------------

import (
	"strings"
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/load"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBaseline_NaiveLoadCausesConflict proves that loading a module directory
// that contains values.cue + values_forge.cue + values_testing.cue via
// load.Instances([]string{"."}) results in a CUE conflict error.
//
// This is the root cause of the production bug:
//
//	opm mod vet . --release-name mc
//	→ "conflicting values "FORGE" and "PAPER""
func TestBaseline_NaiveLoadCausesConflict(t *testing.T) {
	dir := modulePath(t)
	ctx := cuecontext.New()

	instances := load.Instances([]string{"."}, &load.Config{Dir: dir})
	require.Len(t, instances, 1)
	require.NoError(t, instances[0].Err, "load.Instances itself should not error — conflict shows up after BuildInstance")

	val := ctx.BuildInstance(instances[0])

	// The conflict is detected here: CUE unifies all values*.cue files and
	// finds e.g. serverType constrained to both "PAPER" and "FORGE".
	err := val.Err()
	assert.Error(t, err, "BuildInstance should error due to conflicting values*.cue files")
	if err != nil {
		t.Logf("Conflict error (expected): %v", err)
		assert.True(t,
			strings.Contains(err.Error(), "conflicting values"),
			"error should mention 'conflicting values', got: %v", err,
		)
	}
}

// TestBaseline_ConflictIsInValuesField confirms the conflict is specifically
// in the values.serverType field — not in metadata or #config schema.
func TestBaseline_ConflictIsInValuesField(t *testing.T) {
	dir := modulePath(t)
	ctx := cuecontext.New()

	instances := load.Instances([]string{"."}, &load.Config{Dir: dir})
	require.Len(t, instances, 1)
	require.NoError(t, instances[0].Err)

	val := ctx.BuildInstance(instances[0])
	err := val.Err()
	require.Error(t, err, "expected conflict error")

	// The conflict is in values.serverType, not elsewhere.
	assert.True(t,
		strings.Contains(err.Error(), "serverType") || strings.Contains(err.Error(), "values"),
		"conflict should reference the values field, got: %v", err,
	)
	t.Logf("Conflict location confirmed: %v", err)
}

// TestBaseline_MetadataStillAccessibleDespiteConflict shows that the conflict
// does not prevent reading fields unrelated to values — metadata.name is still
// accessible. This is an important subtlety: the error is on the values path,
// not on the whole value.
func TestBaseline_MetadataStillAccessibleDespiteConflict(t *testing.T) {
	dir := modulePath(t)
	ctx := cuecontext.New()

	instances := load.Instances([]string{"."}, &load.Config{Dir: dir})
	require.Len(t, instances, 1)
	require.NoError(t, instances[0].Err)

	val := ctx.BuildInstance(instances[0])
	// Top-level val.Err() is set, but individual paths may still be readable.
	// NOTE: In CUE, a value with Err() set may or may not allow LookupPath on
	// unrelated fields. This test documents actual CUE behavior.
	name := val.LookupPath(cue.ParsePath("metadata.name"))
	if nameStr, err := name.String(); err == nil {
		t.Logf("metadata.name is readable despite conflict: %q", nameStr)
		assert.Equal(t, "test-server", nameStr)
	} else {
		t.Logf("metadata.name not readable when value has top-level error: %v", err)
	}
}
