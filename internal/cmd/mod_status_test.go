package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// --- Tests for flag validation on mod status ---

func TestModStatusCmd_RequiresNameOrReleaseID(t *testing.T) {
	cmd := NewModStatusCmd()
	cmd.SetArgs([]string{"-n", "default"})
	err := cmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "either --name or --release-id is required")
}

func TestModStatusCmd_MutuallyExclusive(t *testing.T) {
	cmd := NewModStatusCmd()
	cmd.SetArgs([]string{"-n", "default", "--name", "my-app", "--release-id", "abc123"})
	err := cmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "--name and --release-id are mutually exclusive")
}

func TestModStatusCmd_RequiresNamespace(t *testing.T) {
	cmd := NewModStatusCmd()
	cmd.SetArgs([]string{"--name", "my-app"})
	err := cmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "required flag")
}

func TestModStatusCmd_FlagsExist(t *testing.T) {
	cmd := NewModStatusCmd()
	assert.Equal(t, "status", cmd.Use)

	// Check flags exist
	f := cmd.Flags()
	assert.NotNil(t, f.Lookup("namespace"))
	assert.NotNil(t, f.Lookup("name"))
	assert.NotNil(t, f.Lookup("release-id"))
	assert.NotNil(t, f.Lookup("output"))
	assert.NotNil(t, f.Lookup("watch"))
	assert.NotNil(t, f.Lookup("kubeconfig"))
	assert.NotNil(t, f.Lookup("context"))
}

func TestModDiffCmd_FlagsExist(t *testing.T) {
	cmd := NewModDiffCmd()
	assert.Equal(t, "diff [path]", cmd.Use)

	f := cmd.Flags()
	assert.NotNil(t, f.Lookup("values"))
	assert.NotNil(t, f.Lookup("namespace"))
	assert.NotNil(t, f.Lookup("name"))
	assert.NotNil(t, f.Lookup("kubeconfig"))
	assert.NotNil(t, f.Lookup("context"))
}
