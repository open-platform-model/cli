package cmdutil

import (
	"errors"
	"fmt"

	"github.com/open-platform-model/cli/internal/cueedit"
	opmexit "github.com/open-platform-model/cli/internal/exit"
	"github.com/open-platform-model/cli/internal/output"
	"github.com/open-platform-model/cli/internal/publish"
)

// RunVersionSet is the shared body of `opm module version set` and
// `opm catalog version set`: rewrite identity/identity.cue's Version in
// place, idempotently and offline (0011 D3, D8). Setting the already-declared
// version touches nothing — no write, no mtime change — and reports the no-op
// as success; both outcomes exit 0. A structurally non-conformant identity
// file is refused (exit 2) through the standard funnel, pointing at
// `opm module vet` for full conformance. No registry I/O ever happens here —
// type-level defects are vet's and publish's to catch.
func RunVersionSet(kind publish.Kind, version string, pathArgs []string) error {
	dir := ResolveModulePath(pathArgs)

	if err := cueedit.CheckVersion(version); err != nil {
		return &opmexit.ExitError{Code: opmexit.ExitValidationError, Err: err}
	}

	current, _, err := cueedit.ReadIdentityVersion(dir)
	if err != nil {
		return versionSetError(err)
	}
	if current == version {
		output.Info(fmt.Sprintf("Version already %s; identity/identity.cue untouched", version))
		return nil
	}

	if _, err := cueedit.SetIdentityVersion(dir, version); err != nil {
		return versionSetError(err)
	}
	if current == "" {
		output.Info(output.FormatCheckmark(fmt.Sprintf("Set %s version %s (identity/identity.cue)", kind, version)))
	} else {
		output.Info(output.FormatCheckmark(fmt.Sprintf("Set %s version %s -> %s (identity/identity.cue)", kind, current, version)))
	}
	return nil
}

// versionSetError maps writer failures to exit codes: an identity shape
// mismatch is a refusal (2) printed through the refusal funnel — the message
// already names the file and the path element it failed to find — and
// anything else is unexpected (1).
func versionSetError(err error) error {
	if errors.Is(err, cueedit.ErrIdentityShape) {
		PrintRefusals([]publish.Refusal{{
			Headline:    err.Error(),
			Consequence: "Nothing was written.",
			Action:      "Fix the identity package, then check full conformance:  opm module vet",
		}})
		return &opmexit.ExitError{
			Code:    opmexit.ExitValidationError,
			Err:     err,
			Printed: true,
		}
	}
	return &opmexit.ExitError{Code: opmexit.ExitGeneralError, Err: err}
}
