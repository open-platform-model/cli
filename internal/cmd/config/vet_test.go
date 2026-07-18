// Package config provides CLI command implementations for config operations.
package config

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	opmconfig "github.com/open-platform-model/cli/internal/config"
)

// writeOpmFile writes content into ~/.opm/<name> under tmpHome, creating the
// directory as needed, and returns the file path.
func writeOpmFile(t *testing.T, tmpHome, name, content string) {
	t.Helper()
	opmDir := filepath.Join(tmpHome, ".opm")
	require.NoError(t, os.MkdirAll(opmDir, 0o700))
	path := filepath.Join(opmDir, name)
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

func TestConfigVet_ValidConfig_NoPlatformFile(t *testing.T) {
	// A missing platform.cue is a note, not a failure (0006 D39).
	tmpHome := setTempHome(t)
	os.Unsetenv("OPM_CONFIG")
	os.Unsetenv("OPM_REGISTRY")

	writeOpmFile(t, tmpHome, "config.cue", validVetConfig)

	cmd := NewConfigVetCmd(&opmconfig.GlobalConfig{})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})

	require.NoError(t, cmd.Execute())
}

func TestConfigVet_ValidConfigAndPlatform(t *testing.T) {
	tmpHome := setTempHome(t)
	os.Unsetenv("OPM_CONFIG")
	os.Unsetenv("OPM_REGISTRY")

	writeOpmFile(t, tmpHome, "config.cue", validVetConfig)
	writeOpmFile(t, tmpHome, "platform.cue", opmconfig.DefaultPlatformTemplate)

	cmd := NewConfigVetCmd(&opmconfig.GlobalConfig{})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})

	require.NoError(t, cmd.Execute())
}

func TestConfigVet_InvalidPlatformFile(t *testing.T) {
	tmpHome := setTempHome(t)
	os.Unsetenv("OPM_CONFIG")
	os.Unsetenv("OPM_REGISTRY")

	writeOpmFile(t, tmpHome, "config.cue", validVetConfig)
	// Missing required name/type
	writeOpmFile(t, tmpHome, "platform.cue", `registry: {}
`)

	cmd := NewConfigVetCmd(&opmconfig.GlobalConfig{})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})

	err := cmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "platform")
}

func TestConfigVet_ImportBearingPlatformFile(t *testing.T) {
	tmpHome := setTempHome(t)
	os.Unsetenv("OPM_CONFIG")
	os.Unsetenv("OPM_REGISTRY")

	writeOpmFile(t, tmpHome, "config.cue", validVetConfig)
	writeOpmFile(t, tmpHome, "platform.cue", `import "strings"

name: strings.ToLower("Cluster")
type: "kubernetes"
`)

	cmd := NewConfigVetCmd(&opmconfig.GlobalConfig{})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})

	err := cmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "data-only")
}

func TestConfigVet_StaleProvidersBlock(t *testing.T) {
	// A pre-D39 config with a providers block fails with the migration hint.
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
	// The platform file resolves as a sibling of the resolved config path,
	// so --config/OPM_CONFIG overrides move both files together.
	tmpHome := setTempHome(t)

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
	// Invalid platform sibling must fail vet even at a custom path
	require.NoError(t, os.WriteFile(filepath.Join(customDir, "platform.cue"), []byte(`bogus: true
`), 0o600))

	os.Setenv("OPM_CONFIG", customConfig)
	defer os.Unsetenv("OPM_CONFIG")
	os.Unsetenv("OPM_REGISTRY")

	cmd := NewConfigVetCmd(&opmconfig.GlobalConfig{})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})

	err := cmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "platform")
}
