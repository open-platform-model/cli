package instance

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/open-platform-model/cli/internal/cmdutil"
	"github.com/open-platform-model/cli/internal/config"
	opmexit "github.com/open-platform-model/cli/internal/exit"
	"github.com/open-platform-model/cli/internal/output"
	"github.com/open-platform-model/cli/internal/workflow/handoff"
)

// defaultHandoffTimeout bounds the post-flip reconcile wait, matching the
// operator install waits.
const defaultHandoffTimeout = 5 * time.Minute

// NewInstanceHandoffCmd creates the instance handoff command.
func NewInstanceHandoffCmd(cfg *config.GlobalConfig) *cobra.Command {
	var kf cmdutil.K8sFlags
	var namespace string
	var platformFlag string

	var (
		timeoutFlag time.Duration
		forceFlag   bool
	)

	c := &cobra.Command{
		Use:   "handoff <name>",
		Short: "Transfer an instance from CLI to operator management",
		Long: `Transfer a CLI-managed instance to the opm operator.

Handoff verifies that the operator can take over safely before changing
anything, then flips spec.owner to "operator" and waits for the operator's
first reconcile.

Before the flip it checks, in order:
  1. the operator is installed and ready
  2. the ModuleInstance exists and the CLI owns it
  3. the instance was not last applied from local module bytes
  4. spec.module resolves from the registry (ignoring any local replacement
     and any cached copy — the operator gets no such shortcuts)
  5. re-rendering the published module reproduces what is deployed

A successful handoff relabels the instance's resources to the operator's
managed-by identity. No workload is restarted, created, or removed.

Handoff is forward-only: there is no reverse mode, and a failure after the
flip leaves the instance with the operator rather than silently undoing it.

Arguments:
  name    Instance name (use -n / --namespace to scope by namespace)

Examples:
  # Hand off an instance to the operator
  opm instance handoff jellyfin -n media

  # Hand off despite a verification digest mismatch
  opm instance handoff jellyfin -n media --force`,
		Args: cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			if platformFlag != "" {
				return &opmexit.ExitError{Code: opmexit.ExitValidationError, Err: fmt.Errorf(
					"handoff does not accept --platform: it verifies against the cluster Platform, because that is what the operator will render against")}
			}
			return runInstanceHandoff(args[0], cfg, &kf, namespace, timeoutFlag, forceFlag)
		},
	}

	kf.AddTo(c)
	c.Flags().StringVarP(&namespace, "namespace", "n", "", "Target namespace")
	c.Flags().DurationVar(&timeoutFlag, "timeout", defaultHandoffTimeout, "Bound on the post-flip reconcile wait")
	c.Flags().BoolVar(&forceFlag, "force", false, "Proceed even when the verification render does not match the deployed state")

	// Declared, and left visible, so the rejection is a stated rule rather than
	// cobra's generic "unknown flag". The platform source is a decision here
	// (0006 D11), not an omission, and --help is where a user looks to find
	// that out.
	c.Flags().StringVar(&platformFlag, "platform", "",
		"Not supported: handoff always verifies against the cluster Platform")

	return c
}

func runInstanceHandoff(name string, cfg *config.GlobalConfig, kf *cmdutil.K8sFlags, namespaceFlag string, timeout time.Duration, force bool) error {
	ctx := context.Background()

	k8sConfig, err := config.ResolveKubernetes(config.ResolveKubernetesOptions{
		Config:         cfg,
		KubeconfigFlag: kf.Kubeconfig,
		ContextFlag:    kf.Context,
		NamespaceFlag:  namespaceFlag,
	})
	if err != nil {
		return &opmexit.ExitError{Code: opmexit.ExitGeneralError, Err: fmt.Errorf("resolving kubernetes config: %w", err)}
	}
	cmdutil.LogResolvedKubernetesConfig(k8sConfig.Namespace.Value, k8sConfig.Kubeconfig.Value, k8sConfig.Context.Value)

	k8sClient, err := cmdutil.NewK8sClient(k8sConfig, cfg.Log.Kubernetes.APIWarnings)
	if err != nil {
		output.Error("connecting to cluster", "error", err)
		return err
	}

	return handoff.Execute(ctx, handoff.Request{
		Name:      name,
		Namespace: k8sConfig.Namespace.Value,
		K8sClient: k8sClient,
		Config:    cfg,
		Log:       output.InstanceLogger(name),
		Timeout:   timeout,
		Force:     force,
	})
}
