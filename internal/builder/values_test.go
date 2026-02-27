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

// moduleWithValuesFile creates a temp directory containing a values.cue file with the
// given CUE content and returns its path.
func moduleWithValuesFile(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "values.cue")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
	return dir
}

// moduleAtPath returns a *module.Module with ModulePath set to the given path.
func moduleAtPath(path string) *module.Module {
	return &module.Module{ModulePath: path}
}

func TestSelectValues(t *testing.T) {
	ctx := cuecontext.New()

	// -----------------------------------------------------------------------
	// Fallback: no --values provided
	// -----------------------------------------------------------------------

	t.Run("no --values, values.cue present used as fallback values file", func(t *testing.T) {
		// values.cue is treated as a regular values file — no semantic distinction
		// from an explicit --values argument. Defaults live in #config, not here.
		dir := moduleWithValuesFile(t, `values: {
			image: { repository: "nginx", tag: "1.25", digest: "" }
			replicas: 2
		}`)
		mod := moduleAtPath(dir)

		got, err := selectValues(ctx, mod, nil)
		require.NoError(t, err)
		assert.True(t, got.Exists())

		img, err := got.LookupPath(cue.ParsePath("image.repository")).String()
		require.NoError(t, err)
		assert.Equal(t, "nginx", img)
	})

	t.Run("no --values, no values.cue present returns ValidationError", func(t *testing.T) {
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
	// Explicit --values files provided
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
		// When --values is provided explicitly, values.cue in the module directory
		// must NOT be read. Only the explicit files are used.
		dir := moduleWithValuesFile(t, `values: {
			image: { repository: "should-be-ignored", tag: "latest", digest: "" }
			replicas: 99
		}`)
		mod := moduleAtPath(dir)
		file := testdataPath(t, "values-a.cue")

		got, err := selectValues(ctx, mod, []string{file})
		require.NoError(t, err)
		assert.True(t, got.Exists())

		// values-a.cue has image="nginx:1.0" at the root "image" key (not image.repository).
		img, err := got.LookupPath(cue.ParsePath("image")).String()
		require.NoError(t, err)
		assert.Equal(t, "nginx:1.0", img, "--values must be the sole source; values.cue must be ignored")
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
