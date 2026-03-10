package release

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

// NewReleaseTreeCmd creates the release tree command.
func NewReleaseTreeCmd(cfg *config.GlobalConfig) *cobra.Command {
	var kf cmdutil.K8sFlags
	var namespace string
	var depthFlag int
	var outputFlag string

	c := &cobra.Command{
		Use:   "tree <file|name|uuid>",
		Short: "Show resource hierarchy for a release",
		Long: `Show the component and resource hierarchy of a deployed OPM release.

Arguments:
  file         Path to a release.cue file or directory containing one.
               The release name and namespace are read from the file's metadata.
               --namespace overrides the namespace found in the file.
  name         Release name (use -n / --namespace to scope by namespace).
  uuid         Release UUID.

Examples:
  # Identify by release.cue file in the current directory
  opm release tree .

  # Identify by release.cue file path
  opm release tree ./releases/jellyfin/release.cue

  # Identify by name, full tree (default depth=2)
  opm release tree jellyfin -n media

  # Component summary only
  opm release tree jellyfin -n media --depth 0`,
		Args: cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			return runReleaseTree(args[0], cfg, &kf, namespace, depthFlag, outputFlag)
		},
	}

	kf.AddTo(c)
	c.Flags().StringVarP(&namespace, "namespace", "n", "", "Target namespace")
	c.Flags().IntVar(&depthFlag, "depth", 2, "Tree depth: 0=summary, 1=resources, 2=full hierarchy")
	c.Flags().StringVarP(&outputFlag, "output", "o", "table", "Output format (table, json, yaml)")

	return c
}

func runReleaseTree(identifier string, cfg *config.GlobalConfig, kf *cmdutil.K8sFlags, namespaceFlag string, depth int, outputFmt string) error { //nolint:gocyclo // mirrors runTree complexity
	ctx := context.Background()

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

	target, err := cmdutil.ResolveReleaseTarget(identifier, cfg, kf, namespaceFlag)
	if err != nil {
		return err
	}

	namespace := target.Namespace
	logName := target.LogName
	releaseLog := output.ReleaseLogger(logName)

	k8sClient, err := cmdutil.NewK8sClient(target.K8sConfig, cfg.Log.Kubernetes.APIWarnings)
	if err != nil {
		releaseLog.Error("connecting to cluster", "error", err)
		return err
	}

	inv, liveResources, _, err := cmdutil.ResolveInventory(ctx, k8sClient, target.Selector, namespace, releaseLog)
	if err != nil {
		return err
	}

	componentMap := make(map[string]string)
	if len(inv.Index) > 0 {
		if change, ok := inv.Changes[inv.Index[0]]; ok {
			for _, entry := range change.Inventory.Entries {
				key := entry.Kind + "/" + entry.Namespace + "/" + entry.Name
				componentMap[key] = entry.Component
			}
		}
	}

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

	formatted, err := kubernetes.FormatTree(result, outputFormat)
	if err != nil {
		releaseLog.Error("formatting tree", "error", err)
		return &oerrors.ExitError{Code: oerrors.ExitGeneralError, Err: err, Printed: true}
	}

	output.Println(formatted)
	return nil
}
