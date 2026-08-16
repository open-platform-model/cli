package catalogcmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/cli/internal/config"
)

func TestNewCatalogRegistryCmd(t *testing.T) {
	cmd := NewCatalogRegistryCmd(&config.GlobalConfig{})

	assert.Equal(t, "registry", cmd.Use)
	assert.NotEmpty(t, cmd.Short)

	names := make([]string, 0, len(cmd.Commands()))
	for _, c := range cmd.Commands() {
		names = append(names, c.Name())
	}
	assert.Contains(t, names, "check")
}

func TestNewCatalogRegistryCheckCmd(t *testing.T) {
	cmd := NewCatalogRegistryCheckCmd(&config.GlobalConfig{})

	assert.Equal(t, "check <path@version>", cmd.Use)
	assert.NotEmpty(t, cmd.Short)
	require.NotNil(t, cmd.Flags().Lookup("compat"))

	// D35's framing is graduation-gated into the help text VERBATIM: the
	// check is an aid, and enforcement exists only at publish.
	assert.Contains(t, cmd.Long, AidSentence)
	assert.Contains(t, cmd.Long, "Exit codes: 0 clean, 2 findings, 3 registry unreachable.")
}

func TestCatalogCmdCarriesRegistryGroup(t *testing.T) {
	cmd := NewCatalogCmd(&config.GlobalConfig{})
	names := make([]string, 0, len(cmd.Commands()))
	for _, c := range cmd.Commands() {
		names = append(names, c.Name())
	}
	assert.Contains(t, names, "registry")
}
