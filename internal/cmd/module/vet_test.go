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

func TestNewModuleVetCmd(t *testing.T) {
	cmd := NewModuleVetCmd(&config.GlobalConfig{})

	assert.Equal(t, "vet [path]", cmd.Use)
	assert.NotEmpty(t, cmd.Short)
	assert.NotEmpty(t, cmd.Long)
	// Args validation is set to MaximumNArgs(1) but not directly testable
}

func TestNewModuleVetCmd_NoLocalVerboseFlag(t *testing.T) {
	cmd := NewModuleVetCmd(&config.GlobalConfig{})

	// Verify that --verbose is NOT a local flag on this command.
	// It should come from the root persistent flag instead.
	localFlag := cmd.Flags().Lookup("verbose")
	assert.Nil(t, localFlag, "--verbose should not be a local flag (should use root persistent flag)")
}

// TestModVet_ValidModule exercises the module vet path with the simple-module
// fixture (core@v2 line, no instance.cue, no debugValues of its own). Core's
// #Module declares `debugValues: _`, and every #config field of the fixture
// carries a default, so the kernel merges the open debugValues with #config
// to a concrete value and vet accepts the module — the verdict build reaches
// for the same input. The fixture imports opmodel.dev/core@v2, so it resolves
// only when a registry (or a warm CUE cache) is available; without one the
// core schema load fails with a connectivity error before the vet check is
// reached, and the test skips rather than false-failing — matching the
// repo's other registry-backed tests.
func TestModVet_ValidModule(t *testing.T) {
	fixtureDir := filepath.Join("..", "..", "..", "tests", "fixtures", "valid", "simple-module")
	if _, err := os.Stat(fixtureDir); os.IsNotExist(err) {
		t.Skip("Test fixture not found:", fixtureDir)
	}

	tmpHome, cleanup := setupTestConfig(t)
	defer cleanup()

	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", origHome)

	cfg := &config.GlobalConfig{}
	cmd := NewModuleVetCmd(cfg)
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{fixtureDir})

	err := cmd.Execute()
	var exitErr *opmexit.ExitError
	if errors.As(err, &exitErr) && exitErr.Code == opmexit.ExitConnectivityError {
		t.Skipf("core@v2 not resolvable (registry/cache unavailable?): %v", err)
	}
	require.NoError(t, err, "an all-defaults #config merges open debugValues to a concrete value")
}

func TestModVet_RejectsInstancePackage(t *testing.T) {
	tmpHome, cleanup := setupTestConfig(t)
	defer cleanup()

	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", origHome)

	os.Unsetenv("OPM_REGISTRY")

	instanceDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(instanceDir, "instance.cue"), []byte(`package jellyfin

kind: "ModuleInstance"
metadata: name: "jf"
`), 0o600))

	cfg := &config.GlobalConfig{}
	cmd := NewModuleVetCmd(cfg)
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{instanceDir})

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "is an instance package, not a module")
	assert.Contains(t, err.Error(), "opm instance")
}

func TestModVet_CUEValidationError(t *testing.T) {
	t.Skip("Requires valid OPM module fixture with CUE errors — skipping for now")
	// This test requires a module that triggers CUE validation errors during render.
	// The test module needs proper imports and structure which is complex to set up inline.
	// Integration tests with real fixtures would be better for this case.
}

// TestModVet_ValuesDetailLogic checks the display detail string vetValuesDetail
// assembles for the "Values satisfy #config" vet check line.
func TestModVet_ValuesDetailLogic(t *testing.T) {
	tests := []struct {
		name           string
		valuesFlags    []string
		expectedDetail string
	}{
		{
			name:           "no values flags uses debugValues",
			valuesFlags:    nil,
			expectedDetail: "debugValues",
		},
		{
			name:           "single external values file",
			valuesFlags:    []string{"/path/to/prod-values.cue"},
			expectedDetail: "prod-values.cue",
		},
		{
			name:           "multiple external values files",
			valuesFlags:    []string{"/path/to/base.cue", "/another/path/prod.cue"},
			expectedDetail: "base.cue, prod.cue",
		},
		{
			name:           "values files with absolute paths show only basename",
			valuesFlags:    []string{"/very/long/path/to/config/values.cue"},
			expectedDetail: "values.cue",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expectedDetail, vetValuesDetail(tt.valuesFlags))
		})
	}
}

func TestModVet_MultipleValuesAreMergedForValidation(t *testing.T) {
	tests := []struct {
		name           string
		valuesFlags    []string
		expectedDetail string
	}{
		{
			name:           "single file",
			valuesFlags:    []string{"prod-values.cue"},
			expectedDetail: "prod-values.cue",
		},
		{
			name:           "multiple files",
			valuesFlags:    []string{"base.cue", "override.cue"},
			expectedDetail: "base.cue, override.cue",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expectedDetail, vetValuesDetail(tt.valuesFlags))
		})
	}
}

// setupTestConfig creates a minimal test config in a temp directory.
func setupTestConfig(t *testing.T) (tmpHome string, cleanup func()) {
	tmpHome, err := os.MkdirTemp("", "mod-vet-config-*")
	require.NoError(t, err)

	opmDir := filepath.Join(tmpHome, ".opm")
	require.NoError(t, os.MkdirAll(opmDir, 0o700))

	cueModDir := filepath.Join(opmDir, "cue.mod")
	require.NoError(t, os.MkdirAll(cueModDir, 0o700))

	// Minimal config
	simpleConfig := `package config

config: {
	providers: {
		"default": {
			registry: "opmodel.dev"
		}
	}
}
`
	require.NoError(t, os.WriteFile(filepath.Join(opmDir, "config.cue"), []byte(simpleConfig), 0o600))

	// Module file
	moduleContent := `module: "test.local/config@v0"

language: {
	version: "v0.15.0"
}
`
	require.NoError(t, os.WriteFile(filepath.Join(cueModDir, "module.cue"), []byte(moduleContent), 0o600))

	cleanup = func() {
		os.RemoveAll(tmpHome)
	}

	return tmpHome, cleanup
}
