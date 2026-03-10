package mod

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/opmodel/cli/internal/cmdutil"
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

func TestRunBuild_RejectsNonManifestOutput(t *testing.T) {
	err := runModBuild(nil, &config.GlobalConfig{}, &cmdutil.RenderFlags{}, "table", false, "")
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "invalid output format"))
}
