package modulecmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/opmodel/cli/internal/cmdutil"
	"github.com/opmodel/cli/internal/config"
	"github.com/opmodel/cli/internal/output"
	workflowapply "github.com/opmodel/cli/internal/workflow/apply"
	"github.com/opmodel/cli/internal/workflow/render"
	oerrors "github.com/opmodel/cli/pkg/errors"
)

// NewModuleApplyCmd creates the module apply command.
func NewModuleApplyCmd(cfg *config.GlobalConfig) *cobra.Command {
	var rf cmdutil.RenderFlags
	var kf cmdutil.K8sFlags

	// Apply-specific flags (local to this command)
	var (
		dryRunFlag     bool
		createNSFlag   bool
		noPruneFlag    bool
		maxHistoryFlag int
		forceFlag      bool
	)

	c := &cobra.Command{
		Use:   "apply [path]",
		Short: "Deploy module to cluster",
		Long: `Deploy an OPM module to a Kubernetes cluster using server-side apply.

This command renders the module and applies the resulting resources to the
cluster. Resources are applied in weight order (CRDs first, webhooks last).

All resources are labeled with OPM metadata for later discovery by
'opm module delete' and 'opm module status'.

An inventory Secret is written after each successful apply to record the
exact set of applied resources. On subsequent applies, stale resources
(present in a previous apply but not in the current render) are pruned.

Arguments:
  path    Path to module directory (default: current directory)

Examples:
  # Apply module in current directory
  opm module apply

  # Apply with custom values and namespace
  opm module apply ./my-module -f prod-values.cue -n production

  # Preview what would be applied
  opm module apply --dry-run

  # Apply without pruning stale resources
  opm module apply --no-prune

  # Apply with verbose output showing transformer matches
  opm module apply --verbose`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			return runModuleApply(args, cfg, &rf, &kf, dryRunFlag, createNSFlag, noPruneFlag, maxHistoryFlag, forceFlag)
		},
	}

	rf.AddTo(c)
	kf.AddTo(c)

	// Apply-specific flags
	c.Flags().BoolVar(&dryRunFlag, "dry-run", false,
		"Server-side dry run (no changes made)")
	c.Flags().BoolVar(&createNSFlag, "create-namespace", false,
		"Create target namespace if it does not exist")
	c.Flags().BoolVar(&noPruneFlag, "no-prune", false,
		"Skip stale resource pruning (stale resources remain on cluster)")
	c.Flags().IntVar(&maxHistoryFlag, "max-history", 10,
		"Maximum number of change history entries to retain in inventory")
	c.Flags().BoolVar(&forceFlag, "force", false,
		"Allow empty render to prune all previously tracked resources")

	return c
}

// runApply executes the apply command with the 8-step inventory-aware flow:
//
//  1. Render resources
//  2. Compute manifest digest
//  3. Compute change ID
//  4. Read previous inventory
//     5a. Compute stale set
//     5b. Apply component-rename safety check
//     5c. Pre-apply existence check (first-time only)
//  6. Apply all rendered resources via SSA
//     7a. Prune stale resources (if all applied successfully and --no-prune not set)
//     7b. Skip prune and inventory write if any apply failed
//  8. Write inventory Secret with new change entry
func runModuleApply(args []string, cfg *config.GlobalConfig, rf *cmdutil.RenderFlags, kf *cmdutil.K8sFlags,
	dryRun, createNS, noPrune bool, maxHistory int, force bool) error {
	ctx := context.Background()

	// Resolve all Kubernetes configuration once (flag > env > config > default).
	// The resolved values are used for both the render pipeline and K8s client,
	// ensuring a single source of truth for all settings.
	k8sConfig, err := config.ResolveKubernetes(config.ResolveKubernetesOptions{
		Config:         cfg,
		KubeconfigFlag: kf.Kubeconfig,
		ContextFlag:    kf.Context,
		NamespaceFlag:  rf.Namespace,
		ProviderFlag:   rf.Provider,
	})
	if err != nil {
		return &oerrors.ExitError{Code: oerrors.ExitGeneralError, Err: fmt.Errorf("resolving kubernetes config: %w", err)}
	}

	// Step 1: Render module via shared pipeline.
	// DebugValues: when no -f flag is given, use the module's debugValues field
	// as the values source (consistent with how opm module vet works in release mode).
	result, err := render.Release(ctx, render.ReleaseOpts{
		Args:        args,
		Values:      rf.Values,
		ReleaseName: rf.ReleaseName,
		K8sConfig:   k8sConfig,
		Config:      cfg,
		DebugValues: len(rf.Values) == 0,
	})
	if err != nil {
		return err
	}

	// Post-render: check errors, show matches, log warnings
	render.ShowOutput(result, render.ShowOutputOpts{Verbose: cfg.Flags.Verbose})

	// Create scoped module logger
	releaseLog := output.ReleaseLogger(result.Release.Name)

	k8sClient, err := cmdutil.NewK8sClient(k8sConfig, cfg.Log.Kubernetes.APIWarnings)
	if err != nil {
		releaseLog.Error("connecting to cluster", "error", err)
		return err
	}
	modulePath := ""
	if len(args) > 0 {
		modulePath = args[0]
	}
	valuesStr := strings.Join(rf.Values, ",")

	return workflowapply.Execute(ctx, workflowapply.Request{
		Result:    result,
		K8sClient: k8sClient,
		Log:       releaseLog,
		Options: workflowapply.Options{
			DryRun:                 dryRun,
			CreateNS:               createNS,
			NoPrune:                noPrune,
			MaxHistory:             maxHistory,
			Force:                  force,
			SuccessUpToDateMessage: "Module up to date",
			SuccessAppliedMessage:  "Module applied",
		},
		Change: workflowapply.ChangeDescriptor{
			Path:      modulePath,
			ValuesStr: valuesStr,
			Version:   result.Module.Version,
			Local:     result.Module.Version == "",
		},
		ModuleName: result.Module.Name,
		ModuleUUID: result.Module.UUID,
	})
}
