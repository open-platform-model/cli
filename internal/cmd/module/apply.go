package modulecmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/opmodel/cli/internal/cmdutil"
	"github.com/opmodel/cli/internal/config"
	opmexit "github.com/opmodel/cli/internal/exit"
	"github.com/opmodel/cli/internal/output"
	workflowapply "github.com/opmodel/cli/internal/workflow/apply"
	"github.com/opmodel/cli/internal/workflow/render"
)

// NewModuleApplyCmd creates the module apply command.
//
// `opm module apply` deploys a module package directly to a Kubernetes cluster
// via the synthetic `#ModuleRelease` flow. It is the deploy counterpart to
// `opm module build`. After the render stage, behavior is identical to
// `opm instance apply` — inventory, prune, dry-run, and ownership semantics
// all apply to the synthesized release.
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
		Short: "Deploy a module to a cluster via synthetic release",
		Long: `Deploy an OPM module package to a Kubernetes cluster by synthesizing
a #ModuleRelease around it and applying the result. Values come from the
module's debugValues (default) or from -f/--values files.

The synthetic release defaults to "<module>-debug". --name and --namespace
participate in release identity (different values produce different releases,
each with its own inventory Secret). For persistent deploys, author a
release.cue file and use 'opm instance apply' instead.

When switching from 'opm module apply' to 'opm instance apply' (or vice versa)
with a different release name, delete the previous release first to avoid
orphan inventory:

  opm instance delete <module>-debug

Arguments:
  path    Path to a module package directory (default: current directory)

Examples:
  # Apply the current module using debugValues
  opm module apply

  # Apply with a custom synthetic release name
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
	c.Flags().StringVar(&nameFlag, "name", "", "Override synthetic release name")
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
			Err:  fmt.Errorf("module apply expects a directory; CUE packages span all files in a dir. Use 'opm instance apply %s' for a release file", modulePath),
		}
	}

	absModulePath, err := filepath.Abs(modulePath)
	if err != nil {
		return &opmexit.ExitError{Code: opmexit.ExitGeneralError, Err: fmt.Errorf("resolving absolute module path: %w", err)}
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

	releaseLog := output.ReleaseLogger(result.Release.Name)

	k8sClient, err := cmdutil.NewK8sClient(k8sConfig, cfg.Log.Kubernetes.APIWarnings)
	if err != nil {
		releaseLog.Error("connecting to cluster", "error", err)
		return err
	}

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
			Path:      absModulePath,
			ValuesStr: "",
			Version:   result.Module.Version,
			Local:     true,
		},
		ModuleName: result.Module.Name,
		ModuleUUID: result.Module.UUID,
	})
}
