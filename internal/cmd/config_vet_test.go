// Package cmd provides CLI command implementations.
package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewConfigVetCmd(t *testing.T) {
	cmd := NewConfigVetCmd()

	assert.Equal(t, "vet", cmd.Use)
	assert.NotEmpty(t, cmd.Short)
	assert.NotEmpty(t, cmd.Long)
}

func TestConfigVet_MissingConfigFile(t *testing.T) {
	// Use temp home directory without config
	tmpHome, err := os.MkdirTemp("", "config-vet-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpHome)

	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", origHome)

	// Clear any config override
	os.Unsetenv("OPM_CONFIG")

	cmd := NewConfigVetCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})

	err = cmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestConfigVet_MissingModuleFile(t *testing.T) {
	tmpHome, err := os.MkdirTemp("", "config-vet-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpHome)

	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", origHome)

	os.Unsetenv("OPM_CONFIG")

	// Create config file but not cue.mod/module.cue
	opmDir := filepath.Join(tmpHome, ".opm")
	require.NoError(t, os.MkdirAll(opmDir, 0o700))
	configFile := filepath.Join(opmDir, "config.cue")
	require.NoError(t, os.WriteFile(configFile, []byte("package config\n"), 0o600))

	cmd := NewConfigVetCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})

	err = cmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "module.cue")
}

func TestConfigVet_ValidConfig(t *testing.T) {
	tmpHome, err := os.MkdirTemp("", "config-vet-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpHome)

	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", origHome)

	os.Unsetenv("OPM_CONFIG")
	os.Unsetenv("OPM_REGISTRY")

	// Create full config structure using templates
	opmDir := filepath.Join(tmpHome, ".opm")
	require.NoError(t, os.MkdirAll(opmDir, 0o700))

	cueModDir := filepath.Join(opmDir, "cue.mod")
	require.NoError(t, os.MkdirAll(cueModDir, 0o700))

	// Write a simple config that doesn't require imports
	simpleConfig := `package config

config: {
	kubernetes: {
		kubeconfig: "~/.kube/config"
		namespace: "default"
	}
}
`
	configFile := filepath.Join(opmDir, "config.cue")
	require.NoError(t, os.WriteFile(configFile, []byte(simpleConfig), 0o600))

	// Write module file
	moduleContent := `module: "test.local/config@v0"

language: {
	version: "v0.15.0"
}
`
	moduleFile := filepath.Join(cueModDir, "module.cue")
	require.NoError(t, os.WriteFile(moduleFile, []byte(moduleContent), 0o600))

	cmd := NewConfigVetCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})

	err = cmd.Execute()
	require.NoError(t, err)
}

func TestConfigVet_InvalidCUESyntax(t *testing.T) {
	tmpHome, err := os.MkdirTemp("", "config-vet-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpHome)

	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", origHome)

	os.Unsetenv("OPM_CONFIG")
	os.Unsetenv("OPM_REGISTRY")

	// Create config structure with invalid CUE
	opmDir := filepath.Join(tmpHome, ".opm")
	require.NoError(t, os.MkdirAll(opmDir, 0o700))

	cueModDir := filepath.Join(opmDir, "cue.mod")
	require.NoError(t, os.MkdirAll(cueModDir, 0o700))

	// Invalid CUE syntax
	invalidConfig := `package config

config: {
	this is not valid CUE syntax!!!
}
`
	configFile := filepath.Join(opmDir, "config.cue")
	require.NoError(t, os.WriteFile(configFile, []byte(invalidConfig), 0o600))

	moduleContent := `module: "test.local/config@v0"

language: {
	version: "v0.15.0"
}
`
	moduleFile := filepath.Join(cueModDir, "module.cue")
	require.NoError(t, os.WriteFile(moduleFile, []byte(moduleContent), 0o600))

	cmd := NewConfigVetCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})

	err = cmd.Execute()
	assert.Error(t, err)
}

func TestConfigVet_CustomConfigPath(t *testing.T) {
	tmpHome, err := os.MkdirTemp("", "config-vet-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpHome)

	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", origHome)

	// Create custom config location
	customDir := filepath.Join(tmpHome, "custom")
	require.NoError(t, os.MkdirAll(customDir, 0o700))

	cueModDir := filepath.Join(customDir, "cue.mod")
	require.NoError(t, os.MkdirAll(cueModDir, 0o700))

	simpleConfig := `package config

config: {
	kubernetes: {
		namespace: "test"
	}
}
`
	customConfig := filepath.Join(customDir, "config.cue")
	require.NoError(t, os.WriteFile(customConfig, []byte(simpleConfig), 0o600))

	moduleContent := `module: "test.local/config@v0"

language: {
	version: "v0.15.0"
}
`
	moduleFile := filepath.Join(cueModDir, "module.cue")
	require.NoError(t, os.WriteFile(moduleFile, []byte(moduleContent), 0o600))

	// Use OPM_CONFIG env var to point to custom config
	os.Setenv("OPM_CONFIG", customConfig)
	defer os.Unsetenv("OPM_CONFIG")
	os.Unsetenv("OPM_REGISTRY")

	cmd := NewConfigVetCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})

	err = cmd.Execute()
	require.NoError(t, err)
}
