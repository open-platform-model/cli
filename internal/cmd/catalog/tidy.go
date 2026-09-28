package catalogcmd

import (
	"github.com/spf13/cobra"

	"github.com/open-platform-model/cli/internal/cmdutil"
	"github.com/open-platform-model/cli/internal/config"
	"github.com/open-platform-model/cli/internal/publish"
)

// NewCatalogTidyCmd creates the catalog tidy command.
func NewCatalogTidyCmd(cfg *config.GlobalConfig) *cobra.Command {
	var check bool

	c := &cobra.Command{
		Use:   "tidy [path]",
		Short: "Resolve, pin and prune the catalog's CUE dependencies",
		Long: `Tidy the catalog's cue.mod/module.cue (and cue.mod/local-module.cue) the
	way 'cue mod tidy' does, without needing the cue binary.

	Every dependency an import needs is pinned at the version minimum version
	selection picks; a dependency imported for the first time is added at its
	newest published version; a dependency nothing imports is removed. An
	already tidy catalog is left untouched (no write, no mtime change).

	The path must be the catalog's module root itself; parent directories are
	not searched. Modules resolve through the same registry every other opm
	command reads (--registry, then OPM_REGISTRY, then the config file).

	With --check nothing is written: the command fails when tidying would
	change a file, for use as a CI gate.

	Exit codes: 0 tidied (or already tidy, or check passed), 1 dependency
	resolution or registry failure, 2 not tidy under --check or not a module
	root.

	Arguments:
	  path    Path to the catalog directory (default: current directory)

	Examples:
	  # Tidy the catalog in the current directory
	  opm catalog tidy

	  # Fail in CI when a dependency is missing or unused
	  opm catalog tidy --check ./src`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			return cmdutil.RunTidy(c.Context(), cfg, publish.KindCatalog, args, check)
		},
	}

	c.Flags().BoolVar(&check, "check", false,
		"Fail with exit 2 if tidying would change any file; write nothing")

	return c
}
