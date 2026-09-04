// Package config provides CLI command implementations for config operations.
package config

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	opmconfig "github.com/open-platform-model/cli/internal/config"
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

// validVetConfigWithRegistry pins the registry the platform module build
// resolves from, so registry-backed vet tests do not depend on the
// process environment.
var validVetConfigWithRegistry = `package config

config: {
	registry: "` + opmconfig.DefaultRegistry + `"
	kubernetes: {
		kubeconfig: "~/.kube/config"
		namespace: "default"
	}
}
`

// skipIfRegistryUnavailable skips when err looks like the registry could
// not be reached (the repo's posture for registry-backed unit tests).
func skipIfRegistryUnavailable(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		return
	}
	msg := err.Error()
	for _, needle := range []string{"dial tcp", "no such host", "connection refused", "i/o timeout", "context deadline exceeded", "TLS handshake", "network is unreachable"} {
		if strings.Contains(msg, needle) {
			t.Skipf("registry unavailable: %v", err)
		}
	}
}

// writePlatformModule seeds ~/.opm/platform under tmpHome and returns it.
func writePlatformModule(t *testing.T, tmpHome string) string {
	t.Helper()
	dir := filepath.Join(tmpHome, ".opm", "platform")
	require.NoError(t, opmconfig.WritePlatformModule(dir))
	return dir
}

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
	// A missing ~/.opm/platform/ is a note, not a failure.
	tmpHome := setTempHome(t)
	os.Unsetenv("OPM_CONFIG")
	os.Unsetenv("OPM_REGISTRY")

	writeOpmFile(t, tmpHome, "config.cue", validVetConfig)

	cmd := NewConfigVetCmd(&opmconfig.GlobalConfig{})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})

	require.NoError(t, cmd.Execute())
}

func TestConfigVet_ValidConfigAndPlatformModule(t *testing.T) {
	// Registry-backed: the seeded module builds against the published core
	// and catalogs, so vet passes end to end.
	tmpHome := setTempHome(t)
	os.Unsetenv("OPM_CONFIG")
	os.Unsetenv("OPM_REGISTRY")

	writeOpmFile(t, tmpHome, "config.cue", validVetConfigWithRegistry)
	writePlatformModule(t, tmpHome)

	cmd := NewConfigVetCmd(&opmconfig.GlobalConfig{})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})

	err := cmd.Execute()
	skipIfRegistryUnavailable(t, err)
	require.NoError(t, err)
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

func TestConfigVet_PlatformDirNotAModule(t *testing.T) {
	// A platform/ directory without cue.mod/module.cue is not a module.
	tmpHome := setTempHome(t)
	os.Unsetenv("OPM_CONFIG")
	os.Unsetenv("OPM_REGISTRY")

	writeOpmFile(t, tmpHome, "config.cue", validVetConfig)
	writeOpmFile(t, tmpHome, filepath.Join("platform", "platform.cue"), `name: "cluster"
`)

	cmd := NewConfigVetCmd(&opmconfig.GlobalConfig{})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "platform module")
	assert.Contains(t, err.Error(), "cue.mod/module.cue")
}

func TestConfigVet_PlatformModuleUnpublishedPin(t *testing.T) {
	// Registry-backed: a pin naming a build that does not exist fails vet
	// naming the dependency and pointing at cue.mod.
	tmpHome := setTempHome(t)
	os.Unsetenv("OPM_CONFIG")
	os.Unsetenv("OPM_REGISTRY")

	writeOpmFile(t, tmpHome, "config.cue", validVetConfigWithRegistry)
	dir := writePlatformModule(t, tmpHome)
	modPath := filepath.Join(dir, "cue.mod", "module.cue")
	content, err := os.ReadFile(modPath)
	require.NoError(t, err)
	bumped := strings.Replace(string(content), opmconfig.DefaultCatalogPins[0], "v4.9.9", 1)
	require.NotEqual(t, string(content), bumped)
	require.NoError(t, os.WriteFile(modPath, []byte(bumped), 0o600))

	cmd := NewConfigVetCmd(&opmconfig.GlobalConfig{})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})

	err = cmd.Execute()
	skipIfRegistryUnavailable(t, err)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "opmodel.dev/catalogs/opm@v4.9.9")
	assert.Contains(t, err.Error(), modPath)
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
	// The platform module resolves as the sibling platform/ of the resolved
	// config path, so --config/OPM_CONFIG overrides move both together.
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
	// An invalid platform sibling (a platform/ that is not a module) must
	// fail vet even at a custom path.
	require.NoError(t, os.MkdirAll(filepath.Join(customDir, "platform"), 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(customDir, "platform", "platform.cue"), []byte(`bogus: true
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
