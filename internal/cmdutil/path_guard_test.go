package cmdutil

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/cli/internal/config"
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

	_, err := ResolveInstanceArg(context.Background(), dir, &config.GlobalConfig{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "module package, not an instance")
}

func TestInstanceDir(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "instance.cue")
	require.NoError(t, os.WriteFile(file, []byte("package test\n"), 0o600))

	tests := []struct {
		name string
		path string
		want string
	}{
		{name: "directory resolves to itself", path: dir, want: dir},
		{name: "file resolves to its parent", path: file, want: dir},
		{name: "missing path resolves to its parent", path: filepath.Join(dir, "missing.cue"), want: dir},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := InstanceDir(tt.path)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
