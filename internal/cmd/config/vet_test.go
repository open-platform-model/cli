// Package config provides CLI command implementations for config operations.
package config

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	opmconfig "github.com/open-platform-model/cli/internal/config"
	"github.com/open-platform-model/cli/internal/output"
)

// writeOpmFile writes content into ~/.opm/<name> under tmpHome, creating the
// directory as needed, and returns the file path.
func writeOpmFile(t *testing.T, tmpHome, name, content string) {
	t.Helper()
	path := filepath.Join(tmpHome, ".opm", name)
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o700))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
}

const validVetConfig = `package config

config: {
	kubernetes: {
		kubeconfig: "~/.kube/config"
		namespace: "default"
	}
}
`

func TestNewConfigVetCmd(t *testing.T) {
	cmd := NewConfigVetCmd(&opmconfig.GlobalConfig{})

	assert.Equal(t, "vet", cmd.Use)
	assert.NotEmpty(t, cmd.Short)
	assert.NotEmpty(t, cmd.Long)
}

func TestConfigVet_MissingConfigFile(t *testing.T) {
	setTempHome(t)
	os.Unsetenv("OPM_CONFIG")

	cmd := NewConfigVetCmd(&opmconfig.GlobalConfig{})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})

	err := cmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestConfigVet_ValidConfig_NoPlatformModule(t *testing.T) {
	// Without a leftover ~/.opm/platform/ vet passes silently: no platform
	// is checked.
	tmpHome := setTempHome(t)
	os.Unsetenv("OPM_CONFIG")
	os.Unsetenv("OPM_REGISTRY")
	logs := captureVetLog(t)

	writeOpmFile(t, tmpHome, "config.cue", validVetConfig)

	cmd := NewConfigVetCmd(&opmconfig.GlobalConfig{})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})

	stdout := captureVetStdout(t, func() { require.NoError(t, cmd.Execute()) })
	configPath := filepath.Join(tmpHome, ".opm", "config.cue")
	assert.Equal(t,
		output.FormatVetCheck("Config file found", configPath)+"\n"+
			output.FormatVetCheck("Config schema validation passed", "")+"\n",
		stdout, "exactly the two config checks, in order")
	assert.NotContains(t, stdout, "Platform module builds", "no platform check line")
	assert.NotContains(t, logs.String(), "no longer read")
}

// captureVetStdout runs fn with os.Stdout redirected, where the vet check
// lines are printed, and returns what it wrote.
func captureVetStdout(t *testing.T, fn func()) string {
	t.Helper()
	oldOut := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = w
	defer func() { os.Stdout = oldOut }()
	fn()
	require.NoError(t, w.Close())
	raw, err := io.ReadAll(r)
	require.NoError(t, err)
	require.NoError(t, r.Close())
	return string(raw)
}

// captureVetLog redirects the CLI's log sink for the test and returns it.
func captureVetLog(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	output.SetLogWriter(&buf)
	t.Cleanup(func() { output.SetLogWriter(os.Stderr) })
	return &buf
}

// TestConfigVet_LeftoverPlatformDirWarns covers "A leftover platform
// directory warns": vet passes, warns that the directory is no longer read,
// and neither builds nor changes it (it need not even be a module).
func TestConfigVet_LeftoverPlatformDirWarns(t *testing.T) {
	tmpHome := setTempHome(t)
	os.Unsetenv("OPM_CONFIG")
	os.Unsetenv("OPM_REGISTRY")
	logs := captureVetLog(t)

	writeOpmFile(t, tmpHome, "config.cue", validVetConfig)
	writeOpmFile(t, tmpHome, filepath.Join("platform", "platform.cue"), "not a module: true\n")

	cmd := NewConfigVetCmd(&opmconfig.GlobalConfig{})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	require.NoError(t, cmd.Execute())

	platformDir := filepath.Join(tmpHome, ".opm", "platform")
	assert.Contains(t, logs.String(), platformDir+" is no longer read by any command; pass it with --platform <dir> or delete it")
	entries, err := os.ReadDir(platformDir)
	require.NoError(t, err)
	assert.Len(t, entries, 1, "the directory is not built or written")
}

func TestConfigVet_LegacyPlatformFileFails(t *testing.T) {
	// A pre-0019 data-only platform.cue fails naming the file, with the
	// --force migration hint; the config checks still pass first.
	tmpHome := setTempHome(t)
	os.Unsetenv("OPM_CONFIG")
	os.Unsetenv("OPM_REGISTRY")

	writeOpmFile(t, tmpHome, "config.cue", validVetConfig)
	writeOpmFile(t, tmpHome, "platform.cue", `name: "cluster"
type: "kubernetes"
`)

	cmd := NewConfigVetCmd(&opmconfig.GlobalConfig{})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), filepath.Join(tmpHome, ".opm", "platform.cue"))
	assert.Contains(t, err.Error(), "opm config init --force")
}

