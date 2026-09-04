package render

import (
	"errors"
	"fmt"
	"strings"

	liberrors "github.com/open-platform-model/library/opm/errors"
	"github.com/open-platform-model/library/opm/kernel"

	"github.com/open-platform-model/cli/internal/cmdutil"
	"github.com/open-platform-model/cli/internal/output"
)

// renderFailedMsg is the header every render failure prints under.
const renderFailedMsg = "render failed"

// printValidationError prints a render failure in a user-friendly format.
//
// A *kernel.RenderError (the fail-closed gate after the build: unresolved
// demands, unmatched components, an over-subscribed provider contract, a
// failed pair) prints the kernel's message followed by the diagnostics rows
// it carries. A skew refusal (*liberrors.SkewError, before evaluation)
// prints the kernel's message verbatim: it already names the path and both
// versions. Everything else goes through the shared validation funnel
// (grouped CUE positions, values schema failures, unresolved demands raised
// outside a render).
func printValidationError(err error) {
	if err == nil {
		return
	}
	var renderErr *kernel.RenderError
	if errors.As(err, &renderErr) {
		output.Error(fmt.Sprintf("%s: %s", renderFailedMsg, renderErr.Err))
		if details := formatRenderDiagnostics(renderErr.Diagnostics); details != "" {
			output.Details(details)
		}
		return
	}
	var skewErr *liberrors.SkewError
	if errors.As(err, &skewErr) {
		output.Error(fmt.Sprintf("%s: %s", renderFailedMsg, err))
		return
	}
	cmdutil.PrintValidationError(renderFailedMsg, err)
}

// formatRenderDiagnostics renders the refusing rows of a render's
// diagnostics as one block, in the existing validation-output style: each
// unresolved demand with its same-base alternatives, each unmatched
// component, each over-subscribed contract key with the catalogs competing
// for it, and each matched pair whose transformer output failed. Rows that
// did not refuse (matched pairs, unhandled traits, version rows) are not
// repeated here.
func formatRenderDiagnostics(d kernel.RenderDiagnostics) string {
	var b strings.Builder
	if len(d.Unresolved) > 0 {
		b.WriteString(cmdutil.FormatUnresolvedDemands(&liberrors.UnresolvedDemandsError{Demands: d.Unresolved}))
		b.WriteString("\n")
	}
	for _, comp := range d.Unmatched {
		fmt.Fprintf(&b, "component %q: no transformer matched\n", comp)
	}
	for _, o := range d.OverSubscribed {
		fmt.Fprintf(&b, "contract %q: provided by more than one enabled catalog: %s\n", o.Key, strings.Join(o.Catalogs, ", "))
	}
	for _, p := range d.FailedPairs {
		fmt.Fprintf(&b, "component %q: transformer %s failed\n", p.Component, p.Transformer)
	}
	return strings.TrimRight(b.String(), "\n")
}
