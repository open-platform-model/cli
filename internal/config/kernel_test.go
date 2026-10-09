package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewKernel_ReturnsKernelWithSchemaCache(t *testing.T) {
	for _, registry := range []string{"", "opmodel.dev=ghcr.io/open-platform-model"} {
		k := NewKernel(registry)
		require.NotNil(t, k, "registry %q", registry)
		require.NotNil(t, k.SchemaCache(), "registry %q", registry)
	}
}

func TestRegistryConfigured(t *testing.T) {
	for _, tc := range []struct {
		name        string
		registry    string
		cueRegistry string
		want        bool
	}{
		{"nothing set", "", "", false},
		{"resolved registry", "opmodel.dev=ghcr.io/open-platform-model", "", true},
		{"only CUE_REGISTRY", "", "opmodel.dev=ghcr.io/open-platform-model", true},
		{"both", "localhost:5000", "opmodel.dev=ghcr.io/open-platform-model", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("CUE_REGISTRY", tc.cueRegistry)
			assert.Equal(t, tc.want, RegistryConfigured(tc.registry))
		})
	}
}

func TestNoRegistryError_NamesTheCauseAndTheNextStep(t *testing.T) {
	cause := errors.New("module opmodel.dev/core@v2.0.0-beta.4: module not found")
	dir := t.TempDir()

	missing := NewNoRegistryError("loading core schema", filepath.Join(dir, "config.cue"), cause)
	assert.False(t, missing.ConfigExists)
	msg := missing.Error()
	assert.Contains(t, msg, "no registry is configured")
	assert.Contains(t, msg, "loading core schema: "+cause.Error(), "the cause stays visible")
	assert.Contains(t, msg, "--registry, OPM_REGISTRY or the registry field of "+filepath.Join(dir, "config.cue"))
	assert.True(t, strings.HasSuffix(msg, "run:  opm config init"), "a first run is pointed at config init: %s", msg)
	assert.NotContains(t, msg, "--force")
	assert.ErrorIs(t, missing, cause)

	// A config file that sets no registry: config init refuses without --force.
	present := filepath.Join(dir, "present.cue")
	require.NoError(t, os.WriteFile(present, []byte("package config\n\nconfig: {}\n"), 0o600))
	existing := NewNoRegistryError("loading core schema", present, cause)
	assert.True(t, existing.ConfigExists)
	assert.Contains(t, existing.Error(), "opm config init --force")

	assert.Contains(t, NewNoRegistryError("loading core schema", "", cause).Error(), "the registry field of the config file.")
}
