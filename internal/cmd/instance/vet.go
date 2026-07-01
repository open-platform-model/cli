package instance

import (
	"context"
	"fmt"

	opmexit "github.com/open-platform-model/cli/internal/exit"

	"github.com/spf13/cobra"

	"github.com/open-platform-model/cli/internal/cmdutil"
	"github.com/open-platform-model/cli/internal/config"
	"github.com/open-platform-model/cli/internal/output"
	"github.com/open-platform-model/cli/internal/workflow/render"
)

// NewInstanceVetCmd creates the instance vet command.
func NewInstanceVetCmd(cfg *config.GlobalConfig) *cobra.Command {
	var rff cmdutil.InstanceFileFlags
	var namespace string

	c := &cobra.Command{
		Use:   "vet <instance.cue>",
		Short: "Validate instance file without generating manifests",
		Long: `Validate an OPM instance file via the render pipeline.

This command loads an instance file, matches components to transformers, and
validates the instance can be rendered successfully.
No manifests are output — purely a pass/fail validation tool.

Arguments:
  instance.cue    Path to the instance .cue file (required)

Examples:
  # Validate an instance file
  opm instance vet ./jellyfin_instance.cue

  # Validate with a specific namespace
  opm instance vet ./jellyfin_instance.cue -n production`,
		Args: cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			return runInstanceVet(args[0], cfg, &rff, namespace)
		},
	}

	rff.AddTo(c)
	c.Flags().StringVarP(&namespace, "namespace", "n", "", "Target namespace")

	return c
}

// runInstanceVet executes the instance vet command.
func runInstanceVet(instanceFile string, cfg *config.GlobalConfig, rff *cmdutil.InstanceFileFlags, namespaceFlag string) error {
	ctx := context.Background()

	k8sConfig, err := config.ResolveKubernetes(config.ResolveKubernetesOptions{
		Config:        cfg,
		NamespaceFlag: namespaceFlag,
		ProviderFlag:  rff.Provider,
	})
	if err != nil {
		return &opmexit.ExitError{Code: opmexit.ExitGeneralError, Err: fmt.Errorf("resolving kubernetes config: %w", err)}
	}

	result, err := render.FromInstanceFile(ctx, render.InstanceFileOpts{
		InstanceFilePath: instanceFile,
		ValuesFiles:      rff.Values,
		K8sConfig:        k8sConfig,
		Config:           cfg,
	})
	if err != nil {
		return err
	}

	render.ShowOutput(result, render.ShowOutputOpts{Verbose: cfg.Flags.Verbose})

	instanceLog := output.InstanceLogger(result.Instance.Name)

	// Print per-resource validation lines (skip when --verbose already showed them)
	if !cfg.Flags.Verbose {
		for _, res := range result.Resources {
			line := output.FormatResourceLine(res.GetKind(), res.GetNamespace(), res.GetName(), output.StatusValid)
			instanceLog.Info(line)
		}
	}

	summary := fmt.Sprintf("Instance valid (%d resources)", result.ResourceCount())
	instanceLog.Info(output.FormatCheckmark(summary))

	return nil
}
