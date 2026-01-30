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

// NewModApplyStubCmd creates a stub for mod apply.
func NewModApplyStubCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "apply",
		Short: "Apply manifests to cluster (not implemented)",
		Long:  "Apply rendered manifests to a Kubernetes cluster.\n\n" + stubMessage,
		RunE: func(cmd *cobra.Command, args []string) error {
			output.Println("opm mod apply: " + stubMessage)
			return nil
		},
	}
}

// NewModDeleteStubCmd creates a stub for mod delete.
func NewModDeleteStubCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete",
		Short: "Delete resources from cluster (not implemented)",
		Long:  "Delete resources created by this module from the cluster.\n\n" + stubMessage,
		RunE: func(cmd *cobra.Command, args []string) error {
			output.Println("opm mod delete: " + stubMessage)
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
