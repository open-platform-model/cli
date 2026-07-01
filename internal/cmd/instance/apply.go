package instance

import (
	"context"
	"fmt"

	opmexit "github.com/open-platform-model/cli/internal/exit"

	"github.com/spf13/cobra"

	"github.com/open-platform-model/cli/internal/cmdutil"
	"github.com/open-platform-model/cli/internal/config"
	"github.com/open-platform-model/cli/internal/output"
	workflowapply "github.com/open-platform-model/cli/internal/workflow/apply"
	"github.com/open-platform-model/cli/internal/workflow/render"
)

// NewInstanceApplyCmd creates the instance apply command.
func NewInstanceApplyCmd(cfg *config.GlobalConfig) *cobra.Command {
	var rff cmdutil.InstanceFileFlags
	var kf cmdutil.K8sFlags
	var namespace string

	var (
		dryRunFlag   bool
		createNSFlag bool
		noPruneFlag  bool
		forceFlag    bool
	)

	c := &cobra.Command{
		Use:   "apply <instance.cue>",
		Short: "Deploy instance to cluster",
		Long: `Deploy an OPM instance file to a Kubernetes cluster using server-side apply.

Arguments:
  instance.cue    Path to the instance .cue file (required)

Examples:
  # Apply an instance file
  opm instance apply ./jellyfin_instance.cue

  # Dry run
  opm instance apply ./jellyfin_instance.cue --dry-run`,
		Args: cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			return runInstanceApply(args[0], cfg, &rff, &kf, namespace, dryRunFlag, createNSFlag, noPruneFlag, forceFlag)
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

// runInstanceApply executes the instance apply command.
func runInstanceApply(instanceFile string, cfg *config.GlobalConfig, rff *cmdutil.InstanceFileFlags, kf *cmdutil.K8sFlags, namespaceFlag string,
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

	result, err := render.FromInstanceFile(ctx, render.InstanceFileOpts{
		InstanceFilePath: instanceFile,
		ValuesFiles:      rff.Values,
		K8sConfig:        k8sConfig,
		Config:           cfg,
	})
	if err != nil {
		return err
	}

	render.ShowOutput(result, render.ShowOutputOpts{Verbose: cfg.Flags.Verbose})

	instanceLog := output.InstanceLogger(result.Instance.Name)

	k8sClient, err := cmdutil.NewK8sClient(k8sConfig, cfg.Log.Kubernetes.APIWarnings)
	if err != nil {
		instanceLog.Error("connecting to cluster", "error", err)
		return err
	}

	valuesStr := ""

	return workflowapply.Execute(ctx, workflowapply.Request{
		Result:    result,
		K8sClient: k8sClient,
		Log:       instanceLog,
		Options: workflowapply.Options{
			DryRun:                 dryRun,
			CreateNS:               createNS,
			NoPrune:                noPrune,
			Force:                  force,
			SuccessUpToDateMessage: "Instance up to date",
			SuccessAppliedMessage:  "Instance applied",
		},
		Change: workflowapply.ChangeDescriptor{
			Path:      instanceFile,
			ValuesStr: valuesStr,
			Version:   result.Module.Version,
			Local:     result.Module.Version == "",
		},
		ModuleName: result.Module.Name,
		ModuleUUID: result.Module.UUID,
	})
}
