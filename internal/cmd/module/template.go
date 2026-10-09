package modulecmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/open-platform-model/cli/internal/cmdutil"
	"github.com/open-platform-model/cli/internal/output"
	"github.com/open-platform-model/cli/internal/scaffold"
)

// NewModuleTemplateCmd creates the module template command group. The
// official templates live in the reserved opmodel.dev/templates segment
// (0011:D25).
func NewModuleTemplateCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "template",
		Short: "Work with the official module templates",
		Long: `Work with the official module templates — the curated set published to
the reserved opmodel.dev/templates segment by the cli's own release
pipeline.`,
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
	var outputFlag string

	c := &cobra.Command{
		Use:   "list",
		Short: "List the official module templates",
		Long: `List the official module templates: name, description, and the default
major an unsuffixed shortcut floats within. Every name is usable as an
'opm mod init' template shortcut. Offline — the table ships in the
binary, release-coupled to the published set.`,
		Args: cobra.NoArgs,
		RunE: func(c *cobra.Command, _ []string) error {
			if err := cmdutil.CheckOutputFormat(outputFlag, "table", "json", "yaml"); err != nil {
				return err
			}
			if outputFlag != "table" {
				rows := make([]templateOutput, 0, len(scaffold.Official))
				for _, tpl := range scaffold.Official {
					rows = append(rows, templateOutput{Name: tpl.Name, Description: tpl.Description, DefaultMajor: tpl.DefaultMajor})
				}
				return cmdutil.WriteStructured(c.OutOrStdout(), outputFlag, rows)
			}
			t := output.NewTable("NAME", "DESCRIPTION", "DEFAULT MAJOR")
			for _, tpl := range scaffold.Official {
				t.Row(tpl.Name, tpl.Description, tpl.DefaultMajor)
			}
			fmt.Fprint(c.OutOrStdout(), t.String())
			return nil
		},
	}

	c.Flags().StringVarP(&outputFlag, "output", "o", "table", "Output format (table, json, yaml)")

	return c
}

// templateOutput is one row of 'opm module template list' in its json and
// yaml form. The field names are a contract for scripts.
type templateOutput struct {
	Name         string `json:"name" yaml:"name"`
	Description  string `json:"description" yaml:"description"`
	DefaultMajor string `json:"defaultMajor" yaml:"defaultMajor"`
}
