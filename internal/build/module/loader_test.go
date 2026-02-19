package module

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolvePath_ValidModule(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(dir, "cue.mod"), 0o755))

	result, err := ResolvePath(dir)
	require.NoError(t, err)
	assert.NotEmpty(t, result)
}

func TestResolvePath_NotFound(t *testing.T) {
	_, err := ResolvePath("/nonexistent/path/that/does/not/exist")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "module directory not found")
}

func TestResolvePath_MissingCueMod(t *testing.T) {
	dir := t.TempDir()
	_, err := ResolvePath(dir)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cue.mod")
}

func TestExtractMetadataFromAST_NilFiles(t *testing.T) {
	name, ns := ExtractMetadataFromAST(nil)
	assert.Empty(t, name)
	assert.Empty(t, ns)
}
