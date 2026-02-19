package mod

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/opmodel/cli/internal/config"
)

func TestNewModBuildCmd(t *testing.T) {
	cmd := NewModBuildCmd(&config.GlobalConfig{})

	assert.Equal(t, "build [path]", cmd.Use)
	assert.NotEmpty(t, cmd.Short)
	assert.NotEmpty(t, cmd.Long)
}

func TestNewModBuildCmd_NoLocalVerboseFlag(t *testing.T) {
	cmd := NewModBuildCmd(&config.GlobalConfig{})

	// Verify that --verbose is NOT a local flag on this command.
	// It should come from the root persistent flag instead.
	localFlag := cmd.Flags().Lookup("verbose")
	assert.Nil(t, localFlag, "--verbose should not be a local flag (should use root persistent flag)")
}
