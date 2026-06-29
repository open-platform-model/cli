package modulecmd

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/opmodel/cli/internal/cmdutil"
	"github.com/opmodel/cli/internal/config"
	opmexit "github.com/opmodel/cli/internal/exit"
)

func TestNewModuleApplyCmd(t *testing.T) {
	cmd := NewModuleApplyCmd(&config.GlobalConfig{})
	assert.Equal(t, "apply [path]", cmd.Use)
	assert.NotEmpty(t, cmd.Short)
	assert.NotEmpty(t, cmd.Long)
	assert.NotNil(t, cmd.Args, "should accept at most 1 positional arg")
}

func TestNewModuleApplyCmd_Long_DocumentsSyntheticIdentity(t *testing.T) {
	cmd := NewModuleApplyCmd(&config.GlobalConfig{})
	long := cmd.Long
	assert.Contains(t, long, "-debug", "Long must mention the synthetic <module>-debug naming convention")
	assert.Contains(t, long, "instance identity", "Long must explain that --name/--namespace participate in instance identity")
	assert.Contains(t, long, "instance.cue", "Long must point users at instance.cue for persistent deploys")
	assert.Contains(t, long, "opm instance apply", "Long must mention the graduation path")
	assert.Contains(t, long, "opm instance delete", "Long must call out the orphan-inventory mitigation")
}

func TestNewModuleApplyCmd_Long_HasUsageExamples(t *testing.T) {
	long := NewModuleApplyCmd(&config.GlobalConfig{}).Long
	// Three required examples: default invocation, --name override, dry-run.
	assert.Contains(t, long, "opm module apply\n", "Long must include the default invocation example")
	assert.Contains(t, long, "--name", "Long must include a --name override example")
	assert.Contains(t, long, "--dry-run", "Long must include a --dry-run example")
}

func TestNewModuleApplyCmd_Flags(t *testing.T) {
	cmd := NewModuleApplyCmd(&config.GlobalConfig{})

	type flagExpect struct {
		name      string
		shorthand string
		typeName  string
		defValue  string
	}
	cases := []flagExpect{
		{"values", "f", "stringArray", "[]"},
		{"provider", "", "string", ""},
		{"name", "", "string", ""},
		{"namespace", "n", "string", ""},
		{"kubeconfig", "", "string", ""},
		{"context", "", "string", ""},
		{"dry-run", "", "bool", "false"},
		{"create-namespace", "", "bool", "false"},
		{"no-prune", "", "bool", "false"},
		{"force", "", "bool", "false"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f := cmd.Flags().Lookup(c.name)
			require.NotNil(t, f, "flag --%s must be registered", c.name)
			assert.Equal(t, c.shorthand, f.Shorthand, "flag --%s shorthand", c.name)
			assert.Equal(t, c.typeName, f.Value.Type(), "flag --%s type", c.name)
			assert.Equal(t, c.defValue, f.DefValue, "flag --%s default value", c.name)
		})
	}
}

func TestNewModuleApplyCmd_RegisteredOnModuleGroup(t *testing.T) {
	group := NewModuleCmd(&config.GlobalConfig{})
	var found bool
	for _, sub := range group.Commands() {
		if sub.Name() == "apply" {
			found = true
			break
		}
	}
	assert.True(t, found, "module group should have an apply subcommand")
}

func TestRunModuleApply_RejectsFileArgument(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "module.cue")
	require.NoError(t, os.WriteFile(filePath, []byte("package x\n"), 0o644))

	err := runModuleApply([]string{filePath}, &config.GlobalConfig{}, &cmdutil.RenderFlags{}, &cmdutil.K8sFlags{}, "", false, false, false, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "expects a directory")
	assert.Contains(t, err.Error(), "opm instance apply", "error must point users at instance apply for files")

	var exitErr *opmexit.ExitError
	require.True(t, errors.As(err, &exitErr), "error must be an *opmexit.ExitError")
	assert.Equal(t, opmexit.ExitGeneralError, exitErr.Code)
}

func TestRunModuleApply_MissingPath(t *testing.T) {
	err := runModuleApply([]string{"/nonexistent/module/dir"}, &config.GlobalConfig{}, &cmdutil.RenderFlags{}, &cmdutil.K8sFlags{}, "", false, false, false, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")

	var exitErr *opmexit.ExitError
	require.True(t, errors.As(err, &exitErr))
	assert.Equal(t, opmexit.ExitGeneralError, exitErr.Code)
}

// TestRunModuleApply_PathValidation_TableDriven covers the validation paths that
// fire BEFORE any render or cluster contact: file-vs-directory, missing path.
// These map to the validation-vs-general exit-code split called out in task 4.2.
// Render-time errors (missing debugValues, malformed CUE) are exercised in the
// integration test under tests/integration/, since constructing a real module
// package in a unit test requires a CUE catalog dependency graph.
func TestRunModuleApply_PathValidation_TableDriven(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "module.cue")
	require.NoError(t, os.WriteFile(filePath, []byte("package x\n"), 0o644))

	cases := []struct {
		name         string
		args         []string
		wantContains string
		wantCode     int
	}{
		{
			name:         "file argument rejected",
			args:         []string{filePath},
			wantContains: "expects a directory",
			wantCode:     opmexit.ExitGeneralError,
		},
		{
			name:         "missing path rejected",
			args:         []string{"/nonexistent/module/dir"},
			wantContains: "not found",
			wantCode:     opmexit.ExitGeneralError,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := runModuleApply(tc.args, &config.GlobalConfig{}, &cmdutil.RenderFlags{}, &cmdutil.K8sFlags{}, "", false, false, false, false)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.wantContains)

			var exitErr *opmexit.ExitError
			require.True(t, errors.As(err, &exitErr))
			assert.Equal(t, tc.wantCode, exitErr.Code)
		})
	}
}
