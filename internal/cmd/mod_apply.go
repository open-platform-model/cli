package cmd

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/opmodel/cli/internal/cmdutil"
	"github.com/opmodel/cli/internal/inventory"
	"github.com/opmodel/cli/internal/kubernetes"
	"github.com/opmodel/cli/internal/output"
)

// NewModApplyCmd creates the mod apply command.
func NewModApplyCmd() *cobra.Command {
	var rf cmdutil.RenderFlags
	var kf cmdutil.K8sFlags

	// Apply-specific flags (local to this command)
	var (
		dryRunFlag     bool
		waitFlag       bool
		timeoutFlag    time.Duration
		createNSFlag   bool
		noPruneFlag    bool
		maxHistoryFlag int
		forceFlag      bool
	)

	cmd := &cobra.Command{
		Use:   "apply [path]",
		Short: "Deploy module to cluster",
		Long: `Deploy an OPM module to a Kubernetes cluster using server-side apply.

This command renders the module and applies the resulting resources to the
cluster. Resources are applied in weight order (CRDs first, webhooks last).

All resources are labeled with OPM metadata for later discovery by
'opm mod delete' and 'opm mod status'.

An inventory Secret is written after each successful apply to record the
exact set of applied resources. On subsequent applies, stale resources
(present in a previous apply but not in the current render) are pruned.

Arguments:
  path    Path to module directory (default: current directory)

Examples:
  # Apply module in current directory
  opm mod apply

  # Apply with custom values and namespace
  opm mod apply ./my-module -f prod-values.cue -n production

  # Preview what would be applied
  opm mod apply --dry-run

  # Apply and wait for resources to be ready
  opm mod apply --wait --timeout 10m

  # Apply without pruning stale resources
  opm mod apply --no-prune

  # Apply with verbose output showing transformer matches
  opm mod apply --verbose`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runApply(cmd, args, &rf, &kf, dryRunFlag, waitFlag, timeoutFlag, createNSFlag, noPruneFlag, maxHistoryFlag, forceFlag)
		},
	}

	rf.AddTo(cmd)
	kf.AddTo(cmd)

	// Apply-specific flags
	cmd.Flags().BoolVar(&dryRunFlag, "dry-run", false,
		"Server-side dry run (no changes made)")
	cmd.Flags().BoolVar(&waitFlag, "wait", false,
		"Wait for resources to be ready")
	cmd.Flags().DurationVar(&timeoutFlag, "timeout", 5*time.Minute,
		"Wait timeout")
	cmd.Flags().BoolVar(&createNSFlag, "create-namespace", false,
		"Create target namespace if it does not exist")
	cmd.Flags().BoolVar(&noPruneFlag, "no-prune", false,
		"Skip stale resource pruning (stale resources remain on cluster)")
	cmd.Flags().IntVar(&maxHistoryFlag, "max-history", 10,
		"Maximum number of change history entries to retain in inventory")
	cmd.Flags().BoolVar(&forceFlag, "force", false,
		"Allow empty render to prune all previously tracked resources")

	return cmd
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
func runApply(_ *cobra.Command, args []string, rf *cmdutil.RenderFlags, kf *cmdutil.K8sFlags, //nolint:gocyclo // orchestration function; complexity is inherent
	dryRun, wait bool, timeout time.Duration, createNS, noProbe bool, maxHistory int, force bool) error {
	ctx := context.Background()

	opmConfig := GetOPMConfig()

	// Step 1: Render module via shared pipeline
	result, err := cmdutil.RenderRelease(ctx, cmdutil.RenderReleaseOpts{
		Args:      args,
		Render:    rf,
		K8s:       kf,
		OPMConfig: opmConfig,
		Registry:  GetRegistry(),
	})
	if err != nil {
		return err
	}

	// Post-render: check errors, show matches, log warnings
	if err := cmdutil.ShowRenderOutput(result, cmdutil.ShowOutputOpts{
		Verbose: verboseFlag,
	}); err != nil {
		return err
	}

	// Create scoped release logger
	releaseLog := output.ReleaseLogger(result.Release.Name)

	// Create Kubernetes client via shared factory
	k8sClient, err := cmdutil.NewK8sClient(kubernetes.ClientOptions{
		Kubeconfig:  kf.Kubeconfig,
		Context:     kf.Context,
		APIWarnings: opmConfig.Config.Log.Kubernetes.APIWarnings,
	})
	if err != nil {
		releaseLog.Error("connecting to cluster", "error", err)
		return err
	}

	namespace := result.Release.Namespace

	// Create namespace if requested
	if createNS && namespace != "" {
		created, nsErr := k8sClient.EnsureNamespace(ctx, namespace, dryRun)
		if nsErr != nil {
			releaseLog.Error("ensuring namespace", "error", nsErr)
			return &ExitError{Code: exitCodeFromK8sError(nsErr), Err: nsErr, Printed: true}
		}
		if created {
			if dryRun {
				releaseLog.Info(fmt.Sprintf("namespace %q would be created", namespace))
			} else {
				releaseLog.Info(fmt.Sprintf("namespace %q created", namespace))
			}
		}
	}

	// Step 2: Compute manifest digest (even for dry-run — useful for idempotency check)
	manifestDigest := inventory.ComputeManifestDigest(result.Resources)
	output.Debug("manifest digest computed", "digest", manifestDigest)

	// Step 3: Compute change ID
	modulePath := ""
	if len(args) > 0 {
		modulePath = args[0]
	}
	// Build a values string from the render options for change ID computation
	valuesStr := strings.Join(rf.Values, ",")
	changeID := inventory.ComputeChangeID(modulePath, result.Release.Version, valuesStr, manifestDigest)
	output.Debug("change ID computed", "changeID", changeID)

	// Step 4: Read previous inventory (nil = first-time apply)
	releaseID := result.Release.ReleaseIdentity
	var prevInventory *inventory.InventorySecret
	if releaseID != "" && !dryRun {
		prevInventory, err = inventory.GetInventory(ctx, k8sClient, result.Release.Name, namespace, releaseID)
		if err != nil {
			releaseLog.Warn("could not read inventory, proceeding without it", "error", err)
		}
	}

	// Extract previous entries for stale detection
	var prevEntries []inventory.InventoryEntry
	if prevInventory != nil && len(prevInventory.Index) > 0 {
		if latestChangeID := prevInventory.Index[0]; latestChangeID != "" {
			if latestChange := prevInventory.Changes[latestChangeID]; latestChange != nil {
				prevEntries = latestChange.Inventory.Entries
			}
		}
	}

	// Build current entries from the rendered resources
	currentEntries := make([]inventory.InventoryEntry, 0, len(result.Resources))
	for _, r := range result.Resources {
		currentEntries = append(currentEntries, inventory.NewEntryFromResource(r))
	}

	// Step 5a: Compute stale set
	staleSet := inventory.ComputeStaleSet(prevEntries, currentEntries)

	// Step 5b: Apply component-rename safety check
	staleSet = inventory.ApplyComponentRenameSafetyCheck(staleSet, currentEntries)

	// Empty render safety gate (before pre-apply check)
	if len(result.Resources) == 0 {
		if len(prevEntries) > 0 && !force {
			return fmt.Errorf("render produced 0 resources but previous inventory has %d entries — "+
				"this would prune all resources; use --force to proceed or --no-prune to skip pruning",
				len(prevEntries))
		}
		if len(prevEntries) == 0 {
			releaseLog.Info("no resources to apply")
			return nil
		}
	}

	// Step 5c: Pre-apply existence check (first-time only, skip for dry-run)
	if prevInventory == nil && !dryRun && !noProbe {
		if err := inventory.PreApplyExistenceCheck(ctx, k8sClient, currentEntries); err != nil {
			return fmt.Errorf("pre-apply existence check failed: %w", err)
		}
	}

	// Step 6: Apply resources via SSA
	if dryRun {
		releaseLog.Info("dry run - no changes will be made")
	}
	if len(result.Resources) > 0 {
		releaseLog.Info(fmt.Sprintf("applying %d resources", len(result.Resources)))
	}

	var applyResult *kubernetes.ApplyResult
	if len(result.Resources) > 0 {
		applyResult, err = kubernetes.Apply(ctx, k8sClient, result.Resources, result.Release, kubernetes.ApplyOptions{
			DryRun: dryRun,
		})
		if err != nil {
			releaseLog.Error("apply failed", "error", err)
			return &ExitError{Code: exitCodeFromK8sError(err), Err: err, Printed: true}
		}

		// Report results
		if len(applyResult.Errors) > 0 {
			releaseLog.Warn(fmt.Sprintf("%d resource(s) had errors", len(applyResult.Errors)))
			for _, e := range applyResult.Errors {
				releaseLog.Error(e.Error())
			}
		}

		if dryRun {
			releaseLog.Info(fmt.Sprintf("dry run complete: %d resources would be applied", applyResult.Applied))
		} else {
			releaseLog.Info(formatApplySummary(applyResult))
		}
	}

	// Steps 7a/7b and 8: only on successful apply, non-dry-run, with release ID
	if !dryRun && releaseID != "" {
		applyHadErrors := applyResult != nil && len(applyResult.Errors) > 0

		if applyHadErrors {
			// Step 7b: skip prune and inventory write on partial failure
			releaseLog.Warn("apply had errors — skipping pruning and inventory write")
			return &ExitError{
				Code:    ExitGeneralError,
				Err:     fmt.Errorf("%d resource(s) failed to apply", len(applyResult.Errors)),
				Printed: true,
			}
		}

		// Step 7a: Prune stale resources (unless --no-prune)
		if len(staleSet) > 0 && !noProbe {
			releaseLog.Info(fmt.Sprintf("pruning %d stale resource(s)", len(staleSet)))
			if err := inventory.PruneStaleResources(ctx, k8sClient, staleSet); err != nil {
				releaseLog.Warn("pruning stale resources failed", "error", err)
				// Non-fatal: inventory still gets written
			}
		}

		// Step 8: Write inventory Secret (skip if nothing changed)
		//
		// If the change ID is already at the head of the index, the manifest,
		// values, and module path are identical to the last apply. Skip the write
		// to avoid unnecessary K8s updates and preserve the original timestamp of
		// when this change was first applied.
		alreadyCurrent := prevInventory != nil &&
			len(prevInventory.Index) > 0 &&
			prevInventory.Index[0] == changeID

		if alreadyCurrent {
			output.Debug("inventory unchanged, skipping write", "changeID", changeID)
		} else {
			newOrUpdatedInventory := prevInventory
			if newOrUpdatedInventory == nil {
				newOrUpdatedInventory = &inventory.InventorySecret{
					Metadata: inventory.InventoryMetadata{
						Kind:             "ModuleRelease",
						APIVersion:       "core.opmodel.dev/v1alpha1",
						ModuleName:       result.Release.ModuleName, // canonical module name, e.g. "minecraft"
						ReleaseName:      result.Release.Name,       // release name, e.g. "mc"
						ReleaseNamespace: namespace,
						ReleaseID:        releaseID,
					},
					Index:   []string{},
					Changes: map[string]*inventory.ChangeEntry{},
				}
			}

			source := inventory.ChangeSource{
				Path:        modulePath,
				Version:     result.Release.Version,
				ReleaseName: result.Release.Name,
				Local:       result.Release.Version == "",
			}

			computedChangeID, changeEntry := inventory.PrepareChange(source, valuesStr, manifestDigest, currentEntries)
			newOrUpdatedInventory.Changes[computedChangeID] = changeEntry
			newOrUpdatedInventory.Index = inventory.UpdateIndex(newOrUpdatedInventory.Index, computedChangeID)
			newOrUpdatedInventory.Metadata.LastTransitionTime = changeEntry.Timestamp
			inventory.PruneHistory(newOrUpdatedInventory, maxHistory)

			if err := inventory.WriteInventory(ctx, k8sClient, newOrUpdatedInventory); err != nil {
				releaseLog.Warn("failed to write inventory Secret", "error", err)
				// Non-fatal: the resources were applied successfully; warn but don't fail
			} else {
				output.Debug("inventory written", "changeID", changeID)
			}
		}
	}

	// Final success message
	if applyResult != nil && len(applyResult.Errors) == 0 && !dryRun {
		if applyResult.Unchanged == applyResult.Applied {
			output.Println(output.FormatCheckmark("Module up to date"))
		} else {
			output.Println(output.FormatCheckmark("Module applied"))
		}
	}

	if applyResult != nil && len(applyResult.Errors) > 0 {
		return &ExitError{
			Code:    ExitGeneralError,
			Err:     fmt.Errorf("%d resource(s) failed to apply", len(applyResult.Errors)),
			Printed: true,
		}
	}

	return nil
}

// formatApplySummary builds a human-readable summary of apply results with
// per-status breakdown (e.g., "applied 5 resources (2 created, 1 configured, 2 unchanged)").
func formatApplySummary(r *kubernetes.ApplyResult) string {
	var parts []string
	if r.Created > 0 {
		parts = append(parts, fmt.Sprintf("%d created", r.Created))
	}
	if r.Configured > 0 {
		parts = append(parts, fmt.Sprintf("%d configured", r.Configured))
	}
	if r.Unchanged > 0 {
		parts = append(parts, fmt.Sprintf("%d unchanged", r.Unchanged))
	}

	summary := fmt.Sprintf("applied %d resources successfully", r.Applied)
	if len(parts) > 0 {
		summary += fmt.Sprintf(" (%s)", strings.Join(parts, ", "))
	}
	return summary
}
