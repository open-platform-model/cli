package release

import (
	"os"
	"path/filepath"
	"testing"

	"cuelang.org/go/cue/cuecontext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testdataDir returns the absolute path to a testdata fixture directory.
// Points to the testdata directory in the parent build package.
func testdataDir(t *testing.T, name string) string {
	t.Helper()
	return testModulePath(t, name)
}

// testdataFile returns the absolute path to a testdata file.
// Points to a file in the testdata directory of the parent build package.
func testdataFile(t *testing.T, name string) string {
	t.Helper()
	return testModulePath(t, name)
}

// ----- Builder.Build() tests -----

func TestBuild_StubsValuesCue_WhenValuesFlagsProvided(t *testing.T) {
	// When valuesFiles are passed to Build() and values.cue exists on disk,
	// the on-disk values.cue should be replaced with a stub so the external
	// values take precedence without conflict.
	//
	// Uses test-module-values-only where values are ONLY in values.cue
	// (not duplicated in module.cue), so the stub effectively removes them.
	cueCtx := cuecontext.New()
	b := NewBuilder(cueCtx, "")
	dir := testdataDir(t, "test-module-values-only")
	valuesFile := testdataFile(t, "external-values.cue")

	// Verify values.cue exists on disk (precondition)
	_, err := os.Stat(filepath.Join(dir, "values.cue"))
	require.NoError(t, err, "precondition: values.cue should exist on disk")

	rel, err := b.Build(dir, Options{
		Name:      "test-release",
		Namespace: "default",
		PkgName:   "testmodule",
	}, []string{valuesFile})
	require.NoError(t, err)
	assert.NotNil(t, rel)
	assert.Equal(t, "test-release", rel.ReleaseMetadata.Name)
	assert.Equal(t, "test-module-values-only", rel.ModuleMetadata.Name)
	assert.Equal(t, "example.com/test-module-values-only@v0#test-module-values-only", rel.ModuleMetadata.FQN)
	assert.Equal(t, "1.0.0", rel.ModuleMetadata.Version)
	assert.Equal(t, "default", rel.ModuleMetadata.DefaultNamespace)
}

func TestBuild_NoValuesCue_WithValuesFlag_Succeeds(t *testing.T) {
	// Build() should succeed for a module without values.cue when external
	// values files are provided.
	cueCtx := cuecontext.New()
	b := NewBuilder(cueCtx, "")
	dir := testdataDir(t, "test-module-no-values")
	valuesFile := testdataFile(t, "external-values.cue")

	rel, err := b.Build(dir, Options{
		Name:      "test-release",
		Namespace: "default",
		PkgName:   "testmodule",
	}, []string{valuesFile})
	require.NoError(t, err)
	assert.NotNil(t, rel)
	assert.Equal(t, "test-release", rel.ReleaseMetadata.Name)
	assert.Equal(t, "test-module-no-values", rel.ModuleMetadata.Name)
	assert.Equal(t, "example.com/test-module-no-values@v0#test-module-no-values", rel.ModuleMetadata.FQN)
	assert.Equal(t, "1.0.0", rel.ModuleMetadata.Version)
	assert.Equal(t, "default", rel.ModuleMetadata.DefaultNamespace)
}

func TestBuild_WithValuesCue_NoValuesFlag_Succeeds(t *testing.T) {
	// Build() with values.cue on disk and no --values flags should
	// work exactly as before (regression test).
	cueCtx := cuecontext.New()
	b := NewBuilder(cueCtx, "")
	dir := testdataDir(t, "test-module")

	rel, err := b.Build(dir, Options{
		Name:      "test-release",
		Namespace: "default",
		PkgName:   "testmodule",
	}, nil)
	require.NoError(t, err)
	assert.NotNil(t, rel)
	assert.Equal(t, "test-release", rel.ReleaseMetadata.Name)
	assert.Equal(t, "test-module", rel.ModuleMetadata.Name)
	assert.Equal(t, "example.com/test-module@v0#test-module", rel.ModuleMetadata.FQN)
	assert.Equal(t, "1.0.0", rel.ModuleMetadata.Version)
	assert.Equal(t, "default", rel.ModuleMetadata.DefaultNamespace)
}
