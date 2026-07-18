package instance

import (
	"context"
	"fmt"

	opmexit "github.com/open-platform-model/cli/internal/exit"

	"github.com/spf13/cobra"

	"github.com/open-platform-model/cli/internal/cmdutil"
	"github.com/open-platform-model/cli/internal/config"
	"github.com/open-platform-model/cli/internal/kubernetes"
	"github.com/open-platform-model/cli/internal/output"
	"github.com/open-platform-model/cli/internal/workflow/query"
)

// NewInstanceTreeCmd creates the instance tree command.
func NewInstanceTreeCmd(cfg *config.GlobalConfig) *cobra.Command {
	var kf cmdutil.K8sFlags
	var namespace string
	var depthFlag int
	var outputFlag string

	c := &cobra.Command{
		Use:   "tree <file|name|uuid>",
		Short: "Show resource hierarchy for an instance",
		Long: `Show the component and resource hierarchy of a deployed OPM instance.

Arguments:
  file         Path to an instance.cue file or directory containing one.
               The instance name and namespace are read from the file's metadata.
               --namespace overrides the namespace found in the file.
  name         Instance name (use -n / --namespace to scope by namespace).
  uuid         Instance UUID.

Examples:
  # Identify by instance.cue file in the current directory
  opm instance tree .

  # Identify by instance.cue file path
  opm instance tree ./instances/jellyfin/instance.cue

  # Identify by name, full tree (default depth=2)
  opm instance tree jellyfin -n media

  # Component summary only
  opm instance tree jellyfin -n media --depth 0`,
		Args: cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			return runInstanceTree(args[0], cfg, &kf, namespace, depthFlag, outputFlag)
		},
	}

	kf.AddTo(c)
	c.Flags().StringVarP(&namespace, "namespace", "n", "", "Target namespace")
	c.Flags().IntVar(&depthFlag, "depth", 2, "Tree depth: 0=summary, 1=resources, 2=full hierarchy")
	c.Flags().StringVarP(&outputFlag, "output", "o", "table", "Output format (table, json, yaml)")

	return c
}

func runInstanceTree(identifier string, cfg *config.GlobalConfig, kf *cmdutil.K8sFlags, namespaceFlag string, depth int, outputFmt string) error {
	ctx := context.Background()

	if depth < 0 || depth > 2 {
		return &opmexit.ExitError{
			Code: opmexit.ExitGeneralError,
			Err:  fmt.Errorf("invalid --depth %d: must be 0, 1, or 2", depth),
		}
	}

	outputFormat, valid := output.ParseFormat(outputFmt)
	if !valid || outputFormat == output.FormatWide || outputFormat == output.FormatDir {
		return &opmexit.ExitError{
			Code: opmexit.ExitGeneralError,
			Err:  fmt.Errorf("invalid output format %q (valid: table, json, yaml)", outputFmt),
		}
	}

	target, err := cmdutil.ResolveInstanceTarget(identifier, cfg, kf, namespaceFlag)
	if err != nil {
		return err
	}

	namespace := target.Namespace
	logName := target.LogName
	instanceLog := output.InstanceLogger(logName)

	k8sClient, err := cmdutil.NewK8sClient(target.K8sConfig, cfg.Log.Kubernetes.APIWarnings)
	if err != nil {
		instanceLog.Error("connecting to cluster", "error", err)
		return err
	}

	inv, liveResources, _, err := query.ResolveInventory(ctx, k8sClient, target.Selector, namespace, instanceLog)
	if err != nil {
		return err
	}

	componentMap := make(map[string]string)
	for _, entry := range inv.Inventory.Entries {
		key := entry.Kind + "/" + entry.Namespace + "/" + entry.Name
		componentMap[key] = entry.Component
	}

	instanceInfo := kubernetes.InstanceInfo{
		Name:      inv.Name,
		Namespace: inv.Namespace,
		Module:    inv.ModulePath,
		Version:   inv.ModuleVersion,
	}

	treeOpts := kubernetes.TreeOptions{
		InstanceInfo:  instanceInfo,
		InventoryLive: liveResources,
		ComponentMap:  componentMap,
		Depth:         depth,
		OutputFormat:  outputFormat,
	}

	result, err := kubernetes.GetModuleTree(ctx, k8sClient, treeOpts)
	if err != nil {
		if kubernetes.IsNoResourcesFound(err) {
			instanceLog.Error("no resources found", "instance", logName, "namespace", namespace)
			return &opmexit.ExitError{Code: opmexit.ExitNotFound, Err: err, Printed: true}
		}
		instanceLog.Error("getting tree", "error", err)
		return &opmexit.ExitError{Code: cmdutil.ExitCodeFromK8sError(err), Err: err, Printed: true}
	}

	formatted, err := kubernetes.FormatTree(result, outputFormat)
	if err != nil {
		instanceLog.Error("formatting tree", "error", err)
		return &opmexit.ExitError{Code: opmexit.ExitGeneralError, Err: err, Printed: true}
	}

	output.Println(formatted)
	return nil
}
