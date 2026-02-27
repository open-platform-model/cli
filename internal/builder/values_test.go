package builder

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	opmerrors "github.com/opmodel/cli/internal/errors"
)

// testdataPath returns the absolute path to a testdata file.
func testdataPath(t *testing.T, name string) string {
	t.Helper()
	p, err := filepath.Abs(filepath.Join("testdata", name))
	require.NoError(t, err)
	return p
}

// writeTempCUE writes a CUE file to a temp directory and returns its path.
func writeTempCUE(t *testing.T, name, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
	return path
}

// tempDirWithValuesFile creates a temp directory containing values.cue with the
// given content and returns the directory path.
func tempDirWithValuesFile(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "values.cue"), []byte(content), 0o644))
	return dir
}

func TestResolveValuesFiles(t *testing.T) {
	t.Run("explicit files returned as-is", func(t *testing.T) {
		files := []string{"/a/values.cue", "/b/prod.cue"}
		got, err := resolveValuesFiles("/some/module", files)
		require.NoError(t, err)
		assert.Equal(t, files, got)
	})

	t.Run("no files falls back to values.cue in module dir", func(t *testing.T) {
		dir := tempDirWithValuesFile(t, `values: { name: "test" }`)
		got, err := resolveValuesFiles(dir, nil)
		require.NoError(t, err)
		require.Len(t, got, 1)
		assert.Equal(t, filepath.Join(dir, "values.cue"), got[0])
	})

	t.Run("no files and no values.cue returns ValidationError", func(t *testing.T) {
		dir := t.TempDir()
		_, err := resolveValuesFiles(dir, nil)
		require.Error(t, err)
		var valErr *opmerrors.ValidationError
		require.ErrorAs(t, err, &valErr)
		assert.Contains(t, valErr.Message, "values.cue")
	})
}
