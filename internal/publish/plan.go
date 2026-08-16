package publish

import (
	"fmt"
	"strings"
)

// Render prints the resolved plan in experiment 02's measured format: kind,
// declared path, cue.mod path, repository, major, tag and its source, one
// line per identity field, the override gate, then the verdict. The plan
// prints on every invocation before any push — it is #PublishPlan rendered,
// and it doubles as the dry run.
//
// Refusal details are not repeated here; they print through the validation
// funnel. The verdict line carries the count.
func (p *Plan) Render() string {
	var b strings.Builder
	row := func(label, value string) {
		if value == "" {
			value = "—"
		}
		fmt.Fprintf(&b, "  %-15s %s\n", label, value)
	}

	row("kind", string(p.Kind))
	row("declaredPath", p.DeclaredPath)
	row("cueModPath", p.CueModPath)
	row("registryRepo", p.RegistryRepo)
	row("major", p.Major)
	row("tag", p.Tag)
	if p.TagSource != "" {
		row("tag from", p.TagSource)
	}
	for _, f := range p.Identity {
		state := string(f.State)
		if f.Defaulted {
			state += " (default)"
		}
		if f.Filled {
			state += " (filled by --version)"
		}
		v := ""
		if f.Value != "" {
			v = " = " + f.Value
		}
		fmt.Fprintf(&b, "  %-15s %-11s %s%s\n", "identity", f.Name, state, v)
	}
	override := fmt.Sprintf("%v", p.OverridePresent)
	if p.OverrideWaived {
		override += " (gate waived by --skip-override-check; replacements are ignored either way)"
	}
	row("local override", override)

	// The catalog gates' per-gate outcomes (D22, D9). Text-only — plan --json
	// remains deferred.
	if p.Kind == KindCatalog {
		g := p.CatalogGates
		row("member gate", fmt.Sprintf("%d members checked, %d refused", g.MembersChecked, g.MembersRefused))
		row("posture gate", fmt.Sprintf("%d traits checked, %d refused", g.TraitsChecked, g.TraitsRefused))
		compat := fmt.Sprintf("%d compared, %d refused, %d alpha-exempt, %d new",
			g.CompatCompared, g.CompatRefused, g.CompatAlpha, g.CompatNew)
		if !p.CompatChecked {
			compat = "not completed"
		}
		row("compat gate", compat)
	}

	// A catalog's registry work is done only when the compatibility walk
	// finished too — a walk aborted mid-flight must not render the GO the
	// gates never finished earning.
	registryDone := p.RegistryChecked && (p.Kind != KindCatalog || p.CompatChecked)

	switch {
	case p.Go() && registryDone:
		fmt.Fprintf(&b, "\n  GO — pushing %s:%s\n", p.RegistryRepo, p.Tag)
	case p.Go():
		// Every local gate passed but the already-published lookup never
		// completed (registry unreachable): a GO here would be a verdict the
		// gates never finished earning.
		fmt.Fprintf(&b, "\n  INCOMPLETE — every local gate passed, but the registry could not be\n  reached for the already-published check; nothing was pushed\n")
	default:
		noun := "refusal"
		if len(p.Refusals) != 1 {
			noun = "refusals"
		}
		fmt.Fprintf(&b, "\n  REFUSED — %d %s\n", len(p.Refusals), noun)
	}
	return b.String()
}
