package release

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/opmodel/cli/internal/cmdutil"
	"github.com/opmodel/cli/internal/config"
	"github.com/opmodel/cli/internal/inventory"
	"github.com/opmodel/cli/internal/output"
	oerrors "github.com/opmodel/cli/pkg/errors"
)

const releaseListConcurrency = 5

// NewReleaseListCmd creates the release list command.
func NewReleaseListCmd(cfg *config.GlobalConfig) *cobra.Command {
	var kf cmdutil.K8sFlags
	var namespace string
	var allNamespaces bool
	var outputFlag string

	c := &cobra.Command{
		Use:   "list",
		Short: "List deployed releases",
		Long: `List all deployed OPM releases in a namespace.

Examples:
  # List releases in the default namespace
  opm release list

  # List releases in a specific namespace
  opm release list -n production

  # List across all namespaces
  opm release list -A`,
		RunE: func(c *cobra.Command, args []string) error {
			return runReleaseList(cfg, &kf, namespace, allNamespaces, outputFlag)
		},
	}

	kf.AddTo(c)
	c.Flags().StringVarP(&namespace, "namespace", "n", "", "Kubernetes namespace (default from config)")
	c.Flags().BoolVarP(&allNamespaces, "all-namespaces", "A", false, "List releases across all namespaces")
	c.Flags().StringVarP(&outputFlag, "output", "o", "table", "Output format (table, wide, yaml, json)")

	return c
}

func runReleaseList(cfg *config.GlobalConfig, kf *cmdutil.K8sFlags, namespaceFlag string, allNamespaces bool, outputFmt string) error {
	ctx := context.Background()

	outputFormat, valid := output.ParseFormat(outputFmt)
	if !valid || outputFormat == output.FormatDir {
		return &oerrors.ExitError{
			Code: oerrors.ExitGeneralError,
			Err:  fmt.Errorf("invalid output format %q (valid: table, wide, yaml, json)", outputFmt),
		}
	}

	k8sConfig, err := config.ResolveKubernetes(config.ResolveKubernetesOptions{
		Config:         cfg,
		KubeconfigFlag: kf.Kubeconfig,
		ContextFlag:    kf.Context,
		NamespaceFlag:  namespaceFlag,
	})
	if err != nil {
		return &oerrors.ExitError{Code: oerrors.ExitGeneralError, Err: fmt.Errorf("resolving kubernetes config: %w", err)}
	}

	targetNamespace := k8sConfig.Namespace.Value
	if allNamespaces {
		targetNamespace = ""
	} else {
		if err := cmdutil.RequireNamespace(k8sConfig); err != nil {
			return err
		}
	}

	k8sClient, err := cmdutil.NewK8sClient(k8sConfig, cfg.Log.Kubernetes.APIWarnings)
	if err != nil {
		return err
	}

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

	summaries := cmdutil.EvaluateReleaseHealth(ctx, k8sClient, inventories, releaseListConcurrency, false)
	return cmdutil.RenderReleaseListOutput(summaries, outputFormat, allNamespaces)
}
