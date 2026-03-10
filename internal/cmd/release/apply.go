package release

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/opmodel/cli/internal/cmdutil"
	"github.com/opmodel/cli/internal/config"
	"github.com/opmodel/cli/internal/inventory"
	"github.com/opmodel/cli/internal/kubernetes"
	"github.com/opmodel/cli/internal/output"
	oerrors "github.com/opmodel/cli/pkg/errors"
)

// NewReleaseApplyCmd creates the release apply command.
func NewReleaseApplyCmd(cfg *config.GlobalConfig) *cobra.Command {
	var rff cmdutil.ReleaseFileFlags
	var kf cmdutil.K8sFlags
	var namespace string

	var (
		dryRunFlag     bool
		createNSFlag   bool
		noPruneFlag    bool
		maxHistoryFlag int
		forceFlag      bool
	)

	c := &cobra.Command{
		Use:   "apply <release.cue>",
		Short: "Deploy release to cluster",
		Long: `Deploy an OPM release file to a Kubernetes cluster using server-side apply.

Arguments:
  release.cue    Path to the release .cue file (required)

Examples:
  # Apply a release file
  opm release apply ./jellyfin_release.cue

  # Apply with a local module
  opm release apply ./jellyfin_release.cue --module ./my-module

  # Dry run
  opm release apply ./jellyfin_release.cue --dry-run`,
		Args: cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			return runReleaseApply(args[0], cfg, &rff, &kf, namespace, dryRunFlag, createNSFlag, noPruneFlag, maxHistoryFlag, forceFlag)
		},
	}

	rff.AddTo(c)
	kf.AddTo(c)
	c.Flags().StringVarP(&namespace, "namespace", "n", "", "Target namespace")
	c.Flags().BoolVar(&dryRunFlag, "dry-run", false, "Server-side dry run (no changes made)")
	c.Flags().BoolVar(&createNSFlag, "create-namespace", false, "Create target namespace if it does not exist")
	c.Flags().BoolVar(&noPruneFlag, "no-prune", false, "Skip stale resource pruning")
	c.Flags().IntVar(&maxHistoryFlag, "max-history", 10, "Maximum number of change history entries to retain")
	c.Flags().BoolVar(&forceFlag, "force", false, "Allow empty render to prune all previously tracked resources")

	return c
}

