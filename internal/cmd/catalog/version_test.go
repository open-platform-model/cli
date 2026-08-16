package catalogcmd

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/open-platform-model/cli/internal/config"
)

func TestNewCatalogVersionCmd(t *testing.T) {
	cmd := NewCatalogVersionCmd()

	assert.Equal(t, "version", cmd.Use)
	assert.NotEmpty(t, cmd.Short)
	assert.NotEmpty(t, cmd.Long)

	names := make([]string, 0, len(cmd.Commands()))
	for _, c := range cmd.Commands() {
		names = append(names, c.Name())
	}
	assert.Contains(t, names, "set")
}

func TestNewCatalogVersionSetCmd(t *testing.T) {
	cmd := newCatalogVersionSetCmd()

	assert.Equal(t, "set <version> [path]", cmd.Use)
	assert.NotEmpty(t, cmd.Short)
	assert.NotEmpty(t, cmd.Long)

	// version is required, path optional.
	assert.Error(t, cmd.Args(cmd, nil))
	assert.NoError(t, cmd.Args(cmd, []string{"1.3.0"}))
	assert.NoError(t, cmd.Args(cmd, []string{"1.3.0", "./src"}))
	assert.Error(t, cmd.Args(cmd, []string{"1.3.0", "./src", "extra"}))

	// Offline by design: no registry or schema flags.
	assert.Nil(t, cmd.Flags().Lookup("version"))
	assert.Nil(t, cmd.Flags().Lookup("dry-run"))
}

func TestNewCatalogCmd_HasVersionGroup(t *testing.T) {
	cmd := NewCatalogCmd(&config.GlobalConfig{})

	names := make([]string, 0, len(cmd.Commands()))
	for _, c := range cmd.Commands() {
		names = append(names, c.Name())
	}
	assert.Contains(t, names, "version")
}
