package cmdutil

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/cli/internal/cmdutil/cmdutiltest"
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

func TestModulePackageError_NamesTheModuleCommand(t *testing.T) {
	dir := cmdutiltest.WriteMinimalModule(t)
	k := config.NewKernel("")
	_, acquireErr := k.AcquireInstanceFromDir(context.Background(), dir)
	require.Error(t, acquireErr)

	for _, cmd := range []string{"opm module build", "opm module apply", "opm module vet"} {
		err := ModulePackageError(context.Background(), k, dir, cmd, acquireErr)
		require.Error(t, err, cmd)
		assert.Contains(t, err.Error(), dir+" is a module, not an instance", cmd)
		assert.Contains(t, err.Error(), "run: "+cmd+" "+dir, cmd)
	}
}

// With no module counterpart (instance diff, the cluster queries) the
// refusal points at the module command group, never at a wrong verb.
func TestModulePackageError_NoCounterpartNamesTheGroup(t *testing.T) {
	dir := cmdutiltest.WriteMinimalModule(t)
	k := config.NewKernel("")
	_, acquireErr := k.AcquireInstanceFromDir(context.Background(), dir)
	require.Error(t, acquireErr)

	err := ModulePackageError(context.Background(), k, dir, "", acquireErr)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "is a module, not an instance")
	assert.Contains(t, err.Error(), "'opm module' commands")
	assert.NotContains(t, err.Error(), "opm module build")
}

func TestModulePackageError_OtherKindIsNotAModule(t *testing.T) {
	dir := cmdutiltest.WriteMinimalModule(t)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "module.cue"),
		[]byte("package demo\n\nkind: \"Platform\"\n"), 0o600))
	k := config.NewKernel("")
	_, acquireErr := k.AcquireInstanceFromDir(context.Background(), dir)
	require.Error(t, acquireErr)

	assert.NoError(t, ModulePackageError(context.Background(), k, dir, "opm module build", acquireErr))
}

func TestModulePackageError_NonKindErrorIsIgnored(t *testing.T) {
	k := config.NewKernel("")
	assert.NoError(t, ModulePackageError(context.Background(), k, cmdutiltest.WriteMinimalModule(t), "opm module build", assert.AnError))
}

func TestResolveInstanceArg_RejectsModulePackagePath(t *testing.T) {
	dir := cmdutiltest.WriteMinimalModule(t)

	_, err := ResolveInstanceArg(context.Background(), dir, &config.GlobalConfig{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "is a module, not an instance")
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
