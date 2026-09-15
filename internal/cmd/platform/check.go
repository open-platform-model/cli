package platformcmd

import (
	"context"
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

	c := &cobra.Command{
		Use:   "check [dir]",
		Short: "Report a platform's contract inventory",
		Long: `Report what a platform's enabled catalogs contract for.

Builds the resolved platform module and reports the contract inventory core
derives from it: every contract the enabled catalogs define and the catalog
that defines each, the transformers that implement it, the provider-fulfilled
contracts nothing implements, and the provider-fulfilled contracts required by
transformers from more than one catalog.

Offline: the command applies nothing, renders nothing and contacts no cluster.
A cold module cache still fetches the platform's pinned core and catalogs.

The exit code carries the routability verdict, not the severity of the word:

  over-subscribed   exits with the validation error code, because a platform
                    package cannot be generated from an over-subscribed
                    platform
  unfulfilled       exits 0 — a platform may define a contract ahead of the
                    provider that implements it, and an unmet demand is
                    refused by the render that demands it

The platform is resolved by the usual precedence, with a directory argument
above all of it: [dir] > --platform > ~/.opm/platform/. The cluster Platform
CR is never read here; this command checks a platform module.

Examples:
  # Check the configured default platform
  opm platform check

  # Check a platform module in a repository
  opm platform check ./platforms/staging`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			return runPlatformCheck(c.Context(), args, cfg, platformFlag)
		},
	}

	c.Flags().StringVar(&platformFlag, "platform", "",
		"Path to a platform module directory (overrides ~/.opm/platform/)")

	return c
}

// runPlatformCheck resolves the platform, builds it, and prints the contract
// inventory report. Only routability decides the exit status (enhancement
// 0015 D18).
func runPlatformCheck(ctx context.Context, args []string, cfg *config.GlobalConfig, platformFlag string) error {
	argDir := ""
	if len(args) > 0 {
		argDir = args[0]
	}

	// Cluster is deliberately nil: a cluster's effective registry is its own
	// subject (0015 D6), and this command reads a platform module.
	dir, res, err := platform.Resolve(ctx, platform.ResolveOptions{
		Argument:     argDir,
		PlatformFlag: platformFlag,
		ConfigPath:   cfg.ConfigPath,
		Registry:     cfg.Registry,
	})
	if err != nil {
		return &opmexit.ExitError{Code: opmexit.ExitNotFound, Err: err}
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
	if !report.Routable() {
		return &opmexit.ExitError{
			Code: opmexit.ExitValidationError,
			Err: fmt.Errorf("platform %s is not routable: %d over-subscribed contract(s) — a platform package cannot be generated from it",
				dir, len(report.OverSubscribed)),
			Printed: true,
		}
	}
	return nil
}
