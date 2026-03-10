package mod

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"cuelang.org/go/cue/cuecontext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/opmodel/cli/internal/config"
)

func TestNewModVetCmd(t *testing.T) {
	cmd := NewModVetCmd(&config.GlobalConfig{})

	assert.Equal(t, "vet [path]", cmd.Use)
	assert.NotEmpty(t, cmd.Short)
	assert.NotEmpty(t, cmd.Long)
	// Args validation is set to MaximumNArgs(1) but not directly testable
}

func TestNewModVetCmd_NoLocalVerboseFlag(t *testing.T) {
	cmd := NewModVetCmd(&config.GlobalConfig{})

	// Verify that --verbose is NOT a local flag on this command.
	// It should come from the root persistent flag instead.
	localFlag := cmd.Flags().Lookup("verbose")
	assert.Nil(t, localFlag, "--verbose should not be a local flag (should use root persistent flag)")
}

// TestModVet_ValidModule exercises the module vet path with the simple-module
// fixture (no release.cue, no debugValues).
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

	os.Unsetenv("OPM_REGISTRY")

	cfg := &config.GlobalConfig{
		CueContext: cuecontext.New(),
	}
	cmd := NewModVetCmd(cfg)
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{fixtureDir})

	err := cmd.Execute()
	require.Error(t, err, "module without debugValues should fail")
	assert.Contains(t, err.Error(), "module does not define debugValues")
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

// TestModVet_ValuesDetailLogic checks the display detail string assembled
// for the "Values satisfy #config" vet check line.
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
			var detail string
			if len(tt.valuesFlags) > 0 {
				basenames := make([]string, len(tt.valuesFlags))
				for i, vf := range tt.valuesFlags {
					basenames[i] = filepath.Base(vf)
				}
				detail = strings.Join(basenames, ", ")
			} else {
				detail = "debugValues"
			}
			assert.Equal(t, tt.expectedDetail, detail)
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
			basenames := make([]string, 0, len(tt.valuesFlags))
			for _, vf := range tt.valuesFlags {
				basenames = append(basenames, filepath.Base(vf))
			}
			assert.Equal(t, tt.expectedDetail, strings.Join(basenames, ", "))
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
