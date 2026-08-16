package catalogcmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/cli/internal/config"
)

func TestNewCatalogCmd(t *testing.T) {
	cmd := NewCatalogCmd(&config.GlobalConfig{})

	assert.Equal(t, "catalog", cmd.Use)
	assert.NotEmpty(t, cmd.Short)
	assert.NotEmpty(t, cmd.Long)

	names := make([]string, 0, len(cmd.Commands()))
	for _, c := range cmd.Commands() {
		names = append(names, c.Name())
	}
	assert.Contains(t, names, "publish")
}

func TestNewCatalogPublishCmd(t *testing.T) {
	cmd := NewCatalogPublishCmd(&config.GlobalConfig{})

	assert.Equal(t, "publish [path]", cmd.Use)
	assert.NotEmpty(t, cmd.Short)
	assert.NotEmpty(t, cmd.Long)

	require.NotNil(t, cmd.Flags().Lookup("version"))
	require.NotNil(t, cmd.Flags().Lookup("dry-run"))
	// A catalog has no override waiver: its dependency divergence propagates
	// into the key space of everything built against it.
	assert.Nil(t, cmd.Flags().Lookup("skip-override-check"))
}
