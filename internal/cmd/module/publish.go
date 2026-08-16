package modulecmd

import (
	"github.com/spf13/cobra"

	"github.com/open-platform-model/cli/internal/cmdutil"
	"github.com/open-platform-model/cli/internal/config"
	"github.com/open-platform-model/cli/internal/publish"
)

// NewModulePublishCmd creates the module publish command.
func NewModulePublishCmd(cfg *config.GlobalConfig) *cobra.Command {
	var flags cmdutil.PublishFlags

	c := &cobra.Command{
		Use:   "publish [path]",
		Short: "Publish a module from its committed source",
		Long: `Publish an OPM module to its registry, at the coordinates the module itself
declares.

	The pipeline reads identity/identity.cue, validates it against core's
	#IdentityPackage, derives repository/major/tag from the declared module path,
	runs the publish gates, prints the resolved plan, and pushes. What is
	published is exactly the committed tree.

	Exit codes: 0 published (or dry-run GO), 2 refused, 3 registry unreachable.

	Arguments:
	  path    Path to the module directory (default: current directory)

	Examples:
	  # Publish the module in the current directory
	  opm module publish

	  # See the plan and every gate verdict without pushing
	  opm module publish ./my-module --dry-run

	  # Fill an open identity Version (writes identity/identity.cue), then publish
	  opm module publish --version 1.3.0`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			return cmdutil.RunPublish(c, cfg, publish.KindModule, args, &flags)
		},
	}

	flags.AddTo(c)
	c.Flags().BoolVar(&flags.SkipOverrideCheck, "skip-override-check", false,
		"Publish despite cue.mod/local-module.cue; replacements are ignored either way")

	return c
}
