package cmdutil

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/opmodel/cli/internal/config"
)

func TestValidateModuleInputPath_RejectsReleasePackage(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "release.cue"), []byte("package test\n"), 0o600))

	err := ValidateModuleInputPath(dir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "release package, not a module")
	assert.Contains(t, err.Error(), "opm release")
}

func TestValidateReleaseInputPath_RejectsModulePackage(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "module.cue"), []byte("package test\n"), 0o600))

	err := ValidateReleaseInputPath(dir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "module package, not a release")
	assert.Contains(t, err.Error(), "opm module")
}

func TestResolveReleaseArg_RejectsModulePackagePath(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "module.cue"), []byte("package test\n"), 0o600))

	_, err := ResolveReleaseArg(dir, &config.GlobalConfig{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "module package, not a release")
}
