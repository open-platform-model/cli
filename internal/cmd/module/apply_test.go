package modulecmd

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/opmodel/cli/internal/config"
)

func TestNewModuleApplyCmd(t *testing.T) {
	cmd := NewModuleApplyCmd(&config.GlobalConfig{})

	assert.Equal(t, "apply [path]", cmd.Use)
	assert.NotEmpty(t, cmd.Short)
	assert.NotEmpty(t, cmd.Long)
}

func TestNewModuleApplyCmd_NoLocalVerboseFlag(t *testing.T) {
	cmd := NewModuleApplyCmd(&config.GlobalConfig{})

	// Verify that --verbose is NOT a local flag on this command.
	// It should come from the root persistent flag instead.
	localFlag := cmd.Flags().Lookup("verbose")
	assert.Nil(t, localFlag, "--verbose should not be a local flag (should use root persistent flag)")
}

func TestNewModuleApplyCmd_HelpContainsVerbose(t *testing.T) {
	cmd := NewModuleApplyCmd(&config.GlobalConfig{})

	// Verify that the Long help text mentions --verbose as an example
	assert.True(t, strings.Contains(cmd.Long, "--verbose"),
		"help text should mention --verbose flag in examples")
}
