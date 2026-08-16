package catalogcmd

import (
	"github.com/spf13/cobra"

	"github.com/open-platform-model/cli/internal/cmdutil"
	"github.com/open-platform-model/cli/internal/publish"
)

// NewCatalogVersionCmd creates the catalog version command group. It takes no
// config: version commands are offline by design (0011 D3/D8) — no registry,
// no schema fetch.
func NewCatalogVersionCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "version",
		Short: "Manage the catalog's declared version",
		Long: `Manage the version an OPM catalog declares in identity/identity.cue.

	The declared version is the single source the publish pipeline reads; these
	commands edit it deliberately so a commit sits between deciding a version
	and pushing an artifact.`,
	}

	c.AddCommand(newCatalogVersionSetCmd())

	return c
}

func newCatalogVersionSetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "set <version> [path]",
		Short: "Set the catalog's declared version",
		Long: `Set the version the catalog declares in identity/identity.cue.

	The write is surgical: only the version value changes — comments, field
	order, alignment, any type assertion on the field, and release-automation
	marker lines all survive byte-for-byte. A defaulted declaration (the shape
	release automation owns) stays defaulted with its marker intact. Setting
	the version the file already declares writes nothing (no mtime change) and
	reports the no-op.

	The command is offline: no registry access, no schema fetch. It refuses an
	identity file that does not structurally carry a Version field; run
	'opm catalog publish --dry-run' for full conformance checking — every
	publish gate runs, nothing is pushed.

	Exit codes: 0 set (or already set), 2 refused.

	Arguments:
	  version    Bare SemVer to declare (no "v" prefix)
	  path       Path to the catalog directory (default: current directory)

	Examples:
	  # Declare the next release version in the current directory
	  opm catalog version set 1.3.0

	  # Same, against an explicit catalog directory
	  opm catalog version set 1.3.0 ./src`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(c *cobra.Command, args []string) error {
			return cmdutil.RunVersionSet(publish.KindCatalog, args[0], args[1:])
		},
	}
}
