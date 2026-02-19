package mod

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/opmodel/cli/internal/cmdtypes"
)

// --- Tests for flag validation on mod status ---

func TestModStatusCmd_RequiresNameOrReleaseID(t *testing.T) {
	cmd := NewModStatusCmd(&cmdtypes.GlobalConfig{})
	cmd.SetArgs([]string{"-n", "default"})
	err := cmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "either --release-name or --release-id is required")
}

func TestModStatusCmd_MutuallyExclusive(t *testing.T) {
	cmd := NewModStatusCmd(&cmdtypes.GlobalConfig{})
	cmd.SetArgs([]string{"-n", "default", "--release-name", "my-app", "--release-id", "abc123"})
	err := cmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "--release-name and --release-id are mutually exclusive")
}

func TestModStatusCmd_NamespaceOptional(t *testing.T) {
	// Namespace is now optional - falls back to config or default "default"
	// Verify the flag exists
	cmd := NewModStatusCmd(&cmdtypes.GlobalConfig{})
	f := cmd.Flags().Lookup("namespace")
	assert.NotNil(t, f)

	// Check that namespace is not in the required annotations
	// Cobra uses the "cobra_annotation_required" annotation to track required flags
	annotations := f.Annotations
	_, isRequired := annotations["cobra_annotation_required"]
	assert.False(t, isRequired, "namespace flag should not be required")
}

func TestModStatusCmd_FlagsExist(t *testing.T) {
	cmd := NewModStatusCmd(&cmdtypes.GlobalConfig{})
	assert.Equal(t, "status", cmd.Use)

	// Check flags exist
	f := cmd.Flags()
	assert.NotNil(t, f.Lookup("namespace"))
	assert.NotNil(t, f.Lookup("release-name"))
	assert.NotNil(t, f.Lookup("release-id"))
	assert.NotNil(t, f.Lookup("output"))
	assert.NotNil(t, f.Lookup("watch"))
	assert.NotNil(t, f.Lookup("kubeconfig"))
	assert.NotNil(t, f.Lookup("context"))
}

func TestModDiffCmd_FlagsExist(t *testing.T) {
	cmd := NewModDiffCmd(&cmdtypes.GlobalConfig{})
	assert.Equal(t, "diff [path]", cmd.Use)

	f := cmd.Flags()
	assert.NotNil(t, f.Lookup("values"))
	assert.NotNil(t, f.Lookup("namespace"))
	assert.NotNil(t, f.Lookup("release-name"))
	assert.NotNil(t, f.Lookup("kubeconfig"))
	assert.NotNil(t, f.Lookup("context"))
}
