package release_test

import (
	"path/filepath"
	"runtime"
	"testing"

	"cuelang.org/go/cue/cuecontext"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	buildmodule "github.com/opmodel/cli/internal/build/module"
)

// testModulePath returns the absolute path to a test module fixture.
// Points to the testdata directory in the parent build package.
func testModulePath(t *testing.T, name string) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("unable to determine test file path")
	}
	// Go up one level from release/ to build/ then into testdata/
	return filepath.Join(filepath.Dir(filename), "..", "testdata", name)
}

// ---------------------------------------------------------------------------
// TestLoad_StaticMetadata
// ---------------------------------------------------------------------------

func TestLoad_StaticMetadata(t *testing.T) {
	ctx := cuecontext.New()

	modulePath := testModulePath(t, "test-module")
	mod, err := buildmodule.Load(ctx, modulePath, "")
	require.NoError(t, err)

	assert.Equal(t, "test-module", mod.Metadata.Name)
	assert.Equal(t, "default", mod.Metadata.DefaultNamespace)
	assert.Equal(t, "testmodule", mod.PkgName())
}

// ---------------------------------------------------------------------------
// TestLoad_MissingMetadata
// ---------------------------------------------------------------------------

func TestLoad_MissingMetadata(t *testing.T) {
	ctx := cuecontext.New()

	modulePath := testModulePath(t, "no-metadata-module")
	mod, err := buildmodule.Load(ctx, modulePath, "")
	require.NoError(t, err)

	// metadata.name is a computed expression; CUE evaluation resolves it to "computed-module"
	assert.Equal(t, "computed-module", mod.Metadata.Name)
	assert.Equal(t, "nometadata", mod.PkgName())
}
