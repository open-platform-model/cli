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
		if details := formatRenderDiagnostics(renderErr.Diagnostics, output.IsVerbose()); details != "" {
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
//
// In verbose mode each unmatched component is followed by the verdict on
// every candidate transformer the build evaluated for it — the required
// labels the predicate found missing, or the primitive FQNs the always-unify
// rung conflicted at — the evidence the kernel now carries on the rows. The
// default output keeps its one line per component.
func formatRenderDiagnostics(d kernel.RenderDiagnostics, verbose bool) string {
	var b strings.Builder
	if len(d.Unresolved) > 0 {
		b.WriteString(cmdutil.FormatUnresolvedDemands(d.Unresolved))
		b.WriteString("\n")
	}
	for _, comp := range d.Unmatched {
		fmt.Fprintf(&b, "component %q: no transformer matched\n", comp.Component)
		if !verbose {
			continue
		}
		for _, v := range comp.Candidates {
			if v.Matched {
				continue
			}
			fmt.Fprintf(&b, "  candidate %q did not match: %s\n", v.Transformer, candidateReason(d.Unify, comp.Component, v))
		}
	}
	for _, o := range d.OverSubscribed {
		fmt.Fprintf(&b, "contract %q: provided by more than one enabled catalog: %s\n", o.Key, strings.Join(o.Catalogs, ", "))
	}
	for _, p := range d.FailedPairs {
		fmt.Fprintf(&b, "component %q: transformer %s failed\n", p.Component, p.Transformer)
	}
	return strings.TrimRight(b.String(), "\n")
}

// candidateReason words why one candidate was refused for a component: the
// labels the predicate rung found missing or divergent, else the primitive
// FQNs its unify refusal (looked up on the diagnostics' Unify rows, which the
// candidate verdict does not repeat) conflicted at.
func candidateReason(unify []liberrors.UnifyRefusal, component string, v liberrors.CandidateVerdict) string {
	if len(v.MissingLabels) > 0 {
		return "missing labels " + strings.Join(v.MissingLabels, ", ")
	}
	for _, u := range unify {
		if u.Component == component && u.Transformer == v.Transformer && len(u.Conflicts) > 0 {
			return "bodies conflict at " + strings.Join(u.Conflicts, ", ")
		}
	}
	return "bodies do not unify"
}
