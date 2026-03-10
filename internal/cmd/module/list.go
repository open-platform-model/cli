package modulecmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/opmodel/cli/internal/cmdutil"
	"github.com/opmodel/cli/internal/config"
	"github.com/opmodel/cli/internal/inventory"
	"github.com/opmodel/cli/internal/output"
	"github.com/opmodel/cli/internal/workflow/query"
	oerrors "github.com/opmodel/cli/pkg/errors"
)

// listConcurrency is the maximum number of concurrent release health evaluations.
const listConcurrency = 5

// NewModuleListCmd creates the module list command.
func NewModuleListCmd(cfg *config.GlobalConfig) *cobra.Command {
	var kf cmdutil.K8sFlags

	var (
		namespace     string
		allNamespaces bool
		outputFlag    string
	)

	c := &cobra.Command{
		Use:   "list",
		Short: "List deployed module releases",
		Long: `List all deployed module releases in a namespace.

Releases are discovered via inventory Secrets. Health status is evaluated
for each release by checking the state of its tracked resources.

By default, releases are listed in the namespace configured in ~/.opm/config.cue
(or "default"). Use -A to list releases across all namespaces.

Examples:
  # List releases in the default namespace
  opm module list

  # List releases in a specific namespace
  opm module list -n production

  # List releases across all namespaces
  opm module list -A

  # Wide output with release ID and last applied time
  opm module list -o wide

  # Machine-readable output
  opm module list -o json
  opm module list -o yaml`,
		RunE: func(c *cobra.Command, args []string) error {
			return runModuleList(cfg, &kf, namespace, allNamespaces, outputFlag)
		},
	}

	kf.AddTo(c)

	c.Flags().StringVarP(&namespace, "namespace", "n", "", "Kubernetes namespace (default from config)")
	c.Flags().BoolVarP(&allNamespaces, "all-namespaces", "A", false, "List releases across all namespaces")
	c.Flags().StringVarP(&outputFlag, "output", "o", "table", "Output format (table, wide, yaml, json)")

	return c
}

// runList executes the list command.
func runModuleList(cfg *config.GlobalConfig, kf *cmdutil.K8sFlags, namespaceFlag string, allNamespaces bool, outputFmt string) error {
	ctx := context.Background()

	// Validate output format
	outputFormat, valid := output.ParseFormat(outputFmt)
	if !valid || outputFormat == output.FormatDir {
		return &oerrors.ExitError{
			Code: oerrors.ExitGeneralError,
			Err:  fmt.Errorf("invalid output format %q (valid: table, wide, yaml, json)", outputFmt),
		}
	}

	// Resolve Kubernetes configuration
	k8sConfig, err := config.ResolveKubernetes(config.ResolveKubernetesOptions{
		Config:         cfg,
		KubeconfigFlag: kf.Kubeconfig,
		ContextFlag:    kf.Context,
		NamespaceFlag:  namespaceFlag,
	})
	if err != nil {
		return &oerrors.ExitError{Code: oerrors.ExitGeneralError, Err: fmt.Errorf("resolving kubernetes config: %w", err)}
	}

	// Determine target namespace
	targetNamespace := k8sConfig.Namespace.Value
	if allNamespaces {
		targetNamespace = "" // K8s convention: empty = all namespaces
	} else {
		if err := cmdutil.RequireNamespace(k8sConfig); err != nil {
			return err
		}
	}

	// Create Kubernetes client
	k8sClient, err := cmdutil.NewK8sClient(k8sConfig, cfg.Log.Kubernetes.APIWarnings)
	if err != nil {
		return err
	}

	// Discover all inventory Secrets
	inventories, err := inventory.ListInventories(ctx, k8sClient, targetNamespace)
	if err != nil {
		return &oerrors.ExitError{Code: cmdutil.ExitCodeFromK8sError(err), Err: fmt.Errorf("listing releases: %w", err)}
	}

	if len(inventories) == 0 {
		if allNamespaces {
			output.Println("No releases found")
		} else {
			output.Println(fmt.Sprintf("No releases found in namespace %q", k8sConfig.Namespace.Value))
		}
		return nil
	}

	summaries := query.EvaluateReleaseHealth(ctx, k8sClient, inventories, listConcurrency, true)

	return query.RenderReleaseListOutput(summaries, outputFormat, allNamespaces)
}
