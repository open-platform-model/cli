package build

import (
	"os"
	"path/filepath"
	"testing"

	"cuelang.org/go/cue/cuecontext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/opmodel/cli/internal/build/module"
	"github.com/opmodel/cli/internal/config"
)

// testdataDir returns the absolute path to a testdata fixture directory.
func testdataDir(t *testing.T, name string) string {
	t.Helper()
	dir, err := filepath.Abs(filepath.Join("testdata", name))
	require.NoError(t, err)
	return dir
}

// testdataFile returns the absolute path to a testdata file.
func testdataFile(t *testing.T, name string) string {
	t.Helper()
	f, err := filepath.Abs(filepath.Join("testdata", name))
	require.NoError(t, err)
	return f
}

// ----- module.ResolvePath tests -----

func TestResolveModulePath_WithoutValuesCue(t *testing.T) {
	// A module directory without values.cue should be accepted by
	// ResolvePath — values.cue existence is no longer its concern.
	dir := testdataDir(t, "test-module-no-values")

	got, err := module.ResolvePath(dir)
	require.NoError(t, err)
	assert.Equal(t, dir, got)
}

func TestResolveModulePath_WithValuesCue(t *testing.T) {
	// A module directory WITH values.cue should still work fine.
	dir := testdataDir(t, "test-module")

	got, err := module.ResolvePath(dir)
	require.NoError(t, err)
	assert.Equal(t, dir, got)
}

func TestResolveModulePath_MissingDirectory(t *testing.T) {
	_, err := module.ResolvePath("/nonexistent/path")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "module directory not found")
}

func TestResolveModulePath_MissingCueMod(t *testing.T) {
	// A directory that exists but has no cue.mod/
	dir := t.TempDir()

	_, err := module.ResolvePath(dir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing cue.mod/")
}

// ----- Render() values.cue conditional check tests -----

func TestRender_NoValuesCue_NoValuesFlag_ReturnsError(t *testing.T) {
	// When no --values flags are provided AND values.cue is missing on disk,
	// Render should fail with a clear error message.
	cueCtx := cuecontext.New()
	cfg := &config.OPMConfig{
		CueContext: cueCtx,
		Registry:   "",
	}
	p := NewPipeline(cfg).(*pipeline)
	dir := testdataDir(t, "test-module-no-values")

	_, err := p.Render(t.Context(), RenderOptions{
		ModulePath: dir,
		Name:       "test",
		Namespace:  "default",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "values.cue not found")
	assert.Contains(t, err.Error(), "--values flag")
}

func TestRender_NoValuesCue_WithValuesFlag_SkipsValuesCueCheck(t *testing.T) {
	// When --values flags ARE provided, the values.cue existence check
	// should be skipped. The pipeline may fail later (e.g., no providers),
	// but it must NOT fail with "values.cue not found".
	cueCtx := cuecontext.New()
	cfg := &config.OPMConfig{
		CueContext: cueCtx,
		Registry:   "",
	}
	p := NewPipeline(cfg).(*pipeline)
	dir := testdataDir(t, "test-module-no-values")
	valuesFile := testdataFile(t, "external-values.cue")

	_, err := p.Render(t.Context(), RenderOptions{
		ModulePath: dir,
		Name:       "test-release",
		Namespace:  "default",
		Values:     []string{valuesFile},
	})
	// The pipeline may fail at provider loading (expected in unit tests),
	// but it must NOT fail with "values.cue not found".
	if err != nil {
		assert.NotContains(t, err.Error(), "values.cue not found",
			"should not require values.cue when --values flag is provided")
	}
}

func TestRender_WithValuesCue_NoValuesFlag_SkipsValuesCueCheck(t *testing.T) {
	// When values.cue exists and no --values flags: should pass the values
	// check and proceed. May fail later (e.g., no providers).
	cueCtx := cuecontext.New()
	cfg := &config.OPMConfig{
		CueContext: cueCtx,
		Registry:   "",
	}
	p := NewPipeline(cfg).(*pipeline)
	dir := testdataDir(t, "test-module")

	_, err := p.Render(t.Context(), RenderOptions{
		ModulePath: dir,
		Name:       "test-release",
		Namespace:  "default",
	})
	// Should not fail with "values.cue not found"
	if err != nil {
		assert.NotContains(t, err.Error(), "values.cue not found")
	}
}

// ----- ReleaseBuilder.Build() tests -----

func TestBuild_StubsValuesCue_WhenValuesFlagsProvided(t *testing.T) {
	// When valuesFiles are passed to Build() and values.cue exists on disk,
	// the on-disk values.cue should be replaced with a stub so the external
	// values take precedence without conflict.
	//
	// Uses test-module-values-only where values are ONLY in values.cue
	// (not duplicated in module.cue), so the stub effectively removes them.
	cueCtx := cuecontext.New()
	b := NewReleaseBuilder(cueCtx, "")
	dir := testdataDir(t, "test-module-values-only")
	valuesFile := testdataFile(t, "external-values.cue")

	// Verify values.cue exists on disk (precondition)
	_, err := os.Stat(filepath.Join(dir, "values.cue"))
	require.NoError(t, err, "precondition: values.cue should exist on disk")

	release, err := b.Build(dir, ReleaseOptions{
		Name:      "test-release",
		Namespace: "default",
		PkgName:   "testmodule",
	}, []string{valuesFile})
	require.NoError(t, err)
	assert.NotNil(t, release)
	assert.Equal(t, "test-release", release.Metadata.Name)
}

func TestBuild_NoValuesCue_WithValuesFlag_Succeeds(t *testing.T) {
	// Build() should succeed for a module without values.cue when external
	// values files are provided.
	cueCtx := cuecontext.New()
	b := NewReleaseBuilder(cueCtx, "")
	dir := testdataDir(t, "test-module-no-values")
	valuesFile := testdataFile(t, "external-values.cue")

	release, err := b.Build(dir, ReleaseOptions{
		Name:      "test-release",
		Namespace: "default",
		PkgName:   "testmodule",
	}, []string{valuesFile})
	require.NoError(t, err)
	assert.NotNil(t, release)
	assert.Equal(t, "test-release", release.Metadata.Name)
}

func TestBuild_WithValuesCue_NoValuesFlag_Succeeds(t *testing.T) {
	// Build() with values.cue on disk and no --values flags should
	// work exactly as before (regression test).
	cueCtx := cuecontext.New()
	b := NewReleaseBuilder(cueCtx, "")
	dir := testdataDir(t, "test-module")

	release, err := b.Build(dir, ReleaseOptions{
		Name:      "test-release",
		Namespace: "default",
		PkgName:   "testmodule",
	}, nil)
	require.NoError(t, err)
	assert.NotNil(t, release)
	assert.Equal(t, "test-release", release.Metadata.Name)
}
