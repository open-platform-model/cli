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

// NewModDiffStubCmd creates a stub for mod diff.
func NewModDiffStubCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "diff",
		Short: "Show differences with cluster (not implemented)",
		Long:  "Show differences between local module and cluster state.\n\n" + stubMessage,
		RunE: func(cmd *cobra.Command, args []string) error {
			output.Println("opm mod diff: " + stubMessage)
			return nil
		},
	}
}

// NewModStatusStubCmd creates a stub for mod status.
func NewModStatusStubCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show resource status (not implemented)",
		Long:  "Show status of resources created by this module.\n\n" + stubMessage,
		RunE: func(cmd *cobra.Command, args []string) error {
			output.Println("opm mod status: " + stubMessage)
			return nil
		},
	}
}
