package operatorcmd

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/open-platform-model/library/opm/kernel"

	"github.com/open-platform-model/cli/internal/cmdutil"
	"github.com/open-platform-model/cli/internal/config"
	opmexit "github.com/open-platform-model/cli/internal/exit"
	"github.com/open-platform-model/cli/internal/modref"
	oplib "github.com/open-platform-model/cli/internal/operator"
	"github.com/open-platform-model/cli/internal/output"
	"github.com/open-platform-model/cli/internal/platform"
	"github.com/open-platform-model/cli/internal/publish"
	"github.com/open-platform-model/cli/internal/version"
	workflowrender "github.com/open-platform-model/cli/internal/workflow/render"
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
		valuesFlag            []string
		resetValuesFlag       bool
		timeoutFlag           time.Duration
		catalogPrereleaseFlag bool
		skipPlatformFlag      bool
	)

	c := &cobra.Command{
		Use:   "install",
		Short: "Install the opm-operator on a cluster",
		Long: `Install the opm-operator from its OPM module, opmodel.dev/modules/opm_operator,
as the CLI-owned ModuleInstance opm-operator in opm-operator-system, wait for
it to roll out, and give it a Platform to reconcile.

Install pulls the module from the configured registry (--registry,
OPM_REGISTRY or the config file); an air-gapped cluster installs from a
mirror that serves the module and its dependencies. Without --version it
installs the module version this CLI pins; --version takes a module version
(0.2.0 pins, v0 floats to the newest release of that major), and install
prints the operator release that version deploys. The module renders against
its own dependency pins, never the cluster Platform.

Every check that can refuse runs before anything is written: the values
against the module's #config, an operator newer than this CLI, a rendered
image that disagrees with the operator version the module states, a
ModuleInstance CRD below this CLI's floor, and objects that exist and are
not OPM's. Then the CRDs are applied and served, then the instance is
applied and recorded (every rendered object, the CRDs and the Namespace
included), and install waits for the controller Deployment's rollout.
Objects left terminating by a previous uninstall are waited out first; all
waits share the one --timeout budget.

An operator installed from an earlier release manifest (by an older CLI or
kubectl apply) is migrated: install takes over exactly the objects it
proves that manifest created, recreates the controller Deployment once
(its selector changes, so patches made to it are lost; pass them as
values), and deletes the earlier role bindings the module replaces. An
object that fails the proof refuses the install before anything is
written, and an interrupted migration completes on the next run.

The operator's settings are instance values: -f/--values files are layered
over the values recorded on the operator's instance, so a reinstall keeps
every recorded value it does not change; --reset-values starts from the
module's defaults instead.

A full install then creates the singleton cluster Platform, subscribed to the
newest published release of the first-party catalog. The version is resolved
from the registry before anything is applied. If the catalog has published
no release yet, the command refuses and names --catalog-prerelease; an
existing Platform is reported and left untouched, never rewritten.

--crds-only applies just the CRDs of the same render, for clusters where the
CLI drives module lifecycle without a running operator.

Examples:
  # Install the pinned operator module and seed the cluster Platform
  opm operator install

  # Route the operator's own module fetches through a mirror
  opm operator install -f operator-values.cue   # values: registry: "opmodel.dev=mirror.example/opm"

  # Install another operator module version
  opm operator install --version 0.2.0

  # Start over from the module's defaults
  opm operator install --reset-values

  # Install the operator without creating a Platform
  opm operator install --skip-platform

  # Install only the CRDs, plus RBAC for a specific user
  opm operator install --crds-only --rbac --user alice`,
		Args: cobra.NoArgs,
		RunE: func(c *cobra.Command, _ []string) error {
			return runOperatorInstall(c.Context(), cfg, &kf, installFlags{
				crdsOnly:          crdsOnlyFlag,
				rbac:              rbacFlag,
				user:              userFlag,
				group:             groupFlag,
				version:           versionFlag,
				values:            valuesFlag,
				resetValues:       resetValuesFlag,
				timeout:           timeoutFlag,
				catalogPrerelease: catalogPrereleaseFlag,
				skipPlatform:      skipPlatformFlag,
			})
		},
	}

	kf.AddTo(c)
	c.Flags().BoolVar(&crdsOnlyFlag, "crds-only", false, "Install only the CustomResourceDefinitions of the operator module's render")
	c.Flags().BoolVar(&rbacFlag, "rbac", false, "Also create the opm-cli-user ClusterRole")
	c.Flags().StringVar(&userFlag, "user", "", "Bind the opm-cli-user ClusterRole to this user (requires --rbac)")
	c.Flags().StringVar(&groupFlag, "group", "", "Bind the opm-cli-user ClusterRole to this group (requires --rbac)")
	c.Flags().StringVar(&versionFlag, "version", "", "Operator module version to install: X.Y.Z pins, vN floats (default: the version this CLI pins)")
	c.Flags().StringArrayVarP(&valuesFlag, "values", "f", nil, "Values file layered over the operator instance's recorded values (repeatable)")
	c.Flags().BoolVar(&resetValuesFlag, "reset-values", false, "Ignore the operator instance's recorded values and start from the module's defaults")
	c.Flags().DurationVar(&timeoutFlag, "timeout", defaultOperatorInstallTimeout, "How long to wait, in total, for terminating objects to clear, the CRDs to be served and the operator to roll out")
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
	values            []string
	resetValues       bool
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
// flag, the same rule --user/--group follow against --rbac. -f and
// --reset-values shape the operator's instance, which --crds-only does not
// write.
func (f installFlags) validate() error {
	if f.catalogPrerelease && !f.seedsPlatform() {
		return errors.New("--catalog-prerelease has no effect without Platform seeding; drop --crds-only/--skip-platform or drop --catalog-prerelease")
	}
	if f.crdsOnly && (len(f.values) > 0 || f.resetValues) {
		return errors.New("-f/--values and --reset-values have no effect with --crds-only, which records no operator instance; drop them or drop --crds-only")
	}
	return nil
}

