package modulecmd

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/open-platform-model/cli/internal/cmdutil"
	"github.com/open-platform-model/cli/internal/config"
	opmexit "github.com/open-platform-model/cli/internal/exit"
	"github.com/open-platform-model/cli/internal/output"
	workflowapply "github.com/open-platform-model/cli/internal/workflow/apply"
	"github.com/open-platform-model/cli/internal/workflow/render"
)

// NewModuleApplyCmd creates the module apply command.
//
// `opm module apply` deploys a module package directly to a Kubernetes cluster
// via the synthetic `#ModuleInstance` flow. It is the deploy counterpart to
// `opm module build`. After the render stage, behavior is identical to
// `opm instance apply` — inventory, prune, dry-run, and ownership semantics
// all apply to the synthesized instance.
func NewModuleApplyCmd(cfg *config.GlobalConfig) *cobra.Command {
	var rf cmdutil.RenderFlags
	var kf cmdutil.K8sFlags
	var nameFlag string

	var (
		dryRunFlag   bool
		createNSFlag bool
		noPruneFlag  bool
		forceFlag    bool
	)

	c := &cobra.Command{
		Use:   "apply [path]",
		Short: "Deploy a module to a cluster via synthetic instance",
		Long: `Deploy an OPM module package to a Kubernetes cluster by synthesizing
a #ModuleInstance around it and applying the result. Values come from the
module's debugValues (default) or from -f/--values files.

The synthetic instance defaults to "<module>-debug". --name and --namespace
participate in instance identity (different values produce different instances,
each with its own ModuleInstance CR). The ModuleInstance CRD must be installed
first (run 'opm operator install --crds-only'). For persistent deploys, author
a instance.cue file and use 'opm instance apply' instead.

When switching from 'opm module apply' to 'opm instance apply' (or vice versa)
with a different instance name, delete the previous instance first to avoid
orphan inventory:

  opm instance delete <module>-debug

Arguments:
  path    Path to a module package directory (default: current directory)

Examples:
  # Apply the current module using debugValues
  opm module apply

  # Apply with a custom synthetic instance name
  opm module apply ./my-module --name my-debug

  # Dry run against a specific namespace
  opm module apply ./my-module -n staging --dry-run`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return runModuleApply(args, cfg, &rf, &kf, nameFlag, dryRunFlag, createNSFlag, noPruneFlag, forceFlag)
		},
	}

	rf.AddTo(c)
	kf.AddTo(c)
	c.Flags().StringVar(&nameFlag, "name", "", "Override synthetic instance name")
	c.Flags().BoolVar(&dryRunFlag, "dry-run", false, "Server-side dry run (no changes made)")
	c.Flags().BoolVar(&createNSFlag, "create-namespace", false, "Create target namespace if it does not exist")
	c.Flags().BoolVar(&noPruneFlag, "no-prune", false, "Skip stale resource pruning")
	c.Flags().BoolVar(&forceFlag, "force", false, "Allow empty render to prune all previously tracked resources")

	return c
}

// runModuleApply executes the module apply command.
func runModuleApply(args []string, cfg *config.GlobalConfig, rf *cmdutil.RenderFlags, kf *cmdutil.K8sFlags,
	nameFlag string, dryRun, createNS, noPrune, force bool) error {
	ctx := context.Background()

	modulePath := cmdutil.ResolveModulePath(args)

	info, statErr := os.Stat(modulePath)
	if statErr != nil {
		if os.IsNotExist(statErr) {
			return &opmexit.ExitError{Code: opmexit.ExitGeneralError, Err: fmt.Errorf("module path %q not found", modulePath)}
		}
		return &opmexit.ExitError{Code: opmexit.ExitGeneralError, Err: fmt.Errorf("stat %q: %w", modulePath, statErr)}
	}
	if !info.IsDir() {
		return &opmexit.ExitError{
			Code: opmexit.ExitGeneralError,
			Err:  fmt.Errorf("module apply expects a directory; CUE packages span all files in a dir. Use 'opm instance apply %s' for a instance file", modulePath),
		}
	}

	k8sConfig, err := config.ResolveKubernetes(config.ResolveKubernetesOptions{
		Config:         cfg,
		KubeconfigFlag: kf.Kubeconfig,
		ContextFlag:    kf.Context,
		NamespaceFlag:  rf.Namespace,
		ProviderFlag:   rf.Provider,
	})
	if err != nil {
		return &opmexit.ExitError{Code: opmexit.ExitGeneralError, Err: fmt.Errorf("resolving kubernetes config: %w", err)}
	}

	result, err := render.FromModule(ctx, render.ModuleOpts{
		ModulePath:  modulePath,
		ValuesFiles: rf.Values,
		Name:        nameFlag,
		K8sConfig:   k8sConfig,
		Config:      cfg,
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
	})
}
