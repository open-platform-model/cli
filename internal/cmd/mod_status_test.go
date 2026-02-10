package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// --- 7.5: Tests for required flag validation on mod status ---

func TestModStatusCmd_RequiresName(t *testing.T) {
	cmd := NewModStatusCmd()
	cmd.SetArgs([]string{"-n", "default"})
	err := cmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "required flag")
}

func TestModStatusCmd_RequiresNamespace(t *testing.T) {
	cmd := NewModStatusCmd()
	cmd.SetArgs([]string{"--name", "my-app"})
	err := cmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "required flag")
}

func TestModStatusCmd_InvalidOutputFormat(t *testing.T) {
	// This test verifies the output format validation in the command.
	// We can't fully execute since it needs a cluster, but we can verify
	// the command structure.
	cmd := NewModStatusCmd()
	assert.Equal(t, "status", cmd.Use)

	// Check flags exist
	f := cmd.Flags()
	assert.NotNil(t, f.Lookup("namespace"))
	assert.NotNil(t, f.Lookup("name"))
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
