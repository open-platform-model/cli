package modulecmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/opmodel/cli/internal/cmdutil"
	"github.com/opmodel/cli/internal/config"
)

func TestNewModuleBuildCmd(t *testing.T) {
	cmd := NewModuleBuildCmd(&config.GlobalConfig{})
	assert.Equal(t, "build [path]", cmd.Use)
	assert.NotEmpty(t, cmd.Short)
}

func TestNewModuleBuildCmd_Flags(t *testing.T) {
	cmd := NewModuleBuildCmd(&config.GlobalConfig{})
	assert.NotNil(t, cmd.Flags().Lookup("output"), "--output/-o flag should be registered")
	assert.NotNil(t, cmd.Flags().Lookup("values"), "--values/-f flag should be registered")
	assert.NotNil(t, cmd.Flags().Lookup("name"), "--name flag should be registered")
	assert.NotNil(t, cmd.Flags().Lookup("namespace"), "--namespace/-n flag should be registered")
	assert.NotNil(t, cmd.Flags().Lookup("provider"), "--provider flag should be registered")
	assert.NotNil(t, cmd.Flags().Lookup("split"), "--split flag should be registered")
	assert.NotNil(t, cmd.Flags().Lookup("out-dir"), "--out-dir flag should be registered")
}

func TestRunModuleBuild_RejectsFileArgument(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "module.cue")
	require.NoError(t, os.WriteFile(filePath, []byte("package x\n"), 0o644))

	err := runModuleBuild([]string{filePath}, &config.GlobalConfig{}, &cmdutil.RenderFlags{}, "", "yaml", false, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "expects a directory")
	assert.Contains(t, err.Error(), "opm instance build")
}

func TestRunModuleBuild_MissingPath(t *testing.T) {
	err := runModuleBuild([]string{"/nonexistent/module/dir"}, &config.GlobalConfig{}, &cmdutil.RenderFlags{}, "", "yaml", false, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// TestRunModuleBuild_DefaultsToCwd verifies that calling with no args defaults
// to the current directory by exercising ResolveModulePath logic — we can't
// fully run the build without a valid module, so we assert the default-path
// resolution surfaces a stat or load error against ".".
func TestRunModuleBuild_DefaultsToCwd(t *testing.T) {
	cwd, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(cwd) })

	dir := t.TempDir()
	require.NoError(t, os.Chdir(dir))

	// "." is a directory — module build should attempt synthesis (and fail
	// because there is no module package). We assert it does NOT fail with
	// the "expects a directory" error path.
	err = runModuleBuild(nil, &config.GlobalConfig{}, &cmdutil.RenderFlags{}, "", "yaml", false, "")
	require.Error(t, err)
	assert.NotContains(t, err.Error(), "expects a directory")
}
