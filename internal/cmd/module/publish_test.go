package modulecmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/cli/internal/config"
)

func TestNewModulePublishCmd(t *testing.T) {
	cmd := NewModulePublishCmd(&config.GlobalConfig{})

	assert.Equal(t, "publish [path]", cmd.Use)
	assert.NotEmpty(t, cmd.Short)
	assert.NotEmpty(t, cmd.Long)

	require.NotNil(t, cmd.Flags().Lookup("version"))
	require.NotNil(t, cmd.Flags().Lookup("dry-run"))
	// D6's waiver is module-only, and its name says what it does: the gate is
	// skipped, resolution never changes.
	require.NotNil(t, cmd.Flags().Lookup("skip-override-check"))
}

func TestNewModulePublishCmd_NoLocalVerboseFlag(t *testing.T) {
	cmd := NewModulePublishCmd(&config.GlobalConfig{})
	assert.Nil(t, cmd.Flags().Lookup("verbose"),
		"--verbose should not be a local flag (should use root persistent flag)")
}
