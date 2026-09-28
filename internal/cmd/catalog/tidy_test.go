package catalogcmd

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/cli/internal/config"
	"github.com/open-platform-model/cli/internal/cuemod/cuemodtest"
	opmexit "github.com/open-platform-model/cli/internal/exit"
)

func TestNewCatalogTidyCmd(t *testing.T) {
	cmd := NewCatalogTidyCmd(&config.GlobalConfig{})

	assert.Equal(t, "tidy [path]", cmd.Use)
	assert.NotEmpty(t, cmd.Short)
	assert.Contains(t, cmd.Long, "Exit codes:")
	assert.NoError(t, cmd.Args(cmd, nil))
	assert.Error(t, cmd.Args(cmd, []string{"./a", "./b"}))

	check := cmd.Flags().Lookup("check")
	require.NotNil(t, check)
	assert.Equal(t, "bool", check.Value.Type())
	assert.Equal(t, "false", check.DefValue)
}

func TestNewCatalogCmd_HasTidy(t *testing.T) {
	cmd := NewCatalogCmd(&config.GlobalConfig{})

	names := make([]string, 0, len(cmd.Commands()))
	for _, c := range cmd.Commands() {
		names = append(names, c.Name())
	}
	assert.Contains(t, names, "tidy")
}

// The catalog command's refusals speak of a catalog, not a module. Not
// parallel: tidy changes the working directory and CUE_REGISTRY.
func TestCatalogTidy_MessagesSayCatalog(t *testing.T) {
	dir, registry := cuemodtest.NewConsumer(t)
	cmd := NewCatalogTidyCmd(&config.GlobalConfig{Registry: registry})
	cmd.SetArgs([]string{"--check", dir})
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true

	err := cmd.Execute()

	var exitErr *opmexit.ExitError
	require.ErrorAs(t, err, &exitErr)
	assert.Equal(t, opmexit.ExitValidationError, exitErr.Code)
	msg := err.Error()
	assert.Contains(t, msg, "catalog is not tidy")
	assert.Contains(t, msg, "opm catalog tidy")
	assert.NotContains(t, strings.ToLower(msg), "module is not tidy")
	assert.NotContains(t, msg, "opm module")
}
