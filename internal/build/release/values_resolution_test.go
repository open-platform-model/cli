package release_test

import (
	"testing"

	"cuelang.org/go/cue/cuecontext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	buildmodule "github.com/opmodel/cli/internal/build/module"
	"github.com/opmodel/cli/internal/build/release"
)

// ----- Builder.Build() tests -----

func TestBuild_StubsValuesCue_WhenValuesFlagsProvided(t *testing.T) {
	// When valuesFiles are passed to Build() and values.cue exists on disk,
	// the external values take precedence (loaded via loadValuesFile).
	//
	// Uses test-module-values-only where values are ONLY in values.cue
	// (not duplicated in module.cue), so the module has no values.cue
	// pre-loaded — external files take full precedence.
	cueCtx := cuecontext.New()
	b := release.NewBuilder(cueCtx, "")
	dir := testModulePath(t, "test-module-values-only")
	valuesFile := testModulePath(t, "external-values.cue")

	mod, err := buildmodule.Load(cueCtx, dir, "")
	require.NoError(t, err, "module.Load should succeed")

	rel, err := b.Build(mod, release.Options{
		Name:      "test-release",
		Namespace: "default",
	}, []string{valuesFile})
	require.NoError(t, err)
	assert.NotNil(t, rel)
	assert.Equal(t, "test-release", rel.Metadata.Name)
	assert.Equal(t, "test-module-values-only", rel.Module.Metadata.Name)
	assert.Equal(t, "example.com/test-module-values-only@v0#test-module-values-only", rel.Module.Metadata.FQN)
	assert.Equal(t, "1.0.0", rel.Module.Metadata.Version)
	assert.Equal(t, "default", rel.Module.Metadata.DefaultNamespace)
}

func TestBuild_NoValuesCue_WithValuesFlag_Succeeds(t *testing.T) {
	// Build() should succeed for a module without values.cue when external
	// values files are provided.
	cueCtx := cuecontext.New()
	b := release.NewBuilder(cueCtx, "")
	dir := testModulePath(t, "test-module-no-values")
	valuesFile := testModulePath(t, "external-values.cue")

	mod, err := buildmodule.Load(cueCtx, dir, "")
	require.NoError(t, err, "module.Load should succeed")

	rel, err := b.Build(mod, release.Options{
		Name:      "test-release",
		Namespace: "default",
	}, []string{valuesFile})
	require.NoError(t, err)
	assert.NotNil(t, rel)
	assert.Equal(t, "test-release", rel.Metadata.Name)
	assert.Equal(t, "test-module-no-values", rel.Module.Metadata.Name)
	assert.Equal(t, "example.com/test-module-no-values@v0#test-module-no-values", rel.Module.Metadata.FQN)
	assert.Equal(t, "1.0.0", rel.Module.Metadata.Version)
	assert.Equal(t, "default", rel.Module.Metadata.DefaultNamespace)
}

func TestBuild_WithValuesCue_NoValuesFlag_Succeeds(t *testing.T) {
	// Build() with values.cue on disk and no --values flags should
	// work exactly as before (regression test).
	cueCtx := cuecontext.New()
	b := release.NewBuilder(cueCtx, "")
	dir := testModulePath(t, "test-module")

	mod, err := buildmodule.Load(cueCtx, dir, "")
	require.NoError(t, err, "module.Load should succeed")

	rel, err := b.Build(mod, release.Options{
		Name:      "test-release",
		Namespace: "default",
	}, nil)
	require.NoError(t, err)
	assert.NotNil(t, rel)
	assert.Equal(t, "test-release", rel.Metadata.Name)
	assert.Equal(t, "test-module", rel.Module.Metadata.Name)
	assert.Equal(t, "example.com/test-module@v0#test-module", rel.Module.Metadata.FQN)
	assert.Equal(t, "1.0.0", rel.Module.Metadata.Version)
	assert.Equal(t, "default", rel.Module.Metadata.DefaultNamespace)
}

func TestBuild_NonConcreteComponent_ReturnsError(t *testing.T) {
	// Build() must return an error if any component is not concrete after FillPath.
	// Provide only partial values (missing 'replicas') so #config.replicas remains
	// "int & >=1" and the web component is therefore not concrete.
	cueCtx := cuecontext.New()
	b := release.NewBuilder(cueCtx, "")
	dir := testModulePath(t, "test-module-no-values")
	partialValues := testModulePath(t, "partial-values.cue")

	mod, err := buildmodule.Load(cueCtx, dir, "")
	require.NoError(t, err)

	_, err = b.Build(mod, release.Options{
		Name:      "test-release",
		Namespace: "default",
	}, []string{partialValues})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not concrete after value injection")
	assert.Contains(t, err.Error(), "web")
}

func TestBuild_DeterministicReleaseUUID(t *testing.T) {
	// Build() should produce a deterministic release UUID based on
	// (fqn, name, namespace) via core.ComputeReleaseUUID.
	cueCtx := cuecontext.New()
	b := release.NewBuilder(cueCtx, "")
	dir := testModulePath(t, "test-module")

	mod, err := buildmodule.Load(cueCtx, dir, "")
	require.NoError(t, err)

	rel1, err := b.Build(mod, release.Options{Name: "my-release", Namespace: "prod"}, nil)
	require.NoError(t, err)

	rel2, err := b.Build(mod, release.Options{Name: "my-release", Namespace: "prod"}, nil)
	require.NoError(t, err)

	assert.Equal(t, rel1.Metadata.UUID, rel2.Metadata.UUID, "same inputs should produce same UUID")
	assert.NotEmpty(t, rel1.Metadata.UUID, "UUID should not be empty")
}
