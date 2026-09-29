package render

import (
	"fmt"
	"strings"

	"github.com/open-platform-model/library/opm/kernel"
)

// formatSkipped words the demands a render skipped under --skip-unprovided
// as the CLI's warnings, in build order: one line per skipped trait of a
// component that rendered, and one line per omitted component (the kernel
// marks every row of it ComponentOmitted) naming every skipped resource of
// it, and any trait it also skipped, at the position of its first row. A row
// the kernel reports alternatives for names them. Never nil: no skipped rows
// is an empty list.
func formatSkipped(rows []kernel.SkippedDemand) []string {
	omitted := map[string][]kernel.SkippedDemand{}
	for _, r := range rows {
		if r.ComponentOmitted {
			omitted[r.Component] = append(omitted[r.Component], r)
		}
	}
	lines := []string{}
	worded := map[string]bool{}
	for _, r := range rows {
		if !r.ComponentOmitted {
			lines = append(lines, fmt.Sprintf(
				"component %q: skipped provider-fulfilled trait %q (no provider on this platform)%s",
				r.Component, r.FQN, alternativesSuffix(r.Alternatives)))
			continue
		}
		if !worded[r.Component] {
			worded[r.Component] = true
			lines = append(lines, omittedLine(r.Component, omitted[r.Component]))
		}
	}
	return lines
}

// omittedLine words one omitted component from all of its skipped rows:
// the resources that dropped it, then any trait it also skipped. With one
// row carrying alternatives the suffix is the trait line's; with several,
// each suffix names the contract it belongs to.
func omittedLine(component string, rows []kernel.SkippedDemand) string {
	var resources, traits []kernel.SkippedDemand
	for _, r := range rows {
		if r.Kind == "trait" {
			traits = append(traits, r)
		} else {
			resources = append(resources, r)
		}
	}
	var b strings.Builder
	fmt.Fprintf(&b, "component %q not rendered: ", component)
	if len(resources) == 1 {
		fmt.Fprintf(&b, "provider-fulfilled resource %q has no provider on this platform", resources[0].FQN)
	} else {
		fmt.Fprintf(&b, "provider-fulfilled resources %s have no provider on this platform", quotedFQNs(resources))
	}
	if len(traits) > 0 {
		fmt.Fprintf(&b, "; also skipped provider-fulfilled trait(s) %s", quotedFQNs(traits))
	}
	single := len(rows) == 1
	for _, r := range rows {
		if len(r.Alternatives) == 0 {
			continue
		}
		if single {
			b.WriteString(alternativesSuffix(r.Alternatives))
		} else {
			fmt.Fprintf(&b, "; %q implemented at: %s", r.FQN, strings.Join(r.Alternatives, ", "))
		}
	}
	return b.String()
}

// alternativesSuffix is the "; implemented at: <keys>" tail of a skipped
// row the kernel reports same-base alternatives for, "" otherwise.
func alternativesSuffix(alternatives []string) string {
	if len(alternatives) == 0 {
		return ""
	}
	return "; implemented at: " + strings.Join(alternatives, ", ")
}

func quotedFQNs(rows []kernel.SkippedDemand) string {
	quoted := make([]string, len(rows))
	for i, r := range rows {
		quoted[i] = fmt.Sprintf("%q", r.FQN)
	}
	return strings.Join(quoted, ", ")
}
