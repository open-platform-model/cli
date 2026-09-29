package cmdutil

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/cli/internal/config"
	opmexit "github.com/open-platform-model/cli/internal/exit"
)

func TestClassifyModuleArg(t *testing.T) {
	cases := map[string]ModuleArgKind{
		".":                              LocalModuleDir,
		"..":                             LocalModuleDir,
		"./x":                            LocalModuleDir,
		"../x":                           LocalModuleDir,
		"./web.app":                      LocalModuleDir,
		t.TempDir():                      LocalModuleDir,
		"my-module":                      LocalModuleDir,
		"modules/web_app":                LocalModuleDir,
		"opmodel.dev/modules/web_app":    PublishedModule,
		"opmodel.dev/modules/web_app@v1": PublishedModule,
	}
	for arg, want := range cases {
		got, err := ClassifyModuleArg(arg)
		require.NoError(t, err, arg)
		assert.Equal(t, want, got, arg)
	}
}

func TestClassifyModuleArg_DottedDirectoryIsAmbiguous(t *testing.T) {
	t.Chdir(t.TempDir())
	require.NoError(t, os.Mkdir("web.app", 0o755))

	_, err := ClassifyModuleArg("web.app")
	var exitErr *opmexit.ExitError
	require.True(t, errors.As(err, &exitErr))
	assert.Equal(t, opmexit.ExitValidationError, exitErr.Code)
	assert.Contains(t, err.Error(), "./web.app")

	kind, err := ClassifyModuleArg("./web.app")
	require.NoError(t, err)
	assert.Equal(t, LocalModuleDir, kind)
}

func TestResolveModuleArg_VersionWithLocalDirectory(t *testing.T) {
	dir := t.TempDir()
	_, err := ResolveModuleArg(context.Background(), &config.GlobalConfig{}, []string{dir}, "v1", "build")
	var exitErr *opmexit.ExitError
	require.True(t, errors.As(err, &exitErr))
	assert.Equal(t, opmexit.ExitValidationError, exitErr.Code)
	assert.Contains(t, err.Error(), "--version applies only to a published module")
}

func TestResolveModuleArg_LocalDirectory(t *testing.T) {
	dir := t.TempDir()
	arg, err := ResolveModuleArg(context.Background(), &config.GlobalConfig{}, []string{dir}, "", "build")
	require.NoError(t, err)
	assert.Equal(t, dir, arg.Dir)
	assert.Nil(t, arg.Published)
}

func TestResolveModuleArg_MajorSuffixRefusedWithoutRegistry(t *testing.T) {
	// An unreachable registry proves the refusal happens before any I/O.
	cfg := &config.GlobalConfig{Registry: "127.0.0.1:1+insecure"}
	_, err := ResolveModuleArg(context.Background(), cfg, []string{"opmodel.dev/modules/web_app@v1"}, "", "build")
	var exitErr *opmexit.ExitError
	require.True(t, errors.As(err, &exitErr))
	assert.Equal(t, opmexit.ExitValidationError, exitErr.Code)
	assert.Contains(t, err.Error(), "must not carry a major")
}

func TestResolveModuleArg_UnreachableRegistryIsConnectivity(t *testing.T) {
	cfg := &config.GlobalConfig{Registry: "127.0.0.1:1+insecure"}
	_, err := ResolveModuleArg(context.Background(), cfg, []string{"opmodel.dev/modules/web_app"}, "v1", "build")
	var exitErr *opmexit.ExitError
	require.True(t, errors.As(err, &exitErr))
	assert.Equal(t, opmexit.ExitConnectivityError, exitErr.Code)
}
