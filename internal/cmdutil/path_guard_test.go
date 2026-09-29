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

// minimalModule is a module package body that passes the kernel's module
// shape gate without importing core, so no registry is needed.
const minimalModule = `package demo

kind: "Module"
metadata: {
	name:       "demo"
	modulePath: "example.com/modules/demo@v0"
	version:    "0.1.0"
}
`

// writeMinimalModule writes a module package that needs no registry: a
// cue.mod without dependencies and minimalModule.
func writeMinimalModule(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "cue.mod"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "cue.mod", "module.cue"),
		[]byte("module: \"example.com/modules/demo@v0\"\nlanguage: version: \"v0.17.0\"\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "module.cue"), []byte(minimalModule), 0o600))
	return dir
}

func TestModulePackageError_ModulePackageNamesModuleBuild(t *testing.T) {
	dir := writeMinimalModule(t)
	k := config.NewKernel("")
	_, acquireErr := k.AcquireInstanceFromDir(context.Background(), dir)
	require.Error(t, acquireErr)

	err := ModulePackageError(context.Background(), k, dir, acquireErr)
	require.Error(t, err)
	assert.Contains(t, err.Error(), dir+" is a module, not an instance")
	assert.Contains(t, err.Error(), "opm module build "+dir)
}

func TestModulePackageError_OtherKindIsNotAModule(t *testing.T) {
	dir := writeMinimalModule(t)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "module.cue"),
		[]byte("package demo\n\nkind: \"Platform\"\n"), 0o600))
	k := config.NewKernel("")
	_, acquireErr := k.AcquireInstanceFromDir(context.Background(), dir)
	require.Error(t, acquireErr)

	assert.NoError(t, ModulePackageError(context.Background(), k, dir, acquireErr))
}

func TestModulePackageError_NonKindErrorIsIgnored(t *testing.T) {
	k := config.NewKernel("")
	assert.NoError(t, ModulePackageError(context.Background(), k, writeMinimalModule(t), assert.AnError))
}

func TestResolveInstanceArg_RejectsModulePackagePath(t *testing.T) {
	dir := writeMinimalModule(t)

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
