package release

import (
	"context"
	"fmt"

	opmexit "github.com/opmodel/cli/internal/exit"

	"github.com/spf13/cobra"

	"github.com/opmodel/cli/internal/cmdutil"
	"github.com/opmodel/cli/internal/config"
	"github.com/opmodel/cli/internal/output"
	workflowapply "github.com/opmodel/cli/internal/workflow/apply"
	"github.com/opmodel/cli/internal/workflow/render"
)

// NewReleaseApplyCmd creates the release apply command.
func NewReleaseApplyCmd(cfg *config.GlobalConfig) *cobra.Command {
	var rff cmdutil.ReleaseFileFlags
	var kf cmdutil.K8sFlags
	var namespace string

	var (
		dryRunFlag   bool
		createNSFlag bool
		noPruneFlag  bool
		forceFlag    bool
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

  # Dry run
  opm release apply ./jellyfin_release.cue --dry-run`,
		Args: cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			return runReleaseApply(args[0], cfg, &rff, &kf, namespace, dryRunFlag, createNSFlag, noPruneFlag, forceFlag)
		},
	}

	rff.AddTo(c)
	kf.AddTo(c)
	c.Flags().StringVarP(&namespace, "namespace", "n", "", "Target namespace")
	c.Flags().BoolVar(&dryRunFlag, "dry-run", false, "Server-side dry run (no changes made)")
	c.Flags().BoolVar(&createNSFlag, "create-namespace", false, "Create target namespace if it does not exist")
	c.Flags().BoolVar(&noPruneFlag, "no-prune", false, "Skip stale resource pruning")
	c.Flags().BoolVar(&forceFlag, "force", false, "Allow empty render to prune all previously tracked resources")

	return c
}

// runReleaseApply executes the release apply command.
func runReleaseApply(releaseFile string, cfg *config.GlobalConfig, rff *cmdutil.ReleaseFileFlags, kf *cmdutil.K8sFlags, namespaceFlag string,
	dryRun, createNS, noPrune, force bool) error {
	ctx := context.Background()

	k8sConfig, err := config.ResolveKubernetes(config.ResolveKubernetesOptions{
		Config:         cfg,
		KubeconfigFlag: kf.Kubeconfig,
		ContextFlag:    kf.Context,
		NamespaceFlag:  namespaceFlag,
		ProviderFlag:   rff.Provider,
	})
	if err != nil {
		return &opmexit.ExitError{Code: opmexit.ExitGeneralError, Err: fmt.Errorf("resolving kubernetes config: %w", err)}
	}

	result, err := render.FromReleaseFile(ctx, render.ReleaseFileOpts{
		ReleaseFilePath: releaseFile,
		ValuesFiles:     rff.Values,
		K8sConfig:       k8sConfig,
		Config:          cfg,
	})
	if err != nil {
		return err
	}

	render.ShowOutput(result, render.ShowOutputOpts{Verbose: cfg.Flags.Verbose})

	releaseLog := output.ReleaseLogger(result.Release.Name)

	k8sClient, err := cmdutil.NewK8sClient(k8sConfig, cfg.Log.Kubernetes.APIWarnings)
	if err != nil {
		releaseLog.Error("connecting to cluster", "error", err)
		return err
	}

	valuesStr := ""

	return workflowapply.Execute(ctx, workflowapply.Request{
		Result:    result,
		K8sClient: k8sClient,
		Log:       releaseLog,
		Options: workflowapply.Options{
			DryRun:                 dryRun,
			CreateNS:               createNS,
			NoPrune:                noPrune,
			Force:                  force,
			SuccessUpToDateMessage: "Release up to date",
			SuccessAppliedMessage:  "Release applied",
		},
		Change: workflowapply.ChangeDescriptor{
			Path:      releaseFile,
			ValuesStr: valuesStr,
			Version:   result.Module.Version,
			Local:     result.Module.Version == "",
		},
		ModuleName: result.Module.Name,
		ModuleUUID: result.Module.UUID,
	})
}
