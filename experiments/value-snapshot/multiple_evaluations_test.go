package valuesnapshot

// ---------------------------------------------------------------------------
// Multiple evaluations: one Schema → N independent Evaluated values
//
// Because FillPath is pure and cue.Value is a value type, the same Schema
// can serve as the base for any number of distinct Evaluated values. Each
// one carries different user values; all coexist simultaneously without
// interfering with each other or with Schema.
//
// This maps directly to a real use case: the same module loaded once at
// startup, then evaluated per-release with different end-user value files.
//
// These tests prove:
//   - A slice of moduleSnapshot values works correctly, each with own Evaluated
//   - All snapshots built from the same Schema agree on Schema-level fields
//   - Different user values produce observably different Evaluated fields
//   - The Schema field is value-equal across all snapshots (same immutable node)
//
// Reference: docs/design/cue-in-go.md "Divergent copies via operations"
// (cue-in-go.md:206-212)
// ---------------------------------------------------------------------------

import (
	"testing"

	"cuelang.org/go/cue"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMultiEval_TwoDifferentUserValues proves that two moduleSnapshot values,
// each built from the same underlying Schema but with different user inputs,
// coexist with independent Evaluated fields.
func TestMultiEval_TwoDifferentUserValues(t *testing.T) {
	ctx, _ := buildSnapshot(t)

	snapA := buildSnapshotWithValues(t, ctx, `{ image: "nginx:1.28.2", replicas: 3 }`)
	snapB := buildSnapshotWithValues(t, ctx, `{ image: "nginx:1.28.2", replicas: 5 }`)

	// Evaluated fields differ between snapshots.
	replicasA, err := snapA.Evaluated.LookupPath(cue.ParsePath("#components.web.spec.replicas")).Int64()
	require.NoError(t, err)
	replicasB, err := snapB.Evaluated.LookupPath(cue.ParsePath("#components.web.spec.replicas")).Int64()
	require.NoError(t, err)

	assert.Equal(t, int64(3), replicasA, "snapA should have replicas=3")
	assert.Equal(t, int64(5), replicasB, "snapB should have replicas=5")
	assert.NotEqual(t, replicasA, replicasB, "the two Evaluated values must differ")

	// Schema fields are identical in both snapshots (same module defaults).
	schemaImageA, err := snapA.Schema.LookupPath(cue.ParsePath("#components.web.spec.image")).String()
	require.NoError(t, err)
	schemaImageB, err := snapB.Schema.LookupPath(cue.ParsePath("#components.web.spec.image")).String()
	require.NoError(t, err)

	assert.Equal(t, schemaImageA, schemaImageB,
		"Schema field must be identical across all snapshots")
	assert.Equal(t, "nginx:latest", schemaImageA)
}

// TestMultiEval_SnapshotSlice proves that a []moduleSnapshot works correctly.
// Each element holds an independent Evaluated value derived from the same Schema.
func TestMultiEval_SnapshotSlice(t *testing.T) {
	ctx, _ := buildSnapshot(t)

	userInputs := []struct {
		image    string
		replicas string
	}{
		{"alpine:3.20", "1"},
		{"nginx:1.28.2", "3"},
		{"debian:12", "5"},
	}

	snaps := make([]moduleSnapshot, len(userInputs))
	for i, input := range userInputs {
		snaps[i] = buildSnapshotWithValues(t, ctx,
			`{ image: "`+input.image+`", replicas: `+input.replicas+` }`)
	}

	// Each snapshot's Evaluated carries the expected overrides.
	for i, input := range userInputs {
		snap := snaps[i]

		image, err := snap.Evaluated.LookupPath(cue.ParsePath("#components.web.spec.image")).String()
		require.NoError(t, err)
		assert.Equal(t, input.image, image,
			"snaps[%d].Evaluated should carry image %q", i, input.image)

		// Schema in every element still has module defaults.
		schemaImage, err := snap.Schema.LookupPath(cue.ParsePath("#components.web.spec.image")).String()
		require.NoError(t, err)
		assert.Equal(t, "nginx:latest", schemaImage,
			"snaps[%d].Schema must retain module default", i)
	}
}

// TestMultiEval_AllEvaluatedValuesDistinct proves that every Evaluated field
// in the slice is observably different when built with distinct user images.
func TestMultiEval_AllEvaluatedValuesDistinct(t *testing.T) {
	ctx, _ := buildSnapshot(t)

	images := []string{"alpine:3.20", "nginx:1.28.2", "debian:12"}
	snaps := make([]moduleSnapshot, len(images))
	for i, img := range images {
		snaps[i] = buildSnapshotWithValues(t, ctx, `{ image: "`+img+`", replicas: 1 }`)
	}

	seen := map[string]bool{}
	for i, snap := range snaps {
		img, err := snap.Evaluated.LookupPath(cue.ParsePath("#components.web.spec.image")).String()
		require.NoError(t, err)
		assert.Equal(t, images[i], img)
		assert.False(t, seen[img], "each Evaluated image should be unique in the slice")
		seen[img] = true
	}
	assert.Len(t, seen, len(images), "all Evaluated images should be distinct")
}

// TestMultiEval_WorkerUsesConfigImage proves that #components.worker also
// reflects the per-snapshot user image, not just web. Both components
// reference #config.image so every Evaluated snapshot propagates the override
// to both — worker in one snapshot is independent from worker in another.
func TestMultiEval_WorkerUsesConfigImage(t *testing.T) {
	ctx, _ := buildSnapshot(t)

	snapA := buildSnapshotWithValues(t, ctx, `{ image: "alpine:3.20", replicas: 1 }`)
	snapB := buildSnapshotWithValues(t, ctx, `{ image: "debian:12", replicas: 1 }`)

	workerImageA, err := snapA.Evaluated.LookupPath(cue.ParsePath("#components.worker.spec.image")).String()
	require.NoError(t, err)
	workerImageB, err := snapB.Evaluated.LookupPath(cue.ParsePath("#components.worker.spec.image")).String()
	require.NoError(t, err)

	assert.Equal(t, "alpine:3.20", workerImageA, "snapA worker should use alpine")
	assert.Equal(t, "debian:12", workerImageB, "snapB worker should use debian")
	assert.NotEqual(t, workerImageA, workerImageB,
		"worker images must differ across snapshots built with different user values")
}

// TestMultiEval_SchemaInvariantAcrossAllSnapshots proves that the Schema field
// in every snapshot, regardless of how many were built or what user values were
// applied, always reads the same module-default image. Schema is the stable
// "source of truth" that all Evaluated values are derived from.
func TestMultiEval_SchemaInvariantAcrossAllSnapshots(t *testing.T) {
	ctx, _ := buildSnapshot(t)

	valueSets := []string{
		`{ image: "alpine:3.20",   replicas: 1 }`,
		`{ image: "nginx:1.28.2",  replicas: 3 }`,
		`{ image: "debian:12",     replicas: 5 }`,
		`{ image: "busybox:1.36",  replicas: 7 }`,
	}

	for i, vals := range valueSets {
		snap := buildSnapshotWithValues(t, ctx, vals)

		schemaImage, err := snap.Schema.LookupPath(cue.ParsePath("#components.web.spec.image")).String()
		require.NoError(t, err)
		assert.Equal(t, "nginx:latest", schemaImage,
			"snapshot[%d].Schema must always report module default image", i)

		schemaReplicas, err := snap.Schema.LookupPath(cue.ParsePath("#components.web.spec.replicas")).Int64()
		require.NoError(t, err)
		assert.Equal(t, int64(1), schemaReplicas,
			"snapshot[%d].Schema must always report module default replicas", i)
	}
}
