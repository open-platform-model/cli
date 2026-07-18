package instance

import (
	"context"
	"fmt"

	opmexit "github.com/open-platform-model/cli/internal/exit"

	"github.com/spf13/cobra"

	"github.com/open-platform-model/cli/internal/cmdutil"
	"github.com/open-platform-model/cli/internal/config"
	"github.com/open-platform-model/cli/internal/inventory"
	"github.com/open-platform-model/cli/internal/output"
	"github.com/open-platform-model/cli/internal/workflow/query"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

const instanceListConcurrency = 5

// NewInstanceListCmd creates the instance list command.
func NewInstanceListCmd(cfg *config.GlobalConfig) *cobra.Command {
	var kf cmdutil.K8sFlags
	var namespace string
	var allNamespaces bool
	var outputFlag string

	c := &cobra.Command{
		Use:   "list",
		Short: "List deployed instances",
		Long: `List all deployed OPM instances in a namespace.

Examples:
  # List instances in the default namespace
  opm instance list

  # List instances in a specific namespace
  opm instance list -n production

  # List across all namespaces
  opm instance list -A`,
		RunE: func(c *cobra.Command, args []string) error {
			return runInstanceList(cfg, &kf, namespace, allNamespaces, outputFlag)
		},
	}

	kf.AddTo(c)
	c.Flags().StringVarP(&namespace, "namespace", "n", "", "Kubernetes namespace (default from config)")
	c.Flags().BoolVarP(&allNamespaces, "all-namespaces", "A", false, "List instances across all namespaces")
	c.Flags().StringVarP(&outputFlag, "output", "o", "table", "Output format (table, wide, yaml, json)")

	return c
}

func runInstanceList(cfg *config.GlobalConfig, kf *cmdutil.K8sFlags, namespaceFlag string, allNamespaces bool, outputFmt string) error {
	ctx := context.Background()

	outputFormat, valid := output.ParseFormat(outputFmt)
	if !valid || outputFormat == output.FormatDir {
		return &opmexit.ExitError{
			Code: opmexit.ExitGeneralError,
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
		return &opmexit.ExitError{Code: opmexit.ExitGeneralError, Err: fmt.Errorf("resolving kubernetes config: %w", err)}
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

	inventories, err := inventory.ListRecords(ctx, k8sClient, targetNamespace)
	if err != nil {
		if allNamespaces && apierrors.IsForbidden(err) {
			return &opmexit.ExitError{
				Code:    opmexit.ExitPermissionDenied,
				Err:     fmt.Errorf("listing instances across all namespaces requires cluster-wide 'list' permission on moduleinstances.opmodel.dev: %w", err),
				Printed: true,
			}
		}
		return &opmexit.ExitError{Code: cmdutil.ExitCodeFromK8sError(err), Err: fmt.Errorf("listing instances: %w", err)}
	}

	if len(inventories) == 0 {
		if allNamespaces {
			output.Println("No instances found")
		} else {
			output.Println(fmt.Sprintf("No instances found in namespace %q", k8sConfig.Namespace.Value))
		}
		return nil
	}

	summaries := query.EvaluateInstanceHealth(ctx, k8sClient, inventories, instanceListConcurrency, false)
	return query.RenderInstanceListOutput(summaries, outputFormat, allNamespaces)
}
