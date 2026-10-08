package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// findCommand returns the command at path below the root, such as
// "instance delete".
func findCommand(t *testing.T, path string) *cobra.Command {
	t.Helper()
	c, rest, err := NewRootCmd().Find(strings.Fields(path))
	require.NoError(t, err)
	require.Empty(t, rest, "%q is not a command", path)
	return c
}

// --force overrides a refusal that protects existing state. These are the
// commands that refuse, and none of them asks a question.
func TestForceIsOfferedWhereACommandRefuses(t *testing.T) {
	for _, path := range []string{"instance apply", "module apply", "platform pull", "config init"} {
		t.Run(path, func(t *testing.T) {
			f := findCommand(t, path).Flags().Lookup("force")
			require.NotNil(t, f, "--force is registered")
			assert.Empty(t, f.Deprecated, "--force is the current spelling here")
			assert.False(t, f.Hidden)
			assert.Equal(t, "false", f.DefValue)
			assert.NotContains(t, strings.ToLower(f.Usage), "confirm", "--force never answers a prompt")
		})
	}
}

func TestConfirmingCommandsAcceptYes(t *testing.T) {
	for _, path := range []string{"instance delete", "module init"} {
		t.Run(path, func(t *testing.T) {
			f := findCommand(t, path).Flags().Lookup("yes")
			require.NotNil(t, f, "--yes is registered")
			assert.Equal(t, "y", f.Shorthand)
			assert.Equal(t, "false", f.DefValue)
			assert.Empty(t, f.Deprecated)
			assert.False(t, f.Hidden)
		})
	}
}

// No command offers a current --force next to --yes: where a command
// confirms, --force can only be the deprecated spelling of --yes.
func TestNoCommandOffersForceAsThePromptFlag(t *testing.T) {
	walkCommands(NewRootCmd(), func(c *cobra.Command) {
		if c.Flags().Lookup("yes") == nil {
			return
		}
		if f := c.Flags().Lookup("force"); f != nil {
			assert.Equal(t, "use --yes", f.Deprecated, "%s: --force beside --yes must be its deprecated alias", c.CommandPath())
		}
	})
}

func TestInstanceDeleteForceIsADeprecatedAliasOfYes(t *testing.T) {
	c := findCommand(t, "instance delete")
	f := c.Flags().Lookup("force")
	require.NotNil(t, f, "the old spelling still parses")
	assert.Equal(t, "use --yes", f.Deprecated)
	assert.True(t, f.Hidden, "a deprecated flag is not in the help")
	assert.NotContains(t, c.Flags().FlagUsages(), "--force")
	assert.Contains(t, c.Flags().FlagUsages(), "--yes")

	var warnings bytes.Buffer
	c.SetOut(&warnings)
	require.NoError(t, c.ParseFlags([]string{"jellyfin", "--force"}))
	assert.Equal(t, "Flag --force has been deprecated, use --yes\n", warnings.String())
	assert.True(t, f.Changed)
}

func TestInstanceDeleteYesPrintsNoDeprecation(t *testing.T) {
	for _, arg := range []string{"--yes", "-y"} {
		c := findCommand(t, "instance delete")
		var warnings bytes.Buffer
		c.SetOut(&warnings)
		require.NoError(t, c.ParseFlags([]string{"jellyfin", arg}))
		assert.Empty(t, warnings.String(), arg)
		assert.True(t, c.Flags().Lookup("yes").Changed, arg)
	}
}

// walkCommands calls fn for c and every command below it.
func walkCommands(c *cobra.Command, fn func(*cobra.Command)) {
	fn(c)
	for _, sub := range c.Commands() {
		walkCommands(sub, fn)
	}
}

// --delete-data is the one flag that lets delete and prune remove
// PersistentVolumeClaims: the three commands offer it, off by default and
// spelled in full, and each help says claims are kept without it.
func TestDeleteDataFlagOnDeleteAndBothApplies(t *testing.T) {
	for _, path := range []string{"instance delete", "instance apply", "module apply"} {
		c := findCommand(t, path)
		f := c.Flags().Lookup("delete-data")
		require.NotNil(t, f, "%s offers --delete-data", path)
		assert.Equal(t, "bool", f.Value.Type(), path)
		assert.Equal(t, "false", f.DefValue, "%s: data is kept by default", path)
		assert.Empty(t, f.Shorthand, "%s: a flag that deletes data has no shorthand", path)
		assert.Empty(t, f.Deprecated, path)
		assert.False(t, f.Hidden, path)
		assert.Contains(t, f.Usage, "PersistentVolumeClaims", path)
		assert.Contains(t, f.Usage, "kept by default", path)
		assert.Contains(t, c.Long, "PersistentVolumeClaims", "%s: the help says claims are kept", path)
		assert.Contains(t, c.Long, "--delete-data", path)
	}
}

// No other command offers --delete-data, so the flag keeps one meaning.
func TestDeleteDataFlagIsOnNoOtherCommand(t *testing.T) {
	allowed := map[string]bool{"opm instance delete": true, "opm instance apply": true, "opm module apply": true}
	walkCommands(NewRootCmd(), func(c *cobra.Command) {
		if c.Flags().Lookup("delete-data") != nil {
			assert.True(t, allowed[c.CommandPath()], "%s offers --delete-data", c.CommandPath())
		}
	})
}
