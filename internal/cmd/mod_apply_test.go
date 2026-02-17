package cmd

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewModApplyCmd(t *testing.T) {
	cmd := NewModApplyCmd()

	assert.Equal(t, "apply [path]", cmd.Use)
	assert.NotEmpty(t, cmd.Short)
	assert.NotEmpty(t, cmd.Long)
}

func TestNewModApplyCmd_NoLocalVerboseFlag(t *testing.T) {
	cmd := NewModApplyCmd()

	// Verify that --verbose is NOT a local flag on this command.
	// It should come from the root persistent flag instead.
	localFlag := cmd.Flags().Lookup("verbose")
	assert.Nil(t, localFlag, "--verbose should not be a local flag (should use root persistent flag)")
}

func TestNewModApplyCmd_HelpContainsVerbose(t *testing.T) {
	cmd := NewModApplyCmd()

	// Verify that the Long help text mentions --verbose as an example
	assert.True(t, strings.Contains(cmd.Long, "--verbose"),
		"help text should mention --verbose flag in examples")
}
