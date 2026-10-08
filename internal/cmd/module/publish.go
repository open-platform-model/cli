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
	published is the module directory as it is on disk, zipped by CUE's module
	machinery: the command checks no git state, so commit first.

	--version fills an open identity Version by writing it into
	identity/identity.cue just before the push (never on --dry-run), and the
	zip carries that written file. It asserts a declared Version and never
	overwrites one.

	Exit codes: 0 published (or dry-run GO), 2 refused or no registry
	configured, 3 a registry operation failed (unreachable, or another
	registry error), 4 the registry refused the credentials.

	Arguments:
	  path    Path to the module directory (default: current directory)

	Examples:
	  # Publish the module in the current directory
	  opm module publish

	  # See the plan and every gate verdict without pushing
	  opm module publish ./my-module --dry-run

	  # Write 1.3.0 into an open identity Version, then publish
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
