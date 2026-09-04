package catalogcmd

import (
	"errors"
	"fmt"

	"cuelang.org/go/cue/cuecontext"
	"github.com/spf13/cobra"

	"github.com/open-platform-model/cli/internal/cmdutil"
	"github.com/open-platform-model/cli/internal/config"
	opmexit "github.com/open-platform-model/cli/internal/exit"
	"github.com/open-platform-model/cli/internal/publish"
)

// AidSentence is 0010 D35's framing, carried in the check command's help text
// verbatim (graduation-gated): publish-side enforcement is the whole of the
// guarantee, and this command is the out-of-band aid beside it.
const AidSentence = "This check is an aid, not a guarantee: nothing requires it to have been run, and enforcement exists only at publish — it does not make an unchecked catalog trustworthy; it makes it checkable."

// NewCatalogRegistryCmd creates the catalog registry command group — the
// commands that start from a published catalog rather than from source.
func NewCatalogRegistryCmd(cfg *config.GlobalConfig) *cobra.Command {
	c := &cobra.Command{
		Use:   "registry",
		Short: "Work with published catalogs",
		Long: `Work with catalogs already published to a registry.

	Use this command group when you are starting from a published artifact:
	verify a build out of band, after the fact, from the consumer's side.`,
	}

	c.AddCommand(NewCatalogRegistryCheckCmd(cfg))

	return c
}

// NewCatalogRegistryCheckCmd creates the catalog registry check command (D7).
func NewCatalogRegistryCheckCmd(cfg *config.GlobalConfig) *cobra.Command {
	var compat bool

	c := &cobra.Command{
		Use:   "check <path@version>",
		Short: "Verify a published catalog out of band",
		Long: `Pull a published catalog by path@version and run the same verification a
consumer performs when a platform acquires it: the declared identity is
concrete, and its modulePath and version agree with the coordinate the build
was fetched by.
The report lists the catalog's members per kind and apiVersion.

	With --compat, additionally compare every beta/GA member against the last
	published build that shipped it, under the additive-only rule — exactly the
	comparison publish enforces.

	` + AidSentence + `

	Exit codes: 0 clean, 2 findings, 3 registry unreachable.

	Examples:
	  # Verify a published build's identity and see what it contains
	  opm catalog registry check opmodel.dev/catalogs/opm@v2.0.0-alpha.3

	  # Additionally check it kept its published contracts
	  opm catalog registry check opmodel.dev/catalogs/opm@v2.0.0-alpha.3 --compat`,
		Args: cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			report, err := publish.RegistryCheck(c.Context(), publish.CheckOptions{
				Coordinate: args[0],
				Compat:     compat,
				Context:    cuecontext.New(),
				Registry:   cfg.Registry,
			})
			if err != nil {
				return checkError(err)
			}

			fmt.Fprint(c.OutOrStdout(), report.Render())
			if report.Clean() {
				return nil
			}
			cmdutil.PrintRefusals(report.Findings)
			n := len(report.Findings)
			noun := "finding"
			if n != 1 {
				noun = "findings"
			}
			return &opmexit.ExitError{
				Code:    opmexit.ExitValidationError,
				Err:     fmt.Errorf("check reported %d %s", n, noun),
				Printed: true,
			}
		},
	}

	c.Flags().BoolVar(&compat, "compat", false,
		"Also compare the fetched build against its published predecessors")

	return c
}

// checkError maps check failures to exit codes: registry unreachability is 3
// (the build was never judged), a coordinate no build answers to is 5, and
// anything else is unexpected (1).
func checkError(err error) error {
	code := opmexit.ExitGeneralError
	var connErr *publish.ConnectivityError
	switch {
	case errors.As(err, &connErr):
		code = opmexit.ExitConnectivityError
	case errors.Is(err, publish.ErrNotPublished):
		code = opmexit.ExitNotFound
	}
	return &opmexit.ExitError{Code: code, Err: err}
}
