// Package cmd provides CLI command implementations.
package cmd

import (
	"github.com/spf13/cobra"

	"github.com/opmodel/cli/internal/output"
)

const stubMessage = `This command is not yet implemented.

Implementation is specified in 004-render-and-lifecycle-spec.
See: opm/specs/cli/004-render-and-lifecycle-spec/spec.md`

// NewModBuildStubCmd creates a stub for mod build.
func NewModBuildStubCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "build",
		Short: "Render module to manifests (not implemented)",
		Long:  "Render an OPM module to Kubernetes manifests.\n\n" + stubMessage,
		RunE: func(cmd *cobra.Command, args []string) error {
			output.Println("opm mod build: " + stubMessage)
			return nil
		},
	}
}
