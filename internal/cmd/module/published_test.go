package modulecmd

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	"cuelang.org/go/mod/modregistrytest"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/cli/internal/cmdutil"
	"github.com/open-platform-model/cli/internal/config"
	opmexit "github.com/open-platform-model/cli/internal/exit"
	"github.com/open-platform-model/cli/internal/output"
)

// captureLogs routes the log stream into a buffer for the test's duration.
func captureLogs(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	output.SetupLogging(output.LogConfig{})
	output.SetLogWriter(&buf)
	t.Cleanup(func() { output.SetupLogging(output.LogConfig{}) })
	return &buf
}

// publishedRegistry serves example.com/modules/web_app at v1.0.0 from an
// in-memory registry and returns its CUE registry mapping.
func publishedRegistry(t *testing.T) string {
	t.Helper()
	t.Setenv("CUE_CACHE_DIR", t.TempDir())
	dir := "example.com_modules_web_app_v1.0.0"
	reg, err := modregistrytest.New(fstest.MapFS{
		dir + "/cue.mod/module.cue": {Data: []byte("module: \"example.com/modules/web_app@v1\"\nlanguage: version: \"v0.17.0\"\n")},
		dir + "/module.cue":         {Data: []byte("package web_app\n")},
	}, "")
	require.NoError(t, err)
	t.Cleanup(reg.Close)
	return reg.Host() + "+insecure"
}

func requireExitCode(t *testing.T, err error, code int) {
	t.Helper()
	var exitErr *opmexit.ExitError
	require.True(t, errors.As(err, &exitErr), "want an *opmexit.ExitError, got %v", err)
	assert.Equal(t, code, exitErr.Code, err.Error())
}

func TestModuleCommands_VersionFlag(t *testing.T) {
	for _, cmd := range []*cobra.Command{
		NewModuleBuildCmd(&config.GlobalConfig{}),
		NewModuleApplyCmd(&config.GlobalConfig{}),
	} {
		f := cmd.Flags().Lookup("version")
		require.NotNil(t, f, "%s registers --version", cmd.Name())
		assert.Equal(t, "string", f.Value.Type())
		assert.Equal(t, "", f.DefValue, "%s --version defaults to empty", cmd.Name())
		assert.Contains(t, cmd.Long, "opmodel.dev/modules/web_app", "%s help shows a published module", cmd.Name())
	}
}

func TestRunModuleBuild_VersionWithLocalDirectory(t *testing.T) {
	err := runModuleBuild([]string{t.TempDir()}, &config.GlobalConfig{}, &cmdutil.RenderFlags{}, "", "v1", "yaml", false, "")
	requireExitCode(t, err, opmexit.ExitValidationError)
	assert.Contains(t, err.Error(), "--version applies only to a published module")
}

// An unpublished pin is refused by resolution, before the kubeconfig (which
// does not exist here) is ever read.
func TestRunModuleApply_UnpublishedPinExitsBeforeClusterConfig(t *testing.T) {
	cfg := &config.GlobalConfig{Registry: publishedRegistry(t)}
	kf := &cmdutil.K8sFlags{Kubeconfig: filepath.Join(t.TempDir(), "missing-kubeconfig")}

	err := runModuleApply([]string{"example.com/modules/web_app"}, cfg, &cmdutil.RenderFlags{}, kf, applyOpts{version: "9.9.9"})
	requireExitCode(t, err, opmexit.ExitValidationError)
	assert.Contains(t, err.Error(), "no published version 9.9.9")
}

func TestRunModuleApply_WarnsWhenApplyingDebugValues(t *testing.T) {
	logs := captureLogs(t)
	kf := &cmdutil.K8sFlags{Kubeconfig: filepath.Join(t.TempDir(), "missing-kubeconfig")}

	// The cluster connection fails after the warning; the warning is what is
	// under test.
	_ = runModuleApply([]string{t.TempDir()}, &config.GlobalConfig{}, &cmdutil.RenderFlags{}, kf, applyOpts{})

	assert.Contains(t, logs.String(), "debugValues")
	assert.Contains(t, logs.String(), "opm instance init")
}

func TestRunModuleApply_NoWarningWithValuesFiles(t *testing.T) {
	logs := captureLogs(t)
	kf := &cmdutil.K8sFlags{Kubeconfig: filepath.Join(t.TempDir(), "missing-kubeconfig")}
	values := filepath.Join(t.TempDir(), "values.cue")
	require.NoError(t, os.WriteFile(values, []byte("values: {}\n"), 0o644))

	_ = runModuleApply([]string{t.TempDir()}, &config.GlobalConfig{}, &cmdutil.RenderFlags{Values: []string{values}}, kf, applyOpts{})

	assert.NotContains(t, logs.String(), "debugValues")
}

func TestRunModuleBuild_NoDebugValuesWarning(t *testing.T) {
	logs := captureLogs(t)

	// No module package in the directory: the build fails, and prints no
	// apply warning on the way.
	_ = runModuleBuild([]string{t.TempDir()}, &config.GlobalConfig{}, &cmdutil.RenderFlags{}, "", "", "yaml", false, "")

	assert.NotContains(t, logs.String(), "opm instance init")
}
