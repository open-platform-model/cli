package modulecmd

import (
	"bytes"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/cli/internal/config"
)

func TestNewModuleTidyCmd(t *testing.T) {
	cmd := NewModuleTidyCmd(&config.GlobalConfig{})

	assert.Equal(t, "tidy [path]", cmd.Use)
	assert.NotEmpty(t, cmd.Short)
	assert.Contains(t, cmd.Long, "Exit codes:")

	// path is optional; at most one.
	assert.NoError(t, cmd.Args(cmd, nil))
	assert.NoError(t, cmd.Args(cmd, []string{"./my-module"}))
	assert.Error(t, cmd.Args(cmd, []string{"./a", "./b"}))

	check := cmd.Flags().Lookup("check")
	require.NotNil(t, check)
	assert.Equal(t, "bool", check.Value.Type())
	assert.Equal(t, "false", check.DefValue)
}

func TestModuleTidy_HelpThroughModAlias(t *testing.T) {
	root := &cobra.Command{Use: "opm"}
	root.AddCommand(NewModuleCmd(&config.GlobalConfig{}))
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"mod", "tidy", "--help"})

	require.NoError(t, root.Execute())
	assert.Contains(t, out.String(), "opm module tidy [path] [flags]")
	assert.Contains(t, out.String(), "--check")
}
