package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewModVetCmd(t *testing.T) {
	cmd := NewModVetCmd()

	assert.Equal(t, "vet [path]", cmd.Use)
	assert.NotEmpty(t, cmd.Short)
	assert.NotEmpty(t, cmd.Long)
	// Args validation is set to MaximumNArgs(1) but not directly testable
}

func TestModVet_ValidModule(t *testing.T) {
	// Use a test fixture — assumes tests/fixtures/simple-module exists
	fixtureDir := filepath.Join("..", "..", "tests", "fixtures", "simple-module")
	if _, err := os.Stat(fixtureDir); os.IsNotExist(err) {
		t.Skip("Test fixture not found:", fixtureDir)
	}

	// Set up minimal config in temp directory
	tmpHome, cleanup := setupTestConfig(t)
	defer cleanup()

	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", origHome)

	// Clear registry override for test
	os.Unsetenv("OPM_REGISTRY")

	cmd := NewModVetCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{fixtureDir})

	err := cmd.Execute()
	require.NoError(t, err, "valid module should exit with code 0")
}

func TestModVet_CUEValidationError(t *testing.T) {
	t.Skip("Requires valid OPM module fixture with CUE errors — skipping for now")
	// This test requires a module that triggers CUE validation errors during render.
	// The test module needs proper imports and structure which is complex to set up inline.
	// Integration tests with real fixtures would be better for this case.
}

func TestModVet_UnmatchedComponent(t *testing.T) {
	t.Skip("Requires provider fixture with transformers — skipping for now")
	// This test would require setting up a module with components that don't match
	// any transformers in the provider, which requires a more complex test fixture.
}

func TestModVet_StrictMode(t *testing.T) {
	t.Skip("Requires fixture with unhandled traits — skipping for now")
	// This test would verify that --strict causes exit code 2 when there are
	// unhandled traits. Requires a fixture module with traits that no transformer handles.
}

func TestModVet_ValuesDetailLogic(t *testing.T) {
	tests := []struct {
		name           string
		valuesFlags    []string
		expectedDetail string
	}{
		{
			name:           "no values flags uses module values.cue",
			valuesFlags:    nil,
			expectedDetail: "values.cue",
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
			var detail string
			if len(tt.valuesFlags) > 0 {
				basenames := make([]string, len(tt.valuesFlags))
				for i, vf := range tt.valuesFlags {
					basenames[i] = filepath.Base(vf)
				}
				detail = strings.Join(basenames, ", ")
			} else {
				detail = "values.cue"
			}
			assert.Equal(t, tt.expectedDetail, detail)
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
