package publish

import (
	"fmt"

	"cuelang.org/go/cue"
)

// CatalogGateOutcomes carries the per-gate counts the plan renders for a
// catalog publish: how many members each gate evaluated and how many it
// refused. Compat counts are filled by the compatibility gate.
type CatalogGateOutcomes struct {
	// MembersChecked / MembersRefused: the #CatalogMemberFQNGate pass over
	// every member, all four kinds, all levels.
	MembersChecked int
	MembersRefused int
	// TraitsChecked / TraitsRefused: the #TraitOptionalGate pass over every
	// trait's optional field.
	TraitsChecked int
	TraitsRefused int
	// Compat counts: members compared against a predecessor, refused by the
	// comparison, skipped as alpha (D34), and passing as new at their
	// name-and-apiVersion key.
	CompatCompared int
	CompatRefused  int
	CompatAlpha    int
	CompatNew      int
}

// gateMemberFQN is D22's structural gate (refusal 11): every member of the
// tree — all four kinds, all levels; D34's alpha carve-out is compat-only —
// is unified against core's #CatalogMemberFQNGate and validated CONCRETELY.
// The pipeline's incomplete-value filter (conformIdentity) is deliberately
// not reused: here incompleteness is the finding — a missing
// declaredAPIVersion passes plain and -c vet alike and surfaces only under
// concrete evaluation (measured, core's identity_package_pins.cue). CUE's
// error is the refusal; nothing is hand-written.
func gateMemberFQN(p *Plan, opts Options, a *artifact, members []Member) {
	for _, m := range members {
		p.CatalogGates.MembersChecked++

		gv := opts.MemberFQNGateSchema
		gv = gv.FillPath(cue.ParsePath("identity"), a.identity)
		gv = gv.FillPath(cue.ParsePath("kind"), m.Kind)
		gv = fillFromMember(gv, m, "name", "metadata.name")
		gv = fillFromMember(gv, m, "declaredFQN", "metadata.fqn")
		gv = fillFromMember(gv, m, "declaredModulePath", "metadata.modulePath")
		gv = fillFromMember(gv, m, "declaredCatalogVersion", "metadata.catalogVersion")
		gv = fillFromMember(gv, m, "declaredAPIVersion", "metadata.apiVersion")

		err := gv.Validate(cue.Concrete(true), cue.Hidden(true), cue.Definitions(true))
		if err == nil {
			continue
		}
		p.CatalogGates.MembersRefused++
		p.refuse(Refusal{
			Headline: fmt.Sprintf("%s (%s) is off its catalog's key space", m.DefName, m.PkgPath),
			Err:      err,
		})
	}
}

// fillFromMember fills one gate field from a member's metadata when the field
// is present. An absent field is left unfilled — the gate's own required
// marker reports it under concrete evaluation, in CUE's words.
func fillFromMember(gate cue.Value, m Member, gateField, memberPath string) cue.Value {
	v := m.Value.LookupPath(cue.ParsePath(memberPath))
	if !v.Exists() {
		return gate
	}
	return gate.FillPath(cue.ParsePath(gateField), v)
}

// gateTraitOptional is D22's posture gate: every trait's `optional` FIELD —
// never the whole member, which would drag its schema `spec` into
// concreteness — is unified against core's #TraitOptionalGate. An unstated
// posture is an incomplete value (rule 1, visible only concretely) and a
// pinned one a conflict (rule 2); both refuse. Traits are selected by their
// own discriminator, so a trait filed under a foreign kind is still gated,
// and v1alpha1 traits are included — the alpha carve-out is compat-only.
func gateTraitOptional(p *Plan, opts Options, members []Member) {
	for _, m := range members {
		if m.Discriminator != "Trait" {
			continue
		}
		p.CatalogGates.TraitsChecked++

		gv := opts.TraitOptionalGateSchema
		if opt := m.Value.LookupPath(cue.ParsePath("optional")); opt.Exists() {
			gv = gv.FillPath(cue.ParsePath("optional"), opt)
		}

		err := gv.Validate(cue.Concrete(true), cue.Hidden(true), cue.Definitions(true))
		if err == nil {
			continue
		}
		p.CatalogGates.TraitsRefused++
		p.refuse(Refusal{
			Headline: fmt.Sprintf("%s (%s) does not state an overridable optional posture", m.DefName, m.PkgPath),
			Err:      err,
		})
	}
}
