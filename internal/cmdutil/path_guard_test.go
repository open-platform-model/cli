package cmdutil

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/opmodel/cli/internal/config"
)

func TestValidateModuleInputPath_RejectsInstancePackage(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "instance.cue"), []byte("package test\n"), 0o600))

	err := ValidateModuleInputPath(dir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "instance package, not a module")
	assert.Contains(t, err.Error(), "opm instance")
}

func TestValidateInstanceInputPath_RejectsModulePackage(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "module.cue"), []byte("package test\n"), 0o600))

	err := ValidateInstanceInputPath(dir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "module package, not an instance")
	assert.Contains(t, err.Error(), "opm module")
}

func TestResolveInstanceArg_RejectsModulePackagePath(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "module.cue"), []byte("package test\n"), 0o600))

	_, err := ResolveInstanceArg(dir, &config.GlobalConfig{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "module package, not an instance")
}
