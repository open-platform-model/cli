package build

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/opmodel/cli/internal/build/module"
)

// testdataDir returns the absolute path to a testdata fixture directory.
func testdataDir(t *testing.T, name string) string {
	t.Helper()
	dir, err := filepath.Abs(filepath.Join("testdata", name))
	require.NoError(t, err)
	return dir
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
	p := NewPipeline(nil, nil, "").(*pipeline)
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
	p := NewPipeline(nil, nil, "").(*pipeline)
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
	p := NewPipeline(nil, nil, "").(*pipeline)
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

// testdataFile returns the absolute path to a testdata file.
func testdataFile(t *testing.T, name string) string {
	t.Helper()
	f, err := filepath.Abs(filepath.Join("testdata", name))
	require.NoError(t, err)
	return f
}
