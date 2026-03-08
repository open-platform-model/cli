package mod

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/opmodel/cli/internal/cmdutil"
	"github.com/opmodel/cli/internal/config"
	"github.com/opmodel/cli/internal/kubernetes"
	"github.com/opmodel/cli/internal/output"
	oerrors "github.com/opmodel/cli/pkg/errors"
)

// NewModTreeCmd creates the mod tree command.
func NewModTreeCmd(cfg *config.GlobalConfig) *cobra.Command {
	var rsf cmdutil.ReleaseSelectorFlags
	var kf cmdutil.K8sFlags

	var (
		depthFlag  int
		outputFlag string
	)

	c := &cobra.Command{
		Use:        "tree",
		Deprecated: "use 'opm release tree <name>' instead",
		Short:      "Show module resource hierarchy",
		Long: `Show the component and resource hierarchy of a deployed OPM release.

Resources are grouped by component and displayed as a tree. Kubernetes ownership
chains (Deployment→ReplicaSet→Pod, StatefulSet→Pod) are walked at depth=2.

Exactly one of --release-name or --release-id is required to identify the release.
The --namespace flag defaults to the value configured in ~/.opm/config.cue.

Depth levels:
  0  Component summary only (resource counts and aggregate status)
  1  OPM-managed resources grouped by component
  2  Full tree with Kubernetes-owned children (default)

Exit codes:
  0  Success
  1  Command error (invalid flags, cluster unreachable, etc.)
  5  No resources found for release

Examples:
  # Full tree (default depth=2)
  opm mod tree --release-name my-app -n production

  # Component summary only
  opm mod tree --release-name my-app -n production --depth 0

  # Resources without K8s children
  opm mod tree --release-name my-app -n production --depth 1

  # JSON output
  opm mod tree --release-name my-app -n production -o json

  # Select by release ID
  opm mod tree --release-id a1b2c3d4-e5f6-7890-abcd-ef1234567890 -n production`,
		RunE: func(c *cobra.Command, args []string) error {
			return runTree(args, cfg, &rsf, &kf, depthFlag, outputFlag)
		},
	}

	rsf.AddTo(c)
	kf.AddTo(c)

	c.Flags().IntVar(&depthFlag, "depth", 2,
		"Tree depth: 0=summary, 1=resources, 2=full hierarchy with K8s children")
	c.Flags().StringVarP(&outputFlag, "output", "o", "table",
		"Output format (table, json, yaml)")

	return c
}

// runTree executes the tree command.
//
//nolint:gocyclo // orchestration function; complexity is inherent
func runTree(_ []string, cfg *config.GlobalConfig, rsf *cmdutil.ReleaseSelectorFlags, kf *cmdutil.K8sFlags, depth int, outputFmt string) error {
	ctx := context.Background()

	// Validate depth and output format first (fast, no I/O).
	if depth < 0 || depth > 2 {
		return &oerrors.ExitError{
			Code: oerrors.ExitGeneralError,
			Err:  fmt.Errorf("invalid --depth %d: must be 0, 1, or 2", depth),
		}
	}

	outputFormat, valid := output.ParseFormat(outputFmt)
	if !valid || outputFormat == output.FormatWide || outputFormat == output.FormatDir {
		return &oerrors.ExitError{
			Code: oerrors.ExitGeneralError,
			Err:  fmt.Errorf("invalid output format %q (valid: table, json, yaml)", outputFmt),
		}
	}

	// Validate release selector flags.
	if err := rsf.Validate(); err != nil {
		return &oerrors.ExitError{Code: oerrors.ExitGeneralError, Err: err}
	}

	// Resolve Kubernetes configuration.
	k8sConfig, err := config.ResolveKubernetes(config.ResolveKubernetesOptions{
		Config:         cfg,
		KubeconfigFlag: kf.Kubeconfig,
		ContextFlag:    kf.Context,
		NamespaceFlag:  rsf.Namespace,
	})
	if err != nil {
		return &oerrors.ExitError{Code: oerrors.ExitGeneralError, Err: fmt.Errorf("resolving kubernetes config: %w", err)}
	}
	if err := cmdutil.RequireNamespace(k8sConfig); err != nil {
		return err
	}

	namespace := k8sConfig.Namespace.Value

	output.Debug("resolved kubernetes config",
		"kubeconfig", k8sConfig.Kubeconfig.Value,
		"context", k8sConfig.Context.Value,
		"namespace", namespace,
	)

	logName := rsf.LogName()
	releaseLog := output.ReleaseLogger(logName)

	// Create Kubernetes client.
	k8sClient, err := cmdutil.NewK8sClient(k8sConfig, cfg.Log.Kubernetes.APIWarnings)
	if err != nil {
		releaseLog.Error("connecting to cluster", "error", err)
		return err
	}

	// Resolve inventory and live resources.
	inv, liveResources, _, err := cmdutil.ResolveInventory(ctx, k8sClient, rsf, namespace, releaseLog)
	if err != nil {
		return err
	}

	// Build ComponentMap from inventory entries (same pattern as status.go).
	componentMap := make(map[string]string)
	if len(inv.Index) > 0 {
		if change, ok := inv.Changes[inv.Index[0]]; ok {
			for _, entry := range change.Inventory.Entries {
				key := entry.Kind + "/" + entry.Namespace + "/" + entry.Name
				componentMap[key] = entry.Component
			}
		}
	}

	// Build ReleaseInfo from inventory metadata.
	releaseInfo := kubernetes.ReleaseInfo{
		Name:      inv.ReleaseMetadata.ReleaseName,
		Namespace: inv.ReleaseMetadata.ReleaseNamespace,
		Module:    inv.ModuleMetadata.Name,
	}
	if len(inv.Index) > 0 {
		if change, ok := inv.Changes[inv.Index[0]]; ok {
			releaseInfo.Version = change.Source.Version
		}
	}

	// Build tree options and call kubernetes layer.
	treeOpts := kubernetes.TreeOptions{
		ReleaseInfo:   releaseInfo,
		InventoryLive: liveResources,
		ComponentMap:  componentMap,
		Depth:         depth,
		OutputFormat:  outputFormat,
	}

	result, err := kubernetes.GetModuleTree(ctx, k8sClient, treeOpts)
	if err != nil {
		if kubernetes.IsNoResourcesFound(err) {
			releaseLog.Error("no resources found", "release", logName, "namespace", namespace)
			return &oerrors.ExitError{Code: oerrors.ExitNotFound, Err: err, Printed: true}
		}
		releaseLog.Error("getting tree", "error", err)
		return &oerrors.ExitError{Code: cmdutil.ExitCodeFromK8sError(err), Err: err, Printed: true}
	}

	// Format and print.
	formatted, err := kubernetes.FormatTree(result, outputFormat)
	if err != nil {
		releaseLog.Error("formatting tree", "error", err)
		return &oerrors.ExitError{Code: oerrors.ExitGeneralError, Err: err, Printed: true}
	}

	output.Println(formatted)
	return nil
}