// runReleaseApply executes the release apply command.
func runReleaseApply(releaseFile string, cfg *config.GlobalConfig, rff *cmdutil.ReleaseFileFlags, kf *cmdutil.K8sFlags, namespaceFlag string, //nolint:gocyclo // orchestration function; complexity is inherent
	dryRun, createNS, noPrune bool, maxHistory int, force bool) error {
	ctx := context.Background()

	k8sConfig, err := config.ResolveKubernetes(config.ResolveKubernetesOptions{
		Config:         cfg,
		KubeconfigFlag: kf.Kubeconfig,
		ContextFlag:    kf.Context,
		NamespaceFlag:  namespaceFlag,
		ProviderFlag:   rff.Provider,
	})
	if err != nil {
		return &oerrors.ExitError{Code: oerrors.ExitGeneralError, Err: fmt.Errorf("resolving kubernetes config: %w", err)}
	}

	result, err := cmdutil.RenderFromReleaseFile(ctx, cmdutil.RenderFromReleaseFileOpts{
		ReleaseFilePath: releaseFile,
		ValuesFiles:     rff.Values,
		ModulePath:      rff.Module,
		K8sConfig:       k8sConfig,
		Config:          cfg,
	})
	if err != nil {
		return err
	}

	cmdutil.ShowRenderOutput(result, cmdutil.ShowOutputOpts{Verbose: cfg.Flags.Verbose})

	releaseLog := output.ReleaseLogger(result.Release.Name)

	k8sClient, err := cmdutil.NewK8sClient(k8sConfig, cfg.Log.Kubernetes.APIWarnings)
	if err != nil {
		releaseLog.Error("connecting to cluster", "error", err)
		return err
	}

	namespace := result.Release.Namespace

	if createNS && namespace != "" {
		created, nsErr := k8sClient.EnsureNamespace(ctx, namespace, dryRun)
		if nsErr != nil {
			releaseLog.Error("ensuring namespace", "error", nsErr)
			return &oerrors.ExitError{Code: cmdutil.ExitCodeFromK8sError(nsErr), Err: nsErr, Printed: true}
		}
		if created {
			if dryRun {
				releaseLog.Info(fmt.Sprintf("namespace %q would be created", namespace))
			} else {
				releaseLog.Info(fmt.Sprintf("namespace %q created", namespace))
			}
		}
	}

	manifestDigest := inventory.ComputeManifestDigest(result.Resources)
	output.Debug("manifest digest computed", "digest", manifestDigest)

	valuesStr := ""
	changeID := inventory.ComputeChangeID(releaseFile, result.Module.Version, valuesStr, manifestDigest)
	output.Debug("change ID computed", "changeID", changeID)

	releaseID := result.Release.UUID
	var prevInventory *inventory.InventorySecret
	if releaseID != "" && !dryRun {
		prevInventory, err = inventory.GetInventory(ctx, k8sClient, result.Release.Name, namespace, releaseID)
		if err != nil {
			releaseLog.Warn("could not read inventory, proceeding without it", "error", err)
		}
	}

	var prevEntries []inventory.InventoryEntry
	if prevInventory != nil && len(prevInventory.Index) > 0 {
		if latestChangeID := prevInventory.Index[0]; latestChangeID != "" {
			if latestChange := prevInventory.Changes[latestChangeID]; latestChange != nil {
				prevEntries = latestChange.Inventory.Entries
			}
		}
	}

	currentEntries := make([]inventory.InventoryEntry, 0, len(result.Resources))
	for _, r := range result.Resources {
		currentEntries = append(currentEntries, inventory.NewEntryFromResource(r))
	}

	staleSet := inventory.ComputeStaleSet(prevEntries, currentEntries)
	staleSet = inventory.ApplyComponentRenameSafetyCheck(staleSet, currentEntries)

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

	if prevInventory == nil && !dryRun {
		if err := inventory.PreApplyExistenceCheck(ctx, k8sClient, currentEntries); err != nil {
			return fmt.Errorf("pre-apply existence check failed: %w", err)
		}
	}

	if dryRun {
		releaseLog.Info("dry run - no changes will be made")
	}
	if len(result.Resources) > 0 {
		releaseLog.Info(fmt.Sprintf("applying %d resources", len(result.Resources)))
	}

	var applyResult *kubernetes.ApplyResult
	if len(result.Resources) > 0 {
		applyResult, err = kubernetes.Apply(ctx, k8sClient, result.Resources, result.Release.Name, kubernetes.ApplyOptions{
			DryRun: dryRun,
		})
		if err != nil {
			releaseLog.Error("apply failed", "error", err)
			return &oerrors.ExitError{Code: cmdutil.ExitCodeFromK8sError(err), Err: err, Printed: true}
		}

		if len(applyResult.Errors) > 0 {
			releaseLog.Warn(fmt.Sprintf("%d resource(s) had errors", len(applyResult.Errors)))
			for _, e := range applyResult.Errors {
				releaseLog.Error(e.Error())
			}
		}

		if dryRun {
			releaseLog.Info(fmt.Sprintf("dry run complete: %d resources would be applied", applyResult.Applied))
		} else {
			releaseLog.Info(cmdutil.FormatApplySummary(applyResult))
		}
	}

	if !dryRun && releaseID != "" {
		applyHadErrors := applyResult != nil && len(applyResult.Errors) > 0
		if applyHadErrors {
			releaseLog.Warn("apply had errors — skipping pruning and inventory write")
			return &oerrors.ExitError{
				Code:    oerrors.ExitGeneralError,
				Err:     fmt.Errorf("%d resource(s) failed to apply", len(applyResult.Errors)),
				Printed: true,
			}
		}

		if len(staleSet) > 0 && !noPrune {
			releaseLog.Info(fmt.Sprintf("pruning %d stale resource(s)", len(staleSet)))
			if err := inventory.PruneStaleResources(ctx, k8sClient, staleSet); err != nil {
				releaseLog.Warn("pruning stale resources failed", "error", err)
			}
		}

		alreadyCurrent := prevInventory != nil &&
			len(prevInventory.Index) > 0 &&
			prevInventory.Index[0] == changeID

		if !alreadyCurrent {
			newOrUpdatedInventory := prevInventory
			if newOrUpdatedInventory == nil {
				newOrUpdatedInventory = &inventory.InventorySecret{
					ReleaseMetadata: inventory.ReleaseMetadata{
						Kind:             "ModuleRelease",
						APIVersion:       "core.opmodel.dev/v1alpha1",
						ReleaseName:      result.Release.Name,
						ReleaseNamespace: namespace,
						ReleaseID:        releaseID,
					},
					Index:   []string{},
					Changes: map[string]*inventory.ChangeEntry{},
				}
			}

			source := inventory.ChangeSource{
				Path:        releaseFile,
				Version:     result.Module.Version,
				ReleaseName: result.Release.Name,
				Local:       result.Module.Version == "",
			}

			computedChangeID, changeEntry := inventory.PrepareChange(source, valuesStr, manifestDigest, currentEntries)
			newOrUpdatedInventory.Changes[computedChangeID] = changeEntry
			newOrUpdatedInventory.Index = inventory.UpdateIndex(newOrUpdatedInventory.Index, computedChangeID)
			newOrUpdatedInventory.ReleaseMetadata.LastTransitionTime = changeEntry.Timestamp
			inventory.PruneHistory(newOrUpdatedInventory, maxHistory)

			if err := inventory.WriteInventory(ctx, k8sClient, newOrUpdatedInventory, result.Module.Name, result.Module.UUID); err != nil {
				releaseLog.Warn("failed to write inventory Secret", "error", err)
			}
		}
	}

	if applyResult != nil && len(applyResult.Errors) == 0 && !dryRun {
		if applyResult.Unchanged == applyResult.Applied {
			output.Println(output.FormatCheckmark("Release up to date"))
		} else {
			output.Println(output.FormatCheckmark("Release applied"))
		}
	}

	if applyResult != nil && len(applyResult.Errors) > 0 {
		return &oerrors.ExitError{
			Code:    oerrors.ExitGeneralError,
			Err:     fmt.Errorf("%d resource(s) failed to apply", len(applyResult.Errors)),
			Printed: true,
		}
	}

	return nil
}
