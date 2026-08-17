package modulecmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/open-platform-model/cli/internal/output"
	"github.com/open-platform-model/cli/internal/scaffold"
)

// NewModuleTemplateCmd creates the module template command group.
func NewModuleTemplateCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "template",
		Short: "Work with the official module templates",
		Long: `Work with the official module templates — the curated set published to
the reserved opmodel.dev/templates segment by the cli's own release
pipeline (0011 D25).`,
	}
	c.AddCommand(newModuleTemplateListCmd())
	return c
}

// newModuleTemplateListCmd lists the official templates from the baked table
// — the same table that drives shortcut expansion, so what this prints is
// exactly what `opm mod init <name>` resolves. Offline by construction: the
// binary that knows the table belongs to the release train that published
// the templates.
func newModuleTemplateListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List the official module templates",
		Long: `List the official module templates: name, description, and the default
major an unsuffixed shortcut floats within. Every name is usable as an
'opm mod init' template shortcut. Offline — the table ships in the
binary, release-coupled to the published set.`,
		Args: cobra.NoArgs,
		RunE: func(c *cobra.Command, _ []string) error {
			t := output.NewTable("NAME", "DESCRIPTION", "DEFAULT MAJOR")
			for _, tpl := range scaffold.Official {
				t.Row(tpl.Name, tpl.Description, tpl.DefaultMajor)
			}
			fmt.Fprint(c.OutOrStdout(), t.String())
			return nil
		},
	}
}
