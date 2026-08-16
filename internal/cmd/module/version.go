package modulecmd

import (
	"github.com/spf13/cobra"

	"github.com/open-platform-model/cli/internal/cmdutil"
	"github.com/open-platform-model/cli/internal/publish"
)

// NewModuleVersionCmd creates the module version command group. It takes no
// config: version commands are offline by design (0011 D3/D8) — no registry,
// no schema fetch.
func NewModuleVersionCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "version",
		Short: "Manage the module's declared version",
		Long: `Manage the version an OPM module declares in identity/identity.cue.

	The declared version is the single source the publish pipeline reads; these
	commands edit it deliberately so a commit sits between deciding a version
	and pushing an artifact.`,
	}

	c.AddCommand(newModuleVersionSetCmd())

	return c
}

func newModuleVersionSetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "set <version> [path]",
		Short: "Set the module's declared version",
		Long: `Set the version the module declares in identity/identity.cue.

	The write is surgical: only the version value changes — comments, field
	order, alignment, any type assertion on the field, and release-automation
	marker lines all survive byte-for-byte. Setting the version the file
	already declares writes nothing (no mtime change) and reports the no-op.

	The command is offline: no registry access, no schema fetch. It refuses an
	identity file that does not structurally carry a Version field; run
	'opm module vet' for full conformance checking.

	Exit codes: 0 set (or already set), 2 refused.

	Arguments:
	  version    Bare SemVer to declare (no "v" prefix)
	  path       Path to the module directory (default: current directory)

	Examples:
	  # Declare the next release version in the current directory
	  opm module version set 1.3.0

	  # Same, against an explicit module directory
	  opm module version set 1.3.0 ./my-module`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(c *cobra.Command, args []string) error {
			return cmdutil.RunVersionSet(publish.KindModule, args[0], args[1:])
		},
	}
}
