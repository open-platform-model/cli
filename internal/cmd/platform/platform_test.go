package platformcmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/cli/internal/config"
)

func TestNewPlatformCmd(t *testing.T) {
	cmd := NewPlatformCmd(&config.GlobalConfig{})

	assert.Equal(t, "platform", cmd.Use)
	assert.NotEmpty(t, cmd.Short)
	assert.NotEmpty(t, cmd.Long)

	names := make([]string, 0, len(cmd.Commands()))
	for _, c := range cmd.Commands() {
		names = append(names, c.Name())
	}
	assert.Contains(t, names, "check")
}

func TestNewPlatformCheckCmd(t *testing.T) {
	cmd := NewPlatformCheckCmd(&config.GlobalConfig{})

	assert.Equal(t, "check [dir]", cmd.Use)
	assert.NotEmpty(t, cmd.Short)
	assert.NotEmpty(t, cmd.Long)

	require.NotNil(t, cmd.Flags().Lookup("platform"))
	// The check reads a platform module offline: no cluster connection flags
	// and no render flags belong on it.
	assert.Nil(t, cmd.Flags().Lookup("kubeconfig"))
	assert.Nil(t, cmd.Flags().Lookup("values"))

	// At most one positional directory.
	assert.Error(t, cmd.Args(cmd, []string{"a", "b"}))
	assert.NoError(t, cmd.Args(cmd, []string{"a"}))
	assert.NoError(t, cmd.Args(cmd, nil))
}
