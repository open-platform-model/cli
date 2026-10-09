package controllercmd

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/open-platform-model/library/opm/kernel"

	"github.com/open-platform-model/cli/internal/cmdutil"
	"github.com/open-platform-model/cli/internal/config"
	ctrllib "github.com/open-platform-model/cli/internal/controller"
	opmexit "github.com/open-platform-model/cli/internal/exit"
	"github.com/open-platform-model/cli/internal/modref"
	"github.com/open-platform-model/cli/internal/output"
	"github.com/open-platform-model/cli/internal/platform"
	"github.com/open-platform-model/cli/internal/publish"
	"github.com/open-platform-model/cli/internal/version"
	workflowrender "github.com/open-platform-model/cli/internal/workflow/render"
)

const defaultControllerInstallTimeout = 5 * time.Minute

// NewControllerInstallCmd creates the controller install command.
func NewControllerInstallCmd(cfg *config.GlobalConfig) *cobra.Command {
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
		Short: "Install the opm-controller on a cluster",
		Long: `Install the opm-controller from its OPM module, opmodel.dev/modules/opm_controller,
as the CLI-owned ModuleInstance opm-controller in opm-system, wait for
it to roll out, and give it a Platform to reconcile.

Install pulls the module from the configured registry (--registry,
OPM_REGISTRY or the config file); an air-gapped cluster installs from a
mirror that serves the module and its dependencies. Without --version it
installs the module version this CLI pins; --version takes a module version
(0.2.0 pins, v0 floats to the newest release of that major), and install
prints the controller release that version deploys. The module renders against
its own dependency pins, never the cluster Platform.

Every check that can refuse runs before anything is written: the values
against the module's #config, a controller newer than this CLI, a rendered
image that disagrees with the controller version the module states, a
ModuleInstance CRD below this CLI's floor, and objects that exist and are
not OPM's. Then the CRDs are applied and served, then the instance is
applied and recorded (every rendered object, the CRDs and the Namespace
included), and install waits for the controller Deployment's rollout.
Objects left terminating by a previous uninstall are waited out first; all
waits share the one --timeout budget.

The controller's settings are instance values: -f/--values files are layered
over the values recorded on the controller's instance, so a reinstall keeps
every recorded value it does not change; --reset-values starts from the
module's defaults instead.

A full install then creates the singleton cluster Platform, subscribed to the
newest published release of the first-party catalog. The version is resolved
from the registry before anything is applied. If the catalog has published
no release yet, the command refuses and names --catalog-prerelease; an
existing Platform is reported and left untouched, never rewritten.

--crds-only applies just the CRDs, for clusters where the CLI drives module
lifecycle without a running controller. They are the CRDs the pinned module
version renders, carried in this CLI, so --crds-only needs no registry and
talks only to the cluster: it resolves and renders no module, writes no
ModuleInstance, and takes no --version, -f or --reset-values. A CRD that
exists is judged as on a full install: one the controller's instance record
lists is applied; one without OPM's managed-by label, or with another
instance's identity, is refused, and the refusal prints the annotation to
set on the CRD so that this instance takes it over; --crds-only adopts nothing
by itself. It also refuses CRDs a newer module version installed: run
--crds-only from a CLI that pins that version or newer, or install that
version in full.

Examples:
  # Install the pinned controller module and seed the cluster Platform
  opm controller install

  # Route the controller's own module fetches through a mirror
  opm controller install -f controller-values.cue   # values: registry: "opmodel.dev=mirror.example/opm"

  # Install another controller module version
  opm controller install --version 0.2.0

  # Start over from the module's defaults
  opm controller install --reset-values

  # Install the controller without creating a Platform
  opm controller install --skip-platform

  # Install only the CRDs this CLI carries (no registry), plus RBAC for a user
  opm controller install --crds-only --rbac --user alice`,
		Args: cobra.NoArgs,
		RunE: func(c *cobra.Command, _ []string) error {
			return runControllerInstall(c.Context(), cfg, &kf, installFlags{
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
	c.Flags().BoolVar(&crdsOnlyFlag, "crds-only", false, "Install only the CustomResourceDefinitions of the pinned controller module, carried in this CLI (no registry)")
	c.Flags().BoolVar(&rbacFlag, "rbac", false, "Also create the opm-cli-user ClusterRole")
	c.Flags().StringVar(&userFlag, "user", "", "Bind the opm-cli-user ClusterRole to this user (requires --rbac)")
	c.Flags().StringVar(&groupFlag, "group", "", "Bind the opm-cli-user ClusterRole to this group (requires --rbac)")
	c.Flags().StringVar(&versionFlag, "version", "", "Controller module version to install: X.Y.Z pins, vN floats (default: the version this CLI pins)")
	c.Flags().StringArrayVarP(&valuesFlag, "values", "f", nil, "Values file layered over the controller instance's recorded values (repeatable)")
	c.Flags().BoolVar(&resetValuesFlag, "reset-values", false, "Ignore the controller instance's recorded values and start from the module's defaults")
	c.Flags().DurationVar(&timeoutFlag, "timeout", defaultControllerInstallTimeout, "How long to wait, in total, for terminating objects to clear, the CRDs to be served and the controller to roll out")
	c.Flags().BoolVar(&catalogPrereleaseFlag, "catalog-prerelease", false, "Subscribe the cluster Platform to the newest catalog prerelease instead of the newest release")
	c.Flags().BoolVar(&skipPlatformFlag, "skip-platform", false, "Do not create the cluster Platform")

	return c
}

// installFlags holds the parsed controller install flags.
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
// --reset-values shape the controller's instance, which --crds-only does not
// write, and --crds-only installs only the CRDs of the pinned version, which
// this CLI carries.
func (f installFlags) validate() error {
	if f.catalogPrerelease && !f.seedsPlatform() {
		return errors.New("--catalog-prerelease has no effect without Platform seeding; drop --crds-only/--skip-platform or drop --catalog-prerelease")
	}
	if f.crdsOnly && (len(f.values) > 0 || f.resetValues) {
		return errors.New("-f/--values and --reset-values have no effect with --crds-only, which records no controller instance; drop them or drop --crds-only")
	}
	if f.crdsOnly && f.version != "" {
		return fmt.Errorf("--crds-only installs the CRDs of controller module %s, the version this CLI pins and carries; drop --version, or drop --crds-only to install module %s in full",
			ctrllib.PinnedModuleVersion, f.version)
	}
	return nil
}

// newModuleRegistry builds the registry client install reads the controller
// module through; tests replace it.
var newModuleRegistry = func(registry string) (ctrllib.ModuleRegistry, error) {
	src, err := modref.NewSource(registry)
	if err != nil {
		return nil, err
	}
	reg, ok := src.(ctrllib.ModuleRegistry)
	if !ok {
		return nil, errors.New("the registry client cannot fetch module sources")
	}
	return reg, nil
}

func runControllerInstall(ctx context.Context, cfg *config.GlobalConfig, kf *cmdutil.K8sFlags, flags installFlags) error {
	rbac := ctrllib.RBACOptions{Enabled: flags.rbac, User: flags.user, Group: flags.group}
	if err := rbac.Validate(); err != nil {
		return &opmexit.ExitError{Code: opmexit.ExitValidationError, Err: err}
	}
	if err := flags.validate(); err != nil {
		return &opmexit.ExitError{Code: opmexit.ExitValidationError, Err: err}
	}

	if ctx == nil {
		ctx = context.Background()
	}

	// --crds-only needs no registry: its CRDs are carried in this CLI.
	var (
		catalogVersion string
		res            *modref.Resolution
		target         = ctrllib.PinnedTarget()
	)
	if !flags.crdsOnly {
		var err error
		catalogVersion, res, target, err = resolveBeforeCluster(ctx, cfg, flags)
		if err != nil {
			return err
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

	env := ctrllib.InstallEnv{
		Client:     k8sClient,
		Render:     moduleRenderer(cfg, k8sConfig),
		CLIVersion: version.Version,
	}
	opts := ctrllib.PlanOptions{
		ValuesFiles: flags.values,
		ResetValues: flags.resetValues,
		Timeout:     flags.timeout,
		Extra:       rbac.Objects(),
	}
	var plan *ctrllib.Plan
	if flags.crdsOnly {
		plan, err = ctrllib.PlanCRDsOnly(ctx, env, opts)
	} else {
		plan, err = ctrllib.PlanInstall(ctx, env, res, target, opts)
	}
	if err != nil {
		return installError(err)
	}

	result, err := ctrllib.Install(ctx, env, plan)
	if err != nil {
		return installError(withRerunHint(result, err))
	}

	if flags.crdsOnly {
		output.Println(output.FormatCheckmark(fmt.Sprintf("opm-controller CRDs (%s) installed from module %s, carried in this CLI",
			target.ControllerVersion, target.Module())))
	} else {
		output.Println(output.FormatCheckmark(fmt.Sprintf("opm-controller %s installed from module %s", target.ControllerVersion, target.Module())))
	}

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

// withRerunHint says a failure after the CRD step is safe to re-run.
func withRerunHint(result *ctrllib.InstallResult, err error) error {
	if result != nil && result.CRDs > 0 {
		return fmt.Errorf("%w (install is idempotent, safe to re-run)", err)
	}
	return err
}

// resolveBeforeCluster resolves the catalog version (when the Platform is
// seeded) and the controller module before the cluster is touched: a lookup
// that cannot produce a version must fail with nothing applied.
func resolveBeforeCluster(ctx context.Context, cfg *config.GlobalConfig, flags installFlags) (catalogVersion string, res *modref.Resolution, target ctrllib.Target, err error) {
	if flags.seedsPlatform() {
		catalogVersion, err = platform.ResolveCatalogVersion(ctx, cfg.Registry, platform.DefaultCatalogPath, flags.catalogPrerelease)
		if err != nil {
			return "", nil, ctrllib.Target{}, catalogResolveError(err)
		}
	}

	reg, err := newModuleRegistry(cfg.Registry)
	if err != nil {
		return "", nil, ctrllib.Target{}, &opmexit.ExitError{Code: opmexit.ExitGeneralError, Err: err}
	}
	res, target, err = ctrllib.ResolveTarget(ctx, reg, modref.Route(cfg.Registry, ctrllib.ControllerModulePath), flags.version)
	if err != nil {
		return "", nil, ctrllib.Target{}, installError(err)
	}
	output.Info(fmt.Sprintf("controller module %s %s (%s; deploys opm-controller %s)",
		ctrllib.ControllerModulePath, target.Module(), selectionLabel(target, flags.version), target.ControllerVersion))
	return catalogVersion, res, target, nil
}

// selectionLabel says how the module version was chosen.
func selectionLabel(t ctrllib.Target, versionFlag string) string {
	if t.Default {
		return "pinned"
	}
	return "--version " + versionFlag
}

// moduleRenderer renders the controller module as the controller's instance,
// against the module's own dependency pins only.
func moduleRenderer(cfg *config.GlobalConfig, k8sConfig *config.ResolvedKubernetesConfig) ctrllib.RenderFunc {
	return func(ctx context.Context, res *modref.Resolution, values kernel.Source) (*workflowrender.Result, error) {
		return workflowrender.FromModule(ctx, ctrllib.ModuleRenderOpts(cfg, k8sConfig, res, values))
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
			err, ctrllib.ControllerModulePath)}
	}
	var (
		ownedErr *ctrllib.OwnedRecordError
		guardErr *ctrllib.GuardError
	)
	if ctrllib.IsRefusal(err) || errors.As(err, &ownedErr) || errors.As(err, &guardErr) {
		return &opmexit.ExitError{Code: opmexit.ExitValidationError, Err: err}
	}
	var rolloutErr *ctrllib.RolloutError
	if errors.As(err, &rolloutErr) {
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
