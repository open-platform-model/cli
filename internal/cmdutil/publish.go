package cmdutil

import (
	"errors"
	"fmt"

	"cuelang.org/go/cue"
	"github.com/spf13/cobra"

	"github.com/open-platform-model/library/opm/kernel"
	"github.com/open-platform-model/library/opm/schema"

	"github.com/open-platform-model/cli/internal/config"
	opmexit "github.com/open-platform-model/cli/internal/exit"
	"github.com/open-platform-model/cli/internal/output"
	"github.com/open-platform-model/cli/internal/publish"
	oerrors "github.com/open-platform-model/cli/pkg/errors"
)

// PublishFlags holds the flags shared by `opm module publish` and
// `opm catalog publish`.
type PublishFlags struct {
	// Version fills an open identity Version or asserts a concrete one
	// (never overwrites — refusal 5).
	Version string
	// DryRun stops after the plan; every gate still runs, including the
	// already-published lookup.
	DryRun bool
	// SkipOverrideCheck waives the local-override gate. Module publish only;
	// catalog publish never registers it.
	SkipOverrideCheck bool
}

// AddTo registers the shared publish flags on the given command.
func (f *PublishFlags) AddTo(cmd *cobra.Command) {
	cmd.Flags().StringVar(&f.Version, "version", "",
		"Fill an open identity Version, or assert the declared one")
	cmd.Flags().BoolVar(&f.DryRun, "dry-run", false,
		"Print the resolved plan and stop; all gates run, nothing is pushed")
}

// RunPublish is the shared body of both publish commands: resolve core's
// #IdentityPackage from the kernel schema cache, compute the plan, print it,
// print any refusals through the validation funnel, and push on GO outside
// --dry-run. Exit codes: refusal 2, registry unreachable 3, unexpected 1.
func RunPublish(cmd *cobra.Command, cfg *config.GlobalConfig, kind publish.Kind, args []string, flags *PublishFlags) error {
	dir := ResolveModulePath(args)

	k := kernel.New(
		kernel.WithRegistry(cfg.Registry),
		kernel.WithSchemaLoader(schema.OCILoader{Registry: cfg.Registry}),
	)
	schemaVal, err := k.SchemaCache().Get(k.CueContext())
	if err != nil {
		// The schema fetch is a registry round-trip like the lookup and the
		// push: failing to reach it is a connectivity failure, not a verdict
		// on the artifact.
		return publishError(&publish.ConnectivityError{Op: "loading core schema", Err: err})
	}
	identitySchema := schemaVal.LookupPath(cue.MakePath(cue.Def("IdentityPackage")))
	if !identitySchema.Exists() {
		return &opmexit.ExitError{
			Code: opmexit.ExitGeneralError,
			Err:  fmt.Errorf("resolved core schema (%s) does not define #IdentityPackage; publish requires a core v2 schema", k.SchemaCache().ResolvedVersion()),
		}
	}

	opts := publish.Options{
		Dir:               dir,
		Kind:              kind,
		Context:           k.CueContext(),
		IdentitySchema:    identitySchema,
		Registry:          cfg.Registry,
		VersionFlag:       flags.Version,
		SkipOverrideCheck: flags.SkipOverrideCheck,
	}

	plan, runErr := publish.Run(cmd.Context(), opts)

	// The plan prints on every invocation before any push — it is the dry
	// run's whole output and the real run's preamble. That holds even when
	// the registry was unreachable (runErr non-nil with a plan): every
	// locally-derived refusal is already on the plan, and hiding it behind
	// the connectivity error would bury the problems the author can fix
	// right now.
	if plan != nil {
		fmt.Fprint(cmd.OutOrStdout(), plan.Render())
		if !plan.Go() {
			PrintRefusals(plan.Refusals)
		}
	}
	if runErr != nil {
		return publishError(runErr)
	}

	if !plan.Go() {
		n := len(plan.Refusals)
		noun := "refusal"
		if n != 1 {
			noun = "refusals"
		}
		return &opmexit.ExitError{
			Code:    opmexit.ExitValidationError,
			Err:     fmt.Errorf("publish refused: %d %s", n, noun),
			Printed: true,
		}
	}

	if flags.DryRun {
		return nil
	}

	if err := publish.Push(cmd.Context(), opts, plan); err != nil {
		return publishError(err)
	}
	output.Info(output.FormatCheckmark(fmt.Sprintf("Published %s:%s", plan.RegistryRepo, plan.Tag)))
	return nil
}

// PrintRefusals routes each refusal through the standard validation-error
// funnel: CUE-carrying refusals (load failures, identity non-conformance)
// through the grouped path that preserves file:line, drafted messages as
// ValidationError with aligned-column details. Shared by publish and the
// vet checks.
func PrintRefusals(refusals []publish.Refusal) {
	for _, r := range refusals {
		if r.Err != nil {
			PrintValidationError(r.Headline, r.Err)
			continue
		}
		PrintValidationError("refused", &oerrors.ValidationError{
			Message: r.Headline,
			Details: r.Details(),
		})
	}
}

// publishError maps pipeline errors to exit codes: registry unreachability
// is a connectivity failure (3) — the artifact was never judged — and
// anything else is unexpected (1).
func publishError(err error) error {
	code := opmexit.ExitGeneralError
	var connErr *publish.ConnectivityError
	if errors.As(err, &connErr) {
		code = opmexit.ExitConnectivityError
	}
	return &opmexit.ExitError{Code: code, Err: err}
}