func TestConfigVet_StaleProvidersBlock(t *testing.T) {
	// A config predating 0006:D39, with a providers block, fails with the migration hint.
	tmpHome := setTempHome(t)
	os.Unsetenv("OPM_CONFIG")
	os.Unsetenv("OPM_REGISTRY")

	writeOpmFile(t, tmpHome, "config.cue", `package config

config: {
	registry: "localhost:5000"
	providers: {
		kubernetes: {}
	}
	kubernetes: {
		namespace: "default"
	}
}
`)

	cmd := NewConfigVetCmd(&opmconfig.GlobalConfig{})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})

	err := cmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "providers")
	assert.Contains(t, err.Error(), "opm config init")
}

func TestConfigVet_InvalidCUESyntax(t *testing.T) {
	tmpHome := setTempHome(t)
	os.Unsetenv("OPM_CONFIG")
	os.Unsetenv("OPM_REGISTRY")

	writeOpmFile(t, tmpHome, "config.cue", `package config

config: {
	this is not valid CUE syntax!!!
}
`)

	cmd := NewConfigVetCmd(&opmconfig.GlobalConfig{})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})

	err := cmd.Execute()
	assert.Error(t, err)
}

func TestConfigVet_SchemaViolation_UnknownField(t *testing.T) {
	tmpHome := setTempHome(t)
	os.Unsetenv("OPM_CONFIG")
	os.Unsetenv("OPM_REGISTRY")

	writeOpmFile(t, tmpHome, "config.cue", `package config

config: {
	registry: "localhost:5000"
	unknownField: "this should fail schema validation"
	kubernetes: {
		namespace: "default"
	}
}
`)

	cmd := NewConfigVetCmd(&opmconfig.GlobalConfig{})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})

	err := cmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "schema validation")
}

func TestConfigVet_SchemaViolation_InvalidNamespace(t *testing.T) {
	tmpHome := setTempHome(t)
	os.Unsetenv("OPM_CONFIG")
	os.Unsetenv("OPM_REGISTRY")

	writeOpmFile(t, tmpHome, "config.cue", `package config

config: {
	kubernetes: {
		namespace: "UPPERCASE-Not-Allowed"
	}
}
`)

	cmd := NewConfigVetCmd(&opmconfig.GlobalConfig{})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})

	err := cmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "schema validation")
}

func TestConfigVet_SchemaViolation_InvalidAPIWarnings(t *testing.T) {
	tmpHome := setTempHome(t)
	os.Unsetenv("OPM_CONFIG")
	os.Unsetenv("OPM_REGISTRY")

	writeOpmFile(t, tmpHome, "config.cue", `package config

config: {
	log: {
		kubernetes: {
			apiWarnings: "invalid-not-an-enum-value"
		}
	}
}
`)

	cmd := NewConfigVetCmd(&opmconfig.GlobalConfig{})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})

	err := cmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "schema validation")
}

func TestConfigVet_CustomConfigPath(t *testing.T) {
	tmpHome := setTempHome(t)

	// Create custom config location (no ~/.opm involvement)
	customDir := filepath.Join(tmpHome, "custom")
	require.NoError(t, os.MkdirAll(customDir, 0o700))

	customConfig := filepath.Join(customDir, "config.cue")
	require.NoError(t, os.WriteFile(customConfig, []byte(`package config

config: {
	kubernetes: {
		namespace: "test"
	}
}
`), 0o600))

	// Use OPM_CONFIG env var to point to custom config
	os.Setenv("OPM_CONFIG", customConfig)
	defer os.Unsetenv("OPM_CONFIG")
	os.Unsetenv("OPM_REGISTRY")

	cmd := NewConfigVetCmd(&opmconfig.GlobalConfig{})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})

	require.NoError(t, cmd.Execute())
}

func TestConfigVet_CustomPathPlatformSibling(t *testing.T) {
	// The leftover platform/ is looked for beside the resolved config path,
	// so --config/OPM_CONFIG overrides move the warning with it.
	tmpHome := setTempHome(t)
	logs := captureVetLog(t)

	customDir := filepath.Join(tmpHome, "custom")
	require.NoError(t, os.MkdirAll(filepath.Join(customDir, "platform"), 0o700))

	customConfig := filepath.Join(customDir, "config.cue")
	require.NoError(t, os.WriteFile(customConfig, []byte(`package config

config: {
	kubernetes: {
		namespace: "test"
	}
}
`), 0o600))

	os.Setenv("OPM_CONFIG", customConfig)
	defer os.Unsetenv("OPM_CONFIG")
	os.Unsetenv("OPM_REGISTRY")

	cmd := NewConfigVetCmd(&opmconfig.GlobalConfig{})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})

	require.NoError(t, cmd.Execute())
	assert.Contains(t, logs.String(), filepath.Join(customDir, "platform")+" is no longer read by any command")
}
