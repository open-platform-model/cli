package operatorcmd

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/open-platform-model/cli/internal/cmdutil"
	"github.com/open-platform-model/cli/internal/config"
	opmexit "github.com/open-platform-model/cli/internal/exit"
	oplib "github.com/open-platform-model/cli/internal/operator"
	"github.com/open-platform-model/cli/internal/output"
)

const defaultOperatorInstallTimeout = 5 * time.Minute

// NewOperatorInstallCmd creates the operator install command.
func NewOperatorInstallCmd(cfg *config.GlobalConfig) *cobra.Command {
	var kf cmdutil.K8sFlags

	var (
		crdsOnlyFlag bool
		rbacFlag     bool
		userFlag     string
		groupFlag    string
		versionFlag  string
		timeoutFlag  time.Duration
	)

	c := &cobra.Command{
		Use:   "install",
		Short: "Install the opm-operator on a cluster",
		Long: `Server-side-apply the opm-operator onto the current cluster and wait for it
to become ready.

By default this applies the full embedded manifest (CRDs, RBAC, Deployment,
Service) and waits for the CRDs to reach Established and the operator
Deployment to complete its rollout. --crds-only applies just the CRDs, for
clusters where the CLI drives module lifecycle without a running operator.

Examples:
  # Install the full operator
  opm operator install

  # Install only the CRDs
  opm operator install --crds-only

  # Install only the CRDs, plus RBAC for a specific user
  opm operator install --crds-only --rbac --user alice

  # Install a specific opm-operator release instead of the embedded pin
  opm operator install --version v1.0.0-alpha.4`,
		Args: cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return runOperatorInstall(cfg, &kf, installFlags{
				crdsOnly: crdsOnlyFlag,
				rbac:     rbacFlag,
				user:     userFlag,
				group:    groupFlag,
				version:  versionFlag,
				timeout:  timeoutFlag,
			})
		},
	}

	kf.AddTo(c)
	c.Flags().BoolVar(&crdsOnlyFlag, "crds-only", false, "Install only the CustomResourceDefinitions")
	c.Flags().BoolVar(&rbacFlag, "rbac", false, "Also create the opm-cli-user ClusterRole")
	c.Flags().StringVar(&userFlag, "user", "", "Bind the opm-cli-user ClusterRole to this user (requires --rbac)")
	c.Flags().StringVar(&groupFlag, "group", "", "Bind the opm-cli-user ClusterRole to this group (requires --rbac)")
	c.Flags().StringVar(&versionFlag, "version", "", "Fetch this opm-operator release tag instead of the embedded pin")
	c.Flags().DurationVar(&timeoutFlag, "timeout", defaultOperatorInstallTimeout, "How long to wait for the install to become ready")

	return c
}

// installFlags holds the parsed operator install flags.
type installFlags struct {
	crdsOnly bool
	rbac     bool
	user     string
	group    string
	version  string
	timeout  time.Duration
}

func runOperatorInstall(cfg *config.GlobalConfig, kf *cmdutil.K8sFlags, flags installFlags) error {
	rbac := oplib.RBACOptions{Enabled: flags.rbac, User: flags.user, Group: flags.group}
	if err := rbac.Validate(); err != nil {
		return &opmexit.ExitError{Code: opmexit.ExitValidationError, Err: err}
	}

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

	output.Info(fmt.Sprintf("installing opm-operator%s", crdsOnlySuffix(flags.crdsOnly)))

	result, err := oplib.Install(ctx, k8sClient, oplib.InstallOptions{
		CRDsOnly: flags.crdsOnly,
		Version:  flags.version,
		Timeout:  flags.timeout,
		RBAC:     rbac,
	})
	if err != nil {
		if result != nil && result.Applied > 0 {
			err = fmt.Errorf("%w (%d resource(s) applied for %s before this failure — install is idempotent, safe to re-run)", err, result.Applied, result.Version)
		}
		return &opmexit.ExitError{Code: cmdutil.ExitCodeFromK8sError(err), Err: err, Printed: false}
	}

	output.Println(output.FormatCheckmark(fmt.Sprintf(
		"opm-operator %s installed (%s, %d resource(s) applied)", result.Version, result.Source, result.Applied,
	)))
	return nil
}

func crdsOnlySuffix(crdsOnly bool) string {
	if crdsOnly {
		return " (CRDs only)"
	}
	return ""
}
