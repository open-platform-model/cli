package controllercmd

import (
	"context"
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/open-platform-model/cli/internal/cmdutil"
	"github.com/open-platform-model/cli/internal/config"
	ctrllib "github.com/open-platform-model/cli/internal/controller"
	opmexit "github.com/open-platform-model/cli/internal/exit"
	"github.com/open-platform-model/cli/internal/output"
)

// NewControllerUninstallCmd creates the controller uninstall command.
func NewControllerUninstallCmd(cfg *config.GlobalConfig) *cobra.Command {
	var kf cmdutil.K8sFlags
	var removeFinalizersFlag bool

	c := &cobra.Command{
		Use:   "uninstall",
		Short: "Remove the opm-controller from a cluster",
		Long: `Delete every object the controller's instance record (ModuleInstance
opm-controller in opm-controller-system) lists, then the record, except the CRDs
and the controller's Namespace — those remain for a deliberate, separate
'kubectl delete crd' once you're sure no ModuleInstance data is still needed.
An object that no longer carries the instance's identity is left behind.

With no record, uninstall deletes nothing: run 'opm controller install' first,
so the running controller is recorded.

Refuses to proceed while any ModuleInstance still carries the controller's
cleanup finalizer: deleting the controller out from under it would orphan its
workload without the controller ever running cleanup. --remove-finalizers
overrides this by stripping just that finalizer (leaving any other
finalizers intact) and proceeding — the instance's resources become
unmanaged.

Examples:
  # Remove the controller (refuses if any ModuleInstance is still active)
  opm controller uninstall

  # Remove the controller, orphaning any still-active ModuleInstances
  opm controller uninstall --remove-finalizers`,
		Args: cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return runControllerUninstall(cfg, &kf, removeFinalizersFlag)
		},
	}

	kf.AddTo(c)
	c.Flags().BoolVar(&removeFinalizersFlag, "remove-finalizers", false,
		"Strip the controller's cleanup finalizer from active ModuleInstances and proceed")

	return c
}

func runControllerUninstall(cfg *config.GlobalConfig, kf *cmdutil.K8sFlags, removeFinalizers bool) error {
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

	output.Info("uninstalling opm-controller")

	result, err := ctrllib.Uninstall(ctx, k8sClient, ctrllib.UninstallOptions{RemoveFinalizers: removeFinalizers})
	if err != nil {
		var guardErr *ctrllib.FinalizerGuardError
		var noRecord *ctrllib.NoRecordError
		if errors.As(err, &guardErr) || errors.As(err, &noRecord) {
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

	output.Println(output.FormatCheckmark(fmt.Sprintf("opm-controller uninstalled (%d resource(s) deleted, %d left behind)", result.Deleted, result.LeftBehind)))
	return nil
}
