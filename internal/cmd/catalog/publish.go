package catalogcmd

import (
	"github.com/spf13/cobra"

	"github.com/open-platform-model/cli/internal/cmdutil"
	"github.com/open-platform-model/cli/internal/config"
	"github.com/open-platform-model/cli/internal/publish"
)

// NewCatalogPublishCmd creates the catalog publish command.
//
// Unlike module publish there is no --skip-override-check: a catalog's
// dependency divergence propagates into the key space of every module built
// against it, so a tree carrying cue.mod/local-module.cue always refuses.
func NewCatalogPublishCmd(cfg *config.GlobalConfig) *cobra.Command {
	var flags cmdutil.PublishFlags

	c := &cobra.Command{
		Use:   "publish [path]",
		Short: "Publish a catalog from its committed source",
		Long: `Publish an OPM catalog to its registry, at the coordinates the catalog itself
declares.

	The pipeline reads identity/identity.cue, validates it against core's
	#IdentityPackage, derives repository/major/tag from the declared module path,
	runs the publish gates, prints the resolved plan, and pushes. What is
	published is exactly the committed tree — no copied build directory, no
	generated version override.

	Exit codes: 0 published (or dry-run GO), 2 refused, 3 registry unreachable.

	Arguments:
	  path    Path to the catalog directory (default: current directory)

	Examples:
	  # See the plan and every gate verdict without pushing
	  opm catalog publish ./src --dry-run

	  # Publish at the committed (release-owned) version
	  opm catalog publish ./src

	  # Assert the version being released
	  opm catalog publish ./src --version 1.3.0`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			return cmdutil.RunPublish(c, cfg, publish.KindCatalog, args, &flags)
		},
	}

	flags.AddTo(c)

	return c
}
