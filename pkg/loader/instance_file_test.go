package loader

import (
	"os"
	"path/filepath"
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// makeInstanceFileFixture creates a temp directory with a cue.mod and an instance .cue file.
// Returns the path to the created .cue file.
func makeInstanceFileFixture(t *testing.T, filename, content string) string {
	t.Helper()
	dir := t.TempDir()

	// Create minimal cue.mod so load.Instances can find the module root.
	modDir := filepath.Join(dir, "cue.mod")
	require.NoError(t, os.MkdirAll(modDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(modDir, "module.cue"), []byte(`module: "test.example.com/releases@v0"
language: version: "v0.15.0"
`), 0o644))

	filePath := filepath.Join(dir, filename)
	require.NoError(t, os.WriteFile(filePath, []byte(content), 0o644))
	return filePath
}

// TestLoadInstanceFile tests loading standalone .cue instance files.
func TestLoadInstanceFile(t *testing.T) {
	ctx := cuecontext.New()

	t.Run("valid ModuleInstance file", func(t *testing.T) {
		filePath := makeInstanceFileFixture(t, "instance.cue", `package instances

kind: "ModuleInstance"
metadata: name: "my-instance"
`)
		val, dir, err := LoadInstanceFile(ctx, filePath, LoadOptions{})
		require.NoError(t, err)
		assert.NotEmpty(t, dir)
		assert.NoError(t, val.Err())

		kind, kindErr := val.LookupPath(cue.ParsePath("kind")).String()
		require.NoError(t, kindErr)
		assert.Equal(t, "ModuleInstance", kind)
	})

	t.Run("invalid CUE syntax", func(t *testing.T) {
		filePath := makeInstanceFileFixture(t, "bad.cue", `package instances

this is not valid CUE !!!`)
		_, _, err := LoadInstanceFile(ctx, filePath, LoadOptions{})
		require.Error(t, err)
	})

	t.Run("file not found", func(t *testing.T) {
		_, _, err := LoadInstanceFile(ctx, "/nonexistent/path/instance.cue", LoadOptions{})
		require.Error(t, err)
	})

	t.Run("returns parent directory", func(t *testing.T) {
		filePath := makeInstanceFileFixture(t, "my_instance.cue", `package instances

kind: "ModuleInstance"
`)
		_, dir, err := LoadInstanceFile(ctx, filePath, LoadOptions{})
		require.NoError(t, err)
		assert.Equal(t, filepath.Dir(filePath), dir)
	})
}
