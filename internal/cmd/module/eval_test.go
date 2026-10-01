package modulecmd

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/cli/internal/config"
	opmexit "github.com/open-platform-model/cli/internal/exit"
)

// evalModuleSource is a module that needs no registry: it carries what the
// kernel's shape gate requires (kind and concrete identity) without
// importing core, so the eval path runs offline.
const evalModuleSource = `package eval_mod

kind: "Module"

metadata: {
	name:       "eval_mod"
	modulePath: "example.com/modules/eval_mod@v0"
	version:    "0.1.0"
}

#config: {
	replicas: *1 | int
	image:    *"nginx:latest" | string
}

debugValues: {}
`

const evalCueModSource = `module: "example.com/modules/eval_mod@v0"
language: version: "v0.17.0"
source: kind: "self"
`

func writeEvalModule(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "cue.mod"), 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "cue.mod", "module.cue"), []byte(evalCueModSource), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "module.cue"), []byte(evalModuleSource), 0o600))
	return dir
}

func runEval(t *testing.T, args ...string) (string, error) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	cmd := NewModuleEvalCmd(&config.GlobalConfig{})
	cmd.SilenceUsage = true // the root command does the same, so usage never reaches stdout
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return stdout.String(), err
}

func TestNewModuleEvalCmd(t *testing.T) {
	cmd := NewModuleEvalCmd(&config.GlobalConfig{})

	assert.Equal(t, "eval [path]", cmd.Use)
	assert.NotEmpty(t, cmd.Short)
	assert.NotEmpty(t, cmd.Long)

	flag := cmd.Flags().Lookup("expression")
	require.NotNil(t, flag)
	assert.Equal(t, "e", flag.Shorthand)
	for _, name := range []string{"platform", "values", "namespace"} {
		assert.Nil(t, cmd.Flags().Lookup(name), "eval takes no render flag --%s", name)
	}
}

func TestModuleCmd_RegistersEval(t *testing.T) {
	cmd, _, err := NewModuleCmd(&config.GlobalConfig{}).Find([]string{"eval"})
	require.NoError(t, err)
	assert.Equal(t, "eval", cmd.Name())
}

func TestModEval_FullModule(t *testing.T) {
	out, err := runEval(t, writeEvalModule(t))
	require.NoError(t, err)

	// Definitions are kept, defaults resolved, and the module prints as bare
	// fields rather than one braced expression.
	assert.Equal(t, `kind: "Module"
metadata: {
	name:       "eval_mod"
	modulePath: "example.com/modules/eval_mod@v0"
	version:    "0.1.0"
}
#config: {
	replicas: 1
	image:    "nginx:latest"
}
debugValues: {}
`, out)
}

func TestModEval_Expression(t *testing.T) {
	dir := writeEvalModule(t)

	tests := []struct {
		name       string
		expression string
		contains   []string
		excludes   []string
	}{
		{
			name:       "definition",
			expression: "#config",
			contains:   []string{"replicas:", "image:", `"nginx:latest"`},
			excludes:   []string{"metadata", `kind: "Module"`},
		},
		{
			name:       "struct field",
			expression: "metadata",
			contains:   []string{`name:`, `"eval_mod"`, `"0.1.0"`},
			excludes:   []string{"#config"},
		},
		{
			name:       "nested scalar",
			expression: "metadata.name",
			contains:   []string{`"eval_mod"`},
			excludes:   []string{"modulePath", "#config"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := runEval(t, dir, "-e", tt.expression)
			require.NoError(t, err)
			for _, s := range tt.contains {
				assert.Contains(t, out, s)
			}
			for _, s := range tt.excludes {
				assert.NotContains(t, out, s)
			}
		})
	}
}

func TestModEval_ExpressionLongFlag(t *testing.T) {
	out, err := runEval(t, writeEvalModule(t), "--expression", "#config")
	require.NoError(t, err)
	assert.Equal(t, "replicas: 1\nimage:    \"nginx:latest\"\n", out)
}

func TestModEval_DefaultsToCurrentDirectory(t *testing.T) {
	dir := writeEvalModule(t)
	t.Chdir(dir)

	out, err := runEval(t, "-e", "metadata.name")
	require.NoError(t, err)
	assert.Equal(t, "\"eval_mod\"\n", out)
}

func TestModEval_BadExpression(t *testing.T) {
	dir := writeEvalModule(t)

	tests := []struct {
		name       string
		expression string
		wantErr    string
	}{
		{"missing field", "metadata.nope", `expression "metadata.nope" does not exist in the module`},
		{"missing definition", "#nope", `expression "#nope" does not exist in the module`},
		{"unparsable path", "metadata..name", `invalid expression "metadata..name"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := runEval(t, dir, "-e", tt.expression)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
			assert.Empty(t, out, "nothing is printed for a bad path")

			var exitErr *opmexit.ExitError
			require.True(t, errors.As(err, &exitErr))
			assert.Equal(t, opmexit.ExitValidationError, exitErr.Code)
		})
	}
}

func TestModEval_RejectsInstancePackage(t *testing.T) {
	instanceDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(instanceDir, "instance.cue"), []byte("package x\n"), 0o600))

	_, err := runEval(t, instanceDir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "is an instance package, not a module")
}

func TestModEval_NotAModule(t *testing.T) {
	_, err := runEval(t, t.TempDir())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "loading module from")

	var exitErr *opmexit.ExitError
	require.True(t, errors.As(err, &exitErr))
	assert.Equal(t, opmexit.ExitGeneralError, exitErr.Code)
}

// TestModEval_FixtureModule evaluates the module-with-debug-values fixture,
// the same module `opm module vet` loads. It imports opmodel.dev/core@v2, so
// it resolves only when a registry (or a warm CUE cache) is available;
// without one the load fails and the test skips, as the vet tests do.
func TestModEval_FixtureModule(t *testing.T) {
	fixtureDir := filepath.Join("..", "..", "..", "tests", "fixtures", "valid", "module-with-debug-values")
	if _, err := os.Stat(fixtureDir); os.IsNotExist(err) {
		t.Skip("Test fixture not found:", fixtureDir)
	}

	full, err := runEval(t, fixtureDir)
	if err != nil {
		t.Skipf("core@v2 not resolvable (registry/cache unavailable?): %v", err)
	}
	assert.Contains(t, full, `"module_with_debug_values"`)
	assert.Contains(t, full, "#config: {")
	assert.Contains(t, full, "debugValues: {")

	cfgOut, err := runEval(t, fixtureDir, "-e", "#config")
	require.NoError(t, err)
	assert.Contains(t, cfgOut, "replicas:")
	assert.NotContains(t, cfgOut, "debugValues")

	nameOut, err := runEval(t, fixtureDir, "-e", "metadata.name")
	require.NoError(t, err)
	assert.Equal(t, "\"module_with_debug_values\"\n", nameOut)

	_, err = runEval(t, fixtureDir, "-e", "metadata.nope")
	require.Error(t, err)
	assert.Contains(t, err.Error(), `expression "metadata.nope" does not exist in the module`)
}
