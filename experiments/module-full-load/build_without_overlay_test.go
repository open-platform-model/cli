package modulefullload

// ---------------------------------------------------------------------------
// Decision 7: release.Builder.Build() uses mod.CUEValue() — no overlay needed
//
// The overlay's only purpose was to inject #opmReleaseMeta (UUID + labels)
// into the CUE namespace so CUE could compute identity. With UUID computation
// moved to Go (Decision 8) and module metadata read from the evaluation
// result (Decision 9), the overlay has no remaining function.
//
// The build phase now does:
//   base  = mod.CUEValue()                          // from Load()
//   filled = base.FillPath("#config", userValues)   // apply user values
//   concreteComponents = ExtractComponents(filled.LookupPath("#components"))
//
// These tests prove:
//   - FillPath("#config", userValues) produces a concrete module value
//   - The original base value is unchanged after FillPath (immutable)
//   - The base value can be reused for multiple independent releases
//   - Concrete components pass IsConcrete() after FillPath
//   - Invalid user values are rejected during Unify/Validate (no overlay needed)
//   - Loading a --values file and merging it into the base value works
// ---------------------------------------------------------------------------

import (
	"os"
	"path/filepath"
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/load"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBuildNoOverlay_FillConfigProducesConcrete proves that FillPath("#config",
// userValues) produces a concrete component spec — the core of the design.
func TestBuildNoOverlay_FillConfigProducesConcrete(t *testing.T) {
	ctx, base := buildBaseValue(t)

	filled := applyValues(t, ctx, base, `{
		image:    "nginx:1.0"
		replicas: 3
		port:     9090
		debug:    false
	}`)

	// After FillPath, web.spec.image should be the concrete value "nginx:1.0".
	webSpecImage, err := filled.LookupPath(cue.ParsePath("#components.web.spec.image")).String()
	require.NoError(t, err, "spec.image should be a concrete string after FillPath")
	assert.Equal(t, "nginx:1.0", webSpecImage)

	webSpecReplicas, err := filled.LookupPath(cue.ParsePath("#components.web.spec.replicas")).Int64()
	require.NoError(t, err, "spec.replicas should be a concrete int after FillPath")
	assert.Equal(t, int64(3), webSpecReplicas)

	// The full filled value should be concrete (all required fields satisfied).
	// Note: we check #components only — the full value includes #config constraints.
	webSpec := filled.LookupPath(cue.ParsePath("#components.web.spec"))
	assert.NoError(t, webSpec.Validate(cue.Concrete(true)),
		"web.spec should be fully concrete after FillPath")
}

// TestBuildNoOverlay_BaseValueUnchanged proves that FillPath does not mutate
// the base value. This is the immutability guarantee that makes mod.CUEValue()
// reusable across multiple Build() calls.
//
// With full #config defaults, the base value already has concrete spec fields
// (the schema defaults). The test verifies that after FillPath with a different
// image, the base still holds the original schema default — not the user value.
func TestBuildNoOverlay_BaseValueUnchanged(t *testing.T) {
	ctx, base := buildBaseValue(t)

	// Before: spec.image resolves to the schema default "nginx:latest".
	beforeImage, err := base.LookupPath(cue.ParsePath("#components.web.spec.image")).String()
	require.NoError(t, err)
	assert.Equal(t, "nginx:latest", beforeImage, "spec.image should be the schema default before FillPath")

	// Apply values with a different image — produces a new value, does not touch base.
	filled := applyValues(t, ctx, base, `{
		image:    "nginx:mutate-test"
		replicas: 1
		port:     8080
		debug:    false
	}`)

	// Filled value has the user image.
	filledImage, err := filled.LookupPath(cue.ParsePath("#components.web.spec.image")).String()
	require.NoError(t, err)
	assert.Equal(t, "nginx:mutate-test", filledImage, "filled value should have the user image")

	// Base is unchanged — still the schema default.
	afterImage, err := base.LookupPath(cue.ParsePath("#components.web.spec.image")).String()
	require.NoError(t, err)
	assert.Equal(t, "nginx:latest", afterImage, "FillPath must not mutate the base value")
}

// TestBuildNoOverlay_MultipleReleases proves that the same base value can be
// used for multiple independent FillPath calls — producing different concrete
// releases from the same module. This is the core of Decision 2 (mod.CUEValue()
// is reusable) and Decision 7 (no overlay per-release).
func TestBuildNoOverlay_MultipleReleases(t *testing.T) {
	ctx, base := buildBaseValue(t)

	releaseA := applyValues(t, ctx, base, `{
		image:    "nginx:1.0"
		replicas: 1
		port:     8080
		debug:    false
	}`)

	releaseB := applyValues(t, ctx, base, `{
		image:    "nginx:2.0"
		replicas: 5
		port:     9090
		debug:    true
	}`)

	imageA, err := releaseA.LookupPath(cue.ParsePath("#components.web.spec.image")).String()
	require.NoError(t, err)

	imageB, err := releaseB.LookupPath(cue.ParsePath("#components.web.spec.image")).String()
	require.NoError(t, err)

	assert.Equal(t, "nginx:1.0", imageA, "release A should have its own image")
	assert.Equal(t, "nginx:2.0", imageB, "release B should have its own image")
	assert.NotEqual(t, imageA, imageB, "releases must be independent")

	replicasA, _ := releaseA.LookupPath(cue.ParsePath("#components.web.spec.replicas")).Int64()
	replicasB, _ := releaseB.LookupPath(cue.ParsePath("#components.web.spec.replicas")).Int64()
	assert.Equal(t, int64(1), replicasA)
	assert.Equal(t, int64(5), replicasB)
}

// TestBuildNoOverlay_ComponentsConcreteAfterFill proves that all components
// pass IsConcrete() after FillPath — required for the executor phase.
func TestBuildNoOverlay_ComponentsConcreteAfterFill(t *testing.T) {
	ctx, base := buildBaseValue(t)

	filled := applyValues(t, ctx, base, `{
		image:    "app:latest"
		replicas: 2
		port:     8080
		debug:    false
	}`)

	componentsVal := filled.LookupPath(cue.MakePath(cue.Def("components")))
	comps := extractComponents(t, componentsVal)
	require.NotEmpty(t, comps)

	for name, comp := range comps {
		assert.True(t, comp.isConcrete(),
			"component %q should be concrete after FillPath", name)
	}
}

// TestBuildNoOverlay_ValuesValidation proves that invalid user values are
// rejected during the Unify step — before FillPath — with a clear error.
// This is equivalent to validateValuesAgainstConfig in the current pipeline.
func TestBuildNoOverlay_ValuesValidation(t *testing.T) {
	_, base := buildBaseValue(t)
	ctx := base.Context()

	// replicas must be int & >=1, but we pass a string.
	badVals := ctx.CompileString(`{
		image:    "nginx:1.0"
		replicas: "three"
		port:     8080
		debug:    false
	}`)
	require.NoError(t, badVals.Err())

	configSchema := base.LookupPath(cue.MakePath(cue.Def("config")))
	require.True(t, configSchema.Exists())

	// Unification with the schema catches the type mismatch.
	unified := configSchema.Unify(badVals)
	err := unified.Validate()
	assert.Error(t, err, "string value for int field should fail validation")

	// Omitting image is fine — #config.image has a default (*"nginx:latest"),
	// so partial values are valid. The schema default fills the gap.
	partialVals := ctx.CompileString(`{
		replicas: 2
	}`)
	unifiedPartial := configSchema.Unify(partialVals)
	assert.NoError(t, unifiedPartial.Validate(cue.Concrete(true)),
		"partial values should be valid when all omitted fields have schema defaults")
}

// TestBuildNoOverlay_ValuesFileMerge simulates the --values file flow:
// load a values file from disk, compile it, unify with the base, FillPath.
// This exercises the loadValuesFile() path without an overlay.
func TestBuildNoOverlay_ValuesFileMerge(t *testing.T) {
	modulePath := testModulePath(t)

	// Write a temporary values file.
	valuesContent := `{
		image:    "from-file:3.0"
		replicas: 4
		port:     7070
		debug:    true
	}`
	tmpFile := filepath.Join(t.TempDir(), "override.cue")
	require.NoError(t, os.WriteFile(tmpFile, []byte(valuesContent), 0o600))

	// Load the base value.
	ctx := cuecontext.New()
	instances := load.Instances([]string{"."}, &load.Config{Dir: modulePath})
	require.Len(t, instances, 1)
	require.NoError(t, instances[0].Err)
	base := ctx.BuildInstance(instances[0])
	require.NoError(t, base.Err())

	// Load the values file as bytes and compile.
	content, err := os.ReadFile(tmpFile)
	require.NoError(t, err)
	fileVals := ctx.CompileBytes(content, cue.Filename(tmpFile))
	require.NoError(t, fileVals.Err())

	// FillPath with file-loaded values.
	filled := base.FillPath(cue.MakePath(cue.Def("config")), fileVals)
	require.NoError(t, filled.Err())

	image, err := filled.LookupPath(cue.ParsePath("#components.web.spec.image")).String()
	require.NoError(t, err)
	assert.Equal(t, "from-file:3.0", image, "values from file should be applied")

	replicas, err := filled.LookupPath(cue.ParsePath("#components.web.spec.replicas")).Int64()
	require.NoError(t, err)
	assert.Equal(t, int64(4), replicas)
}

// TestBuildNoOverlay_WorkerImageFromConfig proves that worker (which hardcodes
// replicas: 1 but references #config.image) correctly picks up the user image.
func TestBuildNoOverlay_WorkerImageFromConfig(t *testing.T) {
	ctx, base := buildBaseValue(t)

	filled := applyValues(t, ctx, base, `{
		image:    "worker-image:v2"
		replicas: 1
		port:     8080
		debug:    false
	}`)

	workerImage, err := filled.LookupPath(cue.ParsePath("#components.worker.spec.image")).String()
	require.NoError(t, err)
	assert.Equal(t, "worker-image:v2", workerImage,
		"worker.spec.image should reflect #config.image from user values")

	workerReplicas, err := filled.LookupPath(cue.ParsePath("#components.worker.spec.replicas")).Int64()
	require.NoError(t, err)
	assert.Equal(t, int64(1), workerReplicas,
		"worker.spec.replicas is hardcoded to 1, not from #config")
}
