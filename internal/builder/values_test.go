package builder

import (
	"os"
	"path/filepath"
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/opmodel/cli/internal/core/module"
	opmerrors "github.com/opmodel/cli/internal/errors"
)

// testdataPath returns the absolute path to a testdata file.
func testdataPath(t *testing.T, name string) string {
	t.Helper()
	p, err := filepath.Abs(filepath.Join("testdata", name))
	require.NoError(t, err)
	return p
}

// loaderFixturePath returns the absolute path to a loader testdata module.
// This allows builder tests to reuse the loader fixtures as real CUE module
// directories when testing values.cue discovery from mod.ModulePath.
func loaderFixturePath(t *testing.T, name string) string {
	t.Helper()
	p, err := filepath.Abs(filepath.Join("..", "loader", "testdata", name))
	require.NoError(t, err)
	return p
}

// moduleAtPath returns a *module.Module with ModulePath set to the given path.
func moduleAtPath(path string) *module.Module {
	return &module.Module{ModulePath: path}
}

func TestSelectValues(t *testing.T) {
	ctx := cuecontext.New()

	// -----------------------------------------------------------------------
	// Discovery from module directory (no --values provided)
	// -----------------------------------------------------------------------

	t.Run("no files, values.cue present in module dir returns its values", func(t *testing.T) {
		// test-module fixture has values.cue with image=nginx, replicas=2.
		mod := moduleAtPath(loaderFixturePath(t, "test-module"))

		got, err := selectValues(ctx, mod, nil)
		require.NoError(t, err)
		assert.True(t, got.Exists())

		img, err := got.LookupPath(cue.ParsePath("image.repository")).String()
		require.NoError(t, err)
		assert.Equal(t, "nginx", img)
	})

	t.Run("no files, no values.cue in module dir returns ValidationError", func(t *testing.T) {
		// Create a temp dir with cue.mod to satisfy ModulePath (no values.cue).
		dir := t.TempDir()
		require.NoError(t, os.MkdirAll(filepath.Join(dir, "cue.mod"), 0o755))

		mod := moduleAtPath(dir)

		_, err := selectValues(ctx, mod, nil)
		require.Error(t, err)

		var valErr *opmerrors.ValidationError
		require.ErrorAs(t, err, &valErr)
		assert.Contains(t, valErr.Message, "values.cue")
	})

	// -----------------------------------------------------------------------
	// --values files provided
	// -----------------------------------------------------------------------

	t.Run("one file provided extracts values field", func(t *testing.T) {
		mod := moduleAtPath(t.TempDir()) // ModulePath unused when --values provided
		file := testdataPath(t, "values-a.cue")

		got, err := selectValues(ctx, mod, []string{file})
		require.NoError(t, err)
		assert.True(t, got.Exists())

		img, err := got.LookupPath(cue.ParsePath("image")).String()
		require.NoError(t, err)
		assert.Equal(t, "nginx:1.0", img)

		replicas, err := got.LookupPath(cue.ParsePath("replicas")).Int64()
		require.NoError(t, err)
		assert.Equal(t, int64(1), replicas)
	})

	t.Run("two files unified, result contains fields from both", func(t *testing.T) {
		mod := moduleAtPath(t.TempDir())
		fileA := testdataPath(t, "values-a.cue")
		fileB := testdataPath(t, "values-b.cue")

		// values-a: image=nginx:1.0, replicas=1
		// values-b: image=nginx:1.0, replicas=1, port=8080 (adds a new field)
		// Unify produces the conjunction of both — all fields present.
		got, err := selectValues(ctx, mod, []string{fileA, fileB})
		require.NoError(t, err)
		assert.True(t, got.Exists())

		img, err := got.LookupPath(cue.ParsePath("image")).String()
		require.NoError(t, err)
		assert.Equal(t, "nginx:1.0", img)

		port, err := got.LookupPath(cue.ParsePath("port")).Int64()
		require.NoError(t, err)
		assert.Equal(t, int64(8080), port)
	})

	t.Run("--values ignores values.cue in module dir", func(t *testing.T) {
		// test-module has values.cue with image=nginx — we override with values-a.cue.
		// The result must come from values-a.cue only, not from values.cue.
		mod := moduleAtPath(loaderFixturePath(t, "test-module"))
		file := testdataPath(t, "values-a.cue")

		got, err := selectValues(ctx, mod, []string{file})
		require.NoError(t, err)
		assert.True(t, got.Exists())

		// values-a.cue has image="nginx:1.0" at the root "image" key (not image.repository).
		img, err := got.LookupPath(cue.ParsePath("image")).String()
		require.NoError(t, err)
		assert.Equal(t, "nginx:1.0", img, "--values must be the sole source, values.cue must be ignored")
	})

	t.Run("file with no values field returns ValidationError", func(t *testing.T) {
		mod := moduleAtPath(t.TempDir())
		file := testdataPath(t, "values-no-field.cue")

		_, err := selectValues(ctx, mod, []string{file})
		require.Error(t, err)

		var valErr *opmerrors.ValidationError
		require.ErrorAs(t, err, &valErr)
		assert.Contains(t, valErr.Message, "'values'")
	})
}
