package operatorcmd

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/open-platform-model/cli/internal/cmdutil"
	"github.com/open-platform-model/cli/internal/config"
	opmexit "github.com/open-platform-model/cli/internal/exit"
	oplib "github.com/open-platform-model/cli/internal/operator"
	"github.com/open-platform-model/cli/internal/output"
	"github.com/open-platform-model/cli/internal/platform"
	"github.com/open-platform-model/cli/internal/publish"
)

const defaultOperatorInstallTimeout = 5 * time.Minute

// NewOperatorInstallCmd creates the operator install command.
func NewOperatorInstallCmd(cfg *config.GlobalConfig) *cobra.Command {
	var kf cmdutil.K8sFlags

	var (
		crdsOnlyFlag          bool
		rbacFlag              bool
		userFlag              string
		groupFlag             string
		versionFlag           string
		timeoutFlag           time.Duration
		catalogPrereleaseFlag bool
		skipPlatformFlag      bool
	)

	c := &cobra.Command{
		Use:   "install",
		Short: "Install the opm-operator on a cluster",
		Long: `Server-side-apply the opm-operator onto the current cluster, wait for it to
become ready, and give it a Platform to reconcile.

By default this applies the full embedded manifest (CRDs, RBAC, Deployment,
Service) and waits for the CRDs to reach Established and the operator
Deployment to complete its rollout. --crds-only applies just the CRDs, for
clusters where the CLI drives module lifecycle without a running operator.

A full install then creates the singleton cluster Platform, subscribed to the
newest published release of the first-party catalog. The version is resolved
from the registry before anything is applied, so a registry problem never
leaves a half-installed cluster. If the catalog has published no release yet,
the command refuses and names --catalog-prerelease; an existing Platform is
reported and left untouched, never rewritten.

Examples:
  # Install the full operator and seed the cluster Platform
  opm operator install

  # Subscribe the Platform to the newest catalog prerelease instead
  opm operator install --catalog-prerelease

  # Install the operator without creating a Platform
  opm operator install --skip-platform

  # Install only the CRDs
  opm operator install --crds-only

  # Install only the CRDs, plus RBAC for a specific user
  opm operator install --crds-only --rbac --user alice

  # Install a specific opm-operator release instead of the embedded pin
  opm operator install --version v1.0.0-alpha.4`,
		Args: cobra.NoArgs,
		RunE: func(c *cobra.Command, _ []string) error {
			return runOperatorInstall(c.Context(), cfg, &kf, installFlags{
				crdsOnly:          crdsOnlyFlag,
				rbac:              rbacFlag,
				user:              userFlag,
				group:             groupFlag,
				version:           versionFlag,
				timeout:           timeoutFlag,
				catalogPrerelease: catalogPrereleaseFlag,
				skipPlatform:      skipPlatformFlag,
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
	c.Flags().BoolVar(&catalogPrereleaseFlag, "catalog-prerelease", false, "Subscribe the cluster Platform to the newest catalog prerelease instead of the newest release")
	c.Flags().BoolVar(&skipPlatformFlag, "skip-platform", false, "Do not create the cluster Platform")

	return c
}

// installFlags holds the parsed operator install flags.
type installFlags struct {
	crdsOnly          bool
	rbac              bool
	user              string
	group             string
	version           string
	timeout           time.Duration
	catalogPrerelease bool
	skipPlatform      bool
}

// seedsPlatform reports whether this invocation will create the cluster
// Platform, and therefore whether it needs a catalog version at all.
func (f installFlags) seedsPlatform() bool {
	return !f.crdsOnly && !f.skipPlatform
}

// validate rejects flag combinations before any registry or cluster call.
// --catalog-prerelease selects a version for the Platform, so pairing it with
// a mode that creates no Platform is an error rather than a silently inert
// flag, the same rule --user/--group follow against --rbac.
func (f installFlags) validate() error {
	if f.catalogPrerelease && !f.seedsPlatform() {
		return errors.New("--catalog-prerelease has no effect without Platform seeding; drop --crds-only/--skip-platform or drop --catalog-prerelease")
	}
	return nil
}

func runOperatorInstall(ctx context.Context, cfg *config.GlobalConfig, kf *cmdutil.K8sFlags, flags installFlags) error {
	rbac := oplib.RBACOptions{Enabled: flags.rbac, User: flags.user, Group: flags.group}
	if err := rbac.Validate(); err != nil {
		return &opmexit.ExitError{Code: opmexit.ExitValidationError, Err: err}
	}
	if err := flags.validate(); err != nil {
		return &opmexit.ExitError{Code: opmexit.ExitValidationError, Err: err}
	}

	if ctx == nil {
		ctx = context.Background()
	}

	// Resolve the catalog version before the cluster is touched: a lookup
	// that cannot produce a version must fail with nothing applied.
	var catalogVersion string
	if flags.seedsPlatform() {
		var err error
		catalogVersion, err = platform.ResolveCatalogVersion(ctx, cfg.Registry, platform.DefaultCatalogPath, flags.catalogPrerelease)
		if err != nil {
			return catalogResolveError(err)
		}
	}

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

	// Seeding runs after the readiness wait, so the Platform CRD is
	// Established before the write is attempted.
	switch {
	case flags.seedsPlatform():
		if err := platform.EnsureClusterPlatformForCatalog(ctx, k8sClient.Dynamic, platform.DefaultCatalogPath, catalogVersion); err != nil {
			return &opmexit.ExitError{Code: cmdutil.ExitCodeFromK8sError(err), Err: err}
		}
	case flags.skipPlatform:
		output.Info("cluster Platform not created (--skip-platform)")
	}

	return nil
}

// catalogResolveError maps catalog resolution failures onto the house
// funnels: a refusal prints and exits 2, an unreachable registry exits 3
// (nothing was ever judged), anything else exits 1.
func catalogResolveError(err error) error {
	var refusalErr *platform.RefusalError
	if errors.As(err, &refusalErr) {
		cmdutil.PrintRefusals([]publish.Refusal{refusalErr.Refusal})
		return &opmexit.ExitError{
			Code:    opmexit.ExitValidationError,
			Err:     errors.New(refusalErr.Refusal.Headline),
			Printed: true,
		}
	}
	var connErr *publish.ConnectivityError
	if errors.As(err, &connErr) {
		return &opmexit.ExitError{Code: opmexit.ExitConnectivityError, Err: err}
	}
	return &opmexit.ExitError{Code: opmexit.ExitGeneralError, Err: err}
}

func crdsOnlySuffix(crdsOnly bool) string {
	if crdsOnly {
		return " (CRDs only)"
	}
	return ""
}
