package valuesnapshot

// ---------------------------------------------------------------------------
// Immutability: FillPath and Unify never mutate the receiver
//
// The CUE Go API guarantees that all cue.Value operations are pure. The value
// on which you call FillPath or Unify is never modified — a new value is
// returned that incorporates the change. The original handle continues to
// point at the same immutable node in the evaluation graph.
//
// These tests prove the invariant concretely using moduleSnapshot.Schema:
//
//   base := schema
//   evaluated := base.FillPath(...)  // produces a new value
//   // base is exactly as it was before — same concrete field values
//
// This is the foundational property that makes storing pre/post values in a
// single struct safe. Neither field can inadvertently change the other.
//
// Reference: docs/design/cue-in-go.md "Divergent copies via operations"
// (cue-in-go.md:188-212)
// ---------------------------------------------------------------------------

import (
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/load"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestImmutability_SchemaUnchangedAfterFillPath proves that reading a field
// from Schema before and after Evaluated was created via FillPath returns the
// same value. FillPath on Schema produces Evaluated — Schema is not touched.
func TestImmutability_SchemaUnchangedAfterFillPath(t *testing.T) {
	modulePath := testModulePath(t)
	ctx := cuecontext.New()

	instances := load.Instances([]string{"."}, &load.Config{Dir: modulePath})
	require.Len(t, instances, 1)
	require.NoError(t, instances[0].Err)

	schema := ctx.BuildInstance(instances[0])
	require.NoError(t, schema.Err())

	// Record Schema's concrete values before any FillPath.
	imageBefore, err := schema.LookupPath(cue.ParsePath("#components.web.spec.image")).String()
	require.NoError(t, err)
	replicasBefore, err := schema.LookupPath(cue.ParsePath("#components.web.spec.replicas")).Int64()
	require.NoError(t, err)

	// Produce Evaluated — the return value carries the overrides; schema must not change.
	userVals := ctx.CompileString(`{ image: "nginx:1.28.2", replicas: 3 }`)
	require.NoError(t, userVals.Err())
	evaluated := schema.FillPath(cue.MakePath(cue.Def("config")), userVals)
	require.NoError(t, evaluated.Err())

	// Confirm Evaluated has the user values (FillPath did something).
	evaluatedImage, err := evaluated.LookupPath(cue.ParsePath("#components.web.spec.image")).String()
	require.NoError(t, err)
	assert.Equal(t, "nginx:1.28.2", evaluatedImage, "Evaluated should carry user image")

	// Schema must report the same values it had before FillPath.
	imageAfter, err := schema.LookupPath(cue.ParsePath("#components.web.spec.image")).String()
	require.NoError(t, err)
	replicasAfter, err := schema.LookupPath(cue.ParsePath("#components.web.spec.replicas")).Int64()
	require.NoError(t, err)

	assert.Equal(t, imageBefore, imageAfter,
		"Schema.web.spec.image must be unchanged after FillPath")
	assert.Equal(t, replicasBefore, replicasAfter,
		"Schema.web.spec.replicas must be unchanged after FillPath")
	assert.Equal(t, "nginx:latest", imageAfter,
		"Schema should still carry module default image")
	assert.Equal(t, int64(1), replicasAfter,
		"Schema should still carry module default replicas")
}

// TestImmutability_UnifyDoesNotMutateSchema proves the same invariant for
// Unify. Calling Unify on Schema to merge in user values produces a new value
// and leaves Schema with its original defaults.
func TestImmutability_UnifyDoesNotMutateSchema(t *testing.T) {
	modulePath := testModulePath(t)
	ctx := cuecontext.New()

	instances := load.Instances([]string{"."}, &load.Config{Dir: modulePath})
	require.Len(t, instances, 1)
	require.NoError(t, instances[0].Err)

	schema := ctx.BuildInstance(instances[0])
	require.NoError(t, schema.Err())

	imageBefore, err := schema.LookupPath(cue.ParsePath("#config.image")).String()
	require.NoError(t, err)

	// Unify #config with new values — must not mutate schema.
	newConfig := ctx.CompileString(`{ image: "alpine:3.20", replicas: 2 }`)
	require.NoError(t, newConfig.Err())
	configSchema := schema.LookupPath(cue.MakePath(cue.Def("config")))
	unified := configSchema.Unify(newConfig)
	require.NoError(t, unified.Err())

	// Confirm the unified value has the new image (Unify did something).
	unifiedImage, err := unified.LookupPath(cue.ParsePath("image")).String()
	require.NoError(t, err)
	assert.Equal(t, "alpine:3.20", unifiedImage, "unified value should carry the new image")

	// Schema's #config.image is unchanged.
	imageAfter, err := schema.LookupPath(cue.ParsePath("#config.image")).String()
	require.NoError(t, err)
	assert.Equal(t, imageBefore, imageAfter,
		"Schema.#config.image must be unchanged after Unify on a sub-value")
}

// TestImmutability_SnapshotSchemaFieldUnchanged proves that holding Schema
// inside a moduleSnapshot struct provides no weaker guarantee. The struct stores
// a copy of the value type — Evaluated inside the same struct cannot affect Schema.
func TestImmutability_SnapshotSchemaFieldUnchanged(t *testing.T) {
	_, snap := buildSnapshot(t)

	// Schema in the snapshot should still have its module defaults.
	image, err := snap.Schema.LookupPath(cue.ParsePath("#components.web.spec.image")).String()
	require.NoError(t, err)
	assert.Equal(t, "nginx:latest", image,
		"snap.Schema should retain module default even though snap.Evaluated was FillPath'd")

	replicas, err := snap.Schema.LookupPath(cue.ParsePath("#components.web.spec.replicas")).Int64()
	require.NoError(t, err)
	assert.Equal(t, int64(1), replicas,
		"snap.Schema should retain module default replicas")
}

// TestImmutability_AssignedCopyIsIndependent proves that assigning a cue.Value
// to a new variable is a genuine independent copy. Producing a different
// FillPath from the copy does not affect the original struct field.
func TestImmutability_AssignedCopyIsIndependent(t *testing.T) {
	ctx, snap := buildSnapshot(t)

	// Take a copy of Schema by simple assignment — value type, no pointer involved.
	schemaCopy := snap.Schema

	// Produce a completely different fill from the copy — discard the result.
	otherVals := ctx.CompileString(`{ image: "busybox:1.0", replicas: 99 }`)
	require.NoError(t, otherVals.Err())
	_ = schemaCopy.FillPath(cue.MakePath(cue.Def("config")), otherVals)

	// snap.Schema is unaffected — still has module defaults.
	image, err := snap.Schema.LookupPath(cue.ParsePath("#components.web.spec.image")).String()
	require.NoError(t, err)
	assert.Equal(t, "nginx:latest", image,
		"snap.Schema must not be affected by FillPath on an assigned copy")

	// schemaCopy itself is also unaffected — FillPath returned a new value, not a mutation.
	copyImage, err := schemaCopy.LookupPath(cue.ParsePath("#components.web.spec.image")).String()
	require.NoError(t, err)
	assert.Equal(t, "nginx:latest", copyImage,
		"schemaCopy must also be unchanged — FillPath returned a new value, not a mutation")
}

// TestImmutability_MultipleChainedFillsDoNotAffectBase proves that calling
// FillPath multiple times — each starting from the same Schema — leaves Schema
// intact after all fills. Each call produces an independent new value.
func TestImmutability_MultipleChainedFillsDoNotAffectBase(t *testing.T) {
	ctx, snap := buildSnapshot(t)

	// Record Schema's image before any additional fills.
	baseImage, err := snap.Schema.LookupPath(cue.ParsePath("#components.web.spec.image")).String()
	require.NoError(t, err)

	// Produce three different fills from the same Schema.
	for _, img := range []string{"alpine:3.20", "busybox:1.36", "debian:12"} {
		img := img
		vals := ctx.CompileString(`{ image: "` + img + `", replicas: 1 }`)
		require.NoError(t, vals.Err())
		result := snap.Schema.FillPath(cue.MakePath(cue.Def("config")), vals)
		require.NoError(t, result.Err())

		// Each result carries its own image.
		resultImage, err := result.LookupPath(cue.ParsePath("#components.web.spec.image")).String()
		require.NoError(t, err)
		assert.Equal(t, img, resultImage, "FillPath result should carry image %q", img)
	}

	// Schema's image is still the original module default after all fills.
	afterImage, err := snap.Schema.LookupPath(cue.ParsePath("#components.web.spec.image")).String()
	require.NoError(t, err)
	assert.Equal(t, baseImage, afterImage,
		"Schema must be unchanged after all FillPath calls")
}