// newModuleRegistry builds the registry client install reads the operator
// module through; tests replace it.
var newModuleRegistry = func(registry string) (oplib.ModuleRegistry, error) {
	src, err := modref.NewSource(registry)
	if err != nil {
		return nil, err
	}
	reg, ok := src.(oplib.ModuleRegistry)
	if !ok {
		return nil, errors.New("the registry client cannot fetch module sources")
	}
	return reg, nil
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

	catalogVersion, res, target, err := resolveBeforeCluster(ctx, cfg, flags)
	if err != nil {
		return err
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

	env := oplib.InstallEnv{
		Client:     k8sClient,
		Render:     moduleRenderer(cfg, k8sConfig),
		CLIVersion: version.Version,
	}
	plan, err := oplib.PlanInstall(ctx, env, res, target, oplib.PlanOptions{
		ValuesFiles: flags.values,
		ResetValues: flags.resetValues,
		CRDsOnly:    flags.crdsOnly,
		Timeout:     flags.timeout,
		Extra:       rbac.Objects(),
	})
	if err != nil {
		return installError(err)
	}

	result, err := oplib.Install(ctx, env, plan)
	if err != nil {
		return installError(withRerunHint(result, err))
	}

	what := "opm-operator " + target.OperatorVersion
	if flags.crdsOnly {
		what = "opm-operator CRDs (" + target.OperatorVersion + ")"
	}
	output.Println(output.FormatCheckmark(fmt.Sprintf("%s installed from module %s", what, target.Module())))

	// Seeding runs after the rollout, so the Platform CRD is Established
	// before the write is attempted.
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

// withRerunHint says a failure after the CRD step is safe to re-run, unless
// the error says so itself (a migration that stopped partway).
func withRerunHint(result *oplib.InstallResult, err error) error {
	var stopped *oplib.MigrationStoppedError
	if result != nil && result.CRDs > 0 && !errors.As(err, &stopped) {
		return fmt.Errorf("%w (install is idempotent, safe to re-run)", err)
	}
	return err
}

// resolveBeforeCluster resolves the catalog version (when the Platform is
// seeded) and the operator module before the cluster is touched: a lookup
// that cannot produce a version must fail with nothing applied.
func resolveBeforeCluster(ctx context.Context, cfg *config.GlobalConfig, flags installFlags) (catalogVersion string, res *modref.Resolution, target oplib.Target, err error) {
	if flags.seedsPlatform() {
		catalogVersion, err = platform.ResolveCatalogVersion(ctx, cfg.Registry, platform.DefaultCatalogPath, flags.catalogPrerelease)
		if err != nil {
			return "", nil, oplib.Target{}, catalogResolveError(err)
		}
	}

	reg, err := newModuleRegistry(cfg.Registry)
	if err != nil {
		return "", nil, oplib.Target{}, &opmexit.ExitError{Code: opmexit.ExitGeneralError, Err: err}
	}
	res, target, err = oplib.ResolveTarget(ctx, reg, modref.Route(cfg.Registry, oplib.OperatorModulePath), flags.version)
	if err != nil {
		return "", nil, oplib.Target{}, installError(err)
	}
	output.Info(fmt.Sprintf("operator module %s %s (%s; deploys opm-operator %s)",
		oplib.OperatorModulePath, target.Module(), selectionLabel(target, flags.version), target.OperatorVersion))
	return catalogVersion, res, target, nil
}

// selectionLabel says how the module version was chosen.
func selectionLabel(t oplib.Target, versionFlag string) string {
	if t.Default {
		return "pinned"
	}
	return "--version " + versionFlag
}

// moduleRenderer renders the operator module as the operator's instance,
// against the module's own dependency pins only.
func moduleRenderer(cfg *config.GlobalConfig, k8sConfig *config.ResolvedKubernetesConfig) oplib.RenderFunc {
	return func(ctx context.Context, res *modref.Resolution, values kernel.Source) (*workflowrender.Result, error) {
		return workflowrender.FromModule(ctx, moduleRenderOpts(cfg, k8sConfig, res, values))
	}
}

// moduleRenderOpts are the render options of the operator's instance: the
// resolved module, the one merged values source, the fixed instance name
// and namespace, and the module's own dependency pins (never a Platform).
func moduleRenderOpts(cfg *config.GlobalConfig, k8sConfig *config.ResolvedKubernetesConfig, res *modref.Resolution, values kernel.Source) workflowrender.ModuleOpts {
	return workflowrender.ModuleOpts{
		Published: res,
		Values:    []kernel.Source{values},
		Name:      oplib.OperatorInstanceName,
		Namespace: oplib.OperatorNamespace,
		DepsOnly:  true,
		K8sConfig: k8sConfig,
		Config:    cfg,
	}
}

// installError maps install failures onto the house funnels: refusals exit
// 2 (a registry refusal printed through the refusal funnel), an unreachable
// registry exits 3 naming the mirror fix, a denied permission exits 4, and
// the rest exit 1 or the code a Kubernetes error carries.
func installError(err error) error {
	var exitErr *opmexit.ExitError
	if errors.As(err, &exitErr) {
		return err
	}
	var refusalErr *modref.RefusalError
	if errors.As(err, &refusalErr) {
		cmdutil.PrintRefusals([]publish.Refusal{refusalErr.Refusal})
		return &opmexit.ExitError{Code: opmexit.ExitValidationError, Err: err, Printed: true}
	}
	var connErr *publish.ConnectivityError
	if errors.As(err, &connErr) {
		return &opmexit.ExitError{Code: opmexit.ExitConnectivityError, Err: fmt.Errorf(
			"%w; install pulls %s from the registry: point --registry or OPM_REGISTRY at a registry or mirror that serves it and its dependencies",
			err, oplib.OperatorModulePath)}
	}
	var (
		ownedErr     *oplib.OwnedRecordError
		guardErr     *oplib.GuardError
		migrationErr *oplib.MigrationRefusalError
	)
	if oplib.IsRefusal(err) || errors.As(err, &ownedErr) || errors.As(err, &guardErr) || errors.As(err, &migrationErr) {
		return &opmexit.ExitError{Code: opmexit.ExitValidationError, Err: err}
	}
	var (
		rolloutErr *oplib.RolloutError
		stoppedErr *oplib.MigrationStoppedError
	)
	if errors.As(err, &rolloutErr) || errors.As(err, &stoppedErr) {
		return &opmexit.ExitError{Code: opmexit.ExitGeneralError, Err: err}
	}
	return &opmexit.ExitError{Code: cmdutil.ExitCodeFromK8sError(err), Err: err}
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
