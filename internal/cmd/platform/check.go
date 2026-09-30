package platformcmd

import (
	"context"
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/open-platform-model/cli/internal/cmdutil"
	"github.com/open-platform-model/cli/internal/config"
	opmexit "github.com/open-platform-model/cli/internal/exit"
	"github.com/open-platform-model/cli/internal/output"
	"github.com/open-platform-model/cli/internal/platform"
)

// NewPlatformCheckCmd creates the platform check command.
func NewPlatformCheckCmd(cfg *config.GlobalConfig) *cobra.Command {
	var platformFlag string
	var kf cmdutil.K8sFlags

	c := &cobra.Command{
		Use:   "check [dir]",
		Short: "Report a platform's contract inventory",
		Long: `Report what a platform's enabled catalogs contract for.

Builds the resolved platform module and reports the contract inventory core
derives from it: every contract the enabled catalogs define and the catalog
that defines each, the transformers that implement it, the provider-fulfilled
contracts nothing implements, the provider-fulfilled contracts required by
transformers of more than one enabled registry entry, with the registry keys
providing each (two majors of one catalog are two entries), and every pair of
transformers whose match predicates are comparable over a shared
catalog-fulfilled contract.

The command applies and renders nothing. It contacts a cluster only to read
its Platform, when neither [dir] nor --platform is given. A cold module cache
still fetches the platform's pinned core and catalogs.

The exit code carries what platform-package generation refuses on, not the
severity of the word:

  over-subscribed   exits with the validation error code, because a platform
                    package cannot be generated from an over-subscribed
                    platform
  comparable        exits with the validation error code — every component
                    the narrower transformer matches is also matched by the
                    broader one, so both would render and nothing tells them
                    apart (0015:D5)
  unfulfilled       exits 0 — a platform may define a contract ahead of the
                    provider that implements it, and an unmet demand is
                    refused by the render that demands it

The platform is resolved as [dir] > --platform > the cluster's Platform,
read through --kubeconfig/--context. With none of the three the command
refuses; it never checks a platform from the OPM home directory.

Examples:
  # Check the platform of the current kubeconfig context's cluster
  opm platform check

  # Check a platform module in a repository
  opm platform check ./platforms/staging`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			return runPlatformCheck(c.Context(), args, cfg, platformFlag, kf)
		},
	}

	c.Flags().StringVar(&platformFlag, "platform", "",
		"Platform module directory")
	kf.AddTo(c)

	return c
}

// runPlatformCheck resolves the platform, builds it, and prints the contract
// inventory report. Routability and discrimination decide the exit status —
// the two conditions platform-package generation refuses on (0015:D5,
// 0010:D37); an unfulfilled contract never does (0015:D18).
func runPlatformCheck(ctx context.Context, args []string, cfg *config.GlobalConfig, platformFlag string, kf cmdutil.K8sFlags) error {
	argDir := ""
	if len(args) > 0 {
		argDir = args[0]
	}

	// The cluster is read only when neither a directory argument nor
	// --platform names the platform.
	var cluster platform.ClusterPlatformGetter
	if argDir == "" && platformFlag == "" {
		getter, err := checkClusterGetter(cfg, kf)
		if err != nil {
			return err
		}
		cluster = getter
	}

	dir, res, err := platform.Resolve(ctx, platform.ResolveOptions{
		Argument:     argDir,
		PlatformFlag: platformFlag,
		ConfigPath:   cfg.ConfigPath,
		Cluster:      cluster,
		NoFallback:   true,
		Registry:     cfg.Registry,
	})
	if err != nil {
		return checkResolveError(err)
	}

	p, err := config.BuildPlatformModule(ctx, dir, cfg.Registry)
	if err != nil {
		cmdutil.PrintValidationError("platform module does not build", err)
		return &opmexit.ExitError{Code: opmexit.ExitValidationError, Err: err, Printed: true}
	}

	inv, err := p.Contracts()
	if err != nil {
		return &opmexit.ExitError{
			Code: opmexit.ExitValidationError,
			Err:  fmt.Errorf("reading the contract inventory of the platform at %s: %w", dir, err),
		}
	}

	report := platform.NewReport(res, inv)
	output.Println(report.Render())
	// Both counts are named, whichever refusal fired: the two have
	// different fixes (disable a competing catalog; discriminate the
	// predicates), so a single combined verdict would hide which one is
	// being reported.
	if !report.Routable() || !report.Discriminated() {
		return &opmexit.ExitError{
			Code: opmexit.ExitValidationError,
			Err: fmt.Errorf("platform %s cannot generate a platform package: %d over-subscribed contract(s), %d comparable transformer pair(s)",
				dir, len(report.OverSubscribed), len(report.Comparable)),
			Printed: true,
		}
	}
	return nil
}

// noPlatformToCheck is the refusal when none of the three sources yields a
// platform.
const noPlatformToCheck = "no platform to check: pass [dir] or --platform <dir>, or point --context at a cluster with a Platform"

// checkClusterGetter builds the cluster Platform getter for the cluster
// step. A kubeconfig with no context yields no getter (the resolver then
// refuses naming all three sources); a kubeconfig that cannot be used is a
// connectivity failure.
func checkClusterGetter(cfg *config.GlobalConfig, kf cmdutil.K8sFlags) (platform.ClusterPlatformGetter, error) {
	k8sConfig, err := config.ResolveKubernetes(config.ResolveKubernetesOptions{
		Config:         cfg,
		KubeconfigFlag: kf.Kubeconfig,
		ContextFlag:    kf.Context,
	})
	if err != nil {
		return nil, &opmexit.ExitError{Code: opmexit.ExitGeneralError, Err: fmt.Errorf("resolving kubernetes config: %w", err)}
	}
	client, err := cmdutil.NewK8sClient(k8sConfig, cfg.Log.Kubernetes.APIWarnings)
	if err != nil {
		if cmdutil.IsNoKubeContext(err) {
			return nil, nil
		}
		return nil, &opmexit.ExitError{Code: opmexit.ExitConnectivityError, Err: fmt.Errorf("connecting to cluster: %w", err)}
	}
	return platform.ClusterPlatformGetterFor(client.Dynamic), nil
}

// checkResolveError maps a resolution failure onto the command's exit codes:
// no source at all, or no readable cluster Platform, is not-found naming the
// three sources; an unreachable cluster is a connectivity failure; anything
// else (a directory that is not a platform module) is not-found as before.
func checkResolveError(err error) error {
	switch {
	case errors.Is(err, platform.ErrNoPlatformSource), errors.Is(err, platform.ErrNoClusterPlatform):
		return &opmexit.ExitError{Code: opmexit.ExitNotFound, Err: fmt.Errorf("%s (%w)", noPlatformToCheck, err)}
	case errors.Is(err, platform.ErrClusterRead):
		return &opmexit.ExitError{Code: opmexit.ExitConnectivityError, Err: err}
	default:
		return &opmexit.ExitError{Code: opmexit.ExitNotFound, Err: err}
	}
}
