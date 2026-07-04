package operatorcmd

import (
	"context"
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/open-platform-model/cli/internal/cmdutil"
	"github.com/open-platform-model/cli/internal/config"
	opmexit "github.com/open-platform-model/cli/internal/exit"
	oplib "github.com/open-platform-model/cli/internal/operator"
	"github.com/open-platform-model/cli/internal/output"
)

// NewOperatorUninstallCmd creates the operator uninstall command.
func NewOperatorUninstallCmd(cfg *config.GlobalConfig) *cobra.Command {
	var kf cmdutil.K8sFlags
	var removeFinalizersFlag bool

	c := &cobra.Command{
		Use:   "uninstall",
		Short: "Remove the opm-operator from a cluster",
		Long: `Delete everything 'opm operator install' applied, except the CRDs and the
operator's Namespace — those remain for a deliberate, separate 'kubectl delete
crd' once you're sure no ModuleInstance data is still needed.

Refuses to proceed while any ModuleInstance still carries the operator's
cleanup finalizer: deleting the operator out from under it would orphan its
workload without the operator ever running cleanup. --remove-finalizers
overrides this by stripping just that finalizer (leaving any other
finalizers intact) and proceeding — the instance's resources become
unmanaged.

Examples:
  # Remove the operator (refuses if any ModuleInstance is still active)
  opm operator uninstall

  # Remove the operator, orphaning any still-active ModuleInstances
  opm operator uninstall --remove-finalizers`,
		Args: cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return runOperatorUninstall(cfg, &kf, removeFinalizersFlag)
		},
	}

	kf.AddTo(c)
	c.Flags().BoolVar(&removeFinalizersFlag, "remove-finalizers", false,
		"Strip the operator's cleanup finalizer from active ModuleInstances and proceed")

	return c
}

func runOperatorUninstall(cfg *config.GlobalConfig, kf *cmdutil.K8sFlags, removeFinalizers bool) error {
	ctx := context.Background()

	k8sConfig, err := config.ResolveKubernetes(config.ResolveKubernetesOptions{
		Config:         cfg,
		KubeconfigFlag: kf.Kubeconfig,
		ContextFlag:    kf.Context,
	})
	if err != nil {
		return &opmexit.ExitError{Code: opmexit.ExitGeneralError, Err: fmt.Errorf("resolving kubernetes config: %w", err)}
	}
	cmdutil.LogResolvedKubernetesConfig("", k8sConfig.Kubeconfig.Value, k8sConfig.Context.Value)

	k8sClient, err := cmdutil.NewK8sClient(k8sConfig, cfg.Log.Kubernetes.APIWarnings)
	if err != nil {
		return err
	}

	output.Info("uninstalling opm-operator")

	result, err := oplib.Uninstall(ctx, k8sClient, oplib.UninstallOptions{RemoveFinalizers: removeFinalizers})
	if err != nil {
		var guardErr *oplib.FinalizerGuardError
		if errors.As(err, &guardErr) {
			return &opmexit.ExitError{Code: opmexit.ExitValidationError, Err: err}
		}
		return &opmexit.ExitError{Code: cmdutil.ExitCodeFromK8sError(err), Err: err}
	}

	if len(result.Errors) > 0 {
		output.Warn(fmt.Sprintf("%d resource(s) had errors", len(result.Errors)))
		for _, e := range result.Errors {
			output.Error(e.Error())
		}
		return &opmexit.ExitError{
			Code:    opmexit.ExitGeneralError,
			Err:     fmt.Errorf("%d resource(s) failed to delete", len(result.Errors)),
			Printed: true,
		}
	}

	output.Println(output.FormatCheckmark(fmt.Sprintf("opm-operator uninstalled (%d resource(s) deleted)", result.Deleted)))
	return nil
}
