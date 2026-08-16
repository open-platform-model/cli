package publish

import (
	"fmt"
	"strings"

	"cuelang.org/go/cue"
	cueerrors "cuelang.org/go/cue/errors"
)

// identityFieldNames are the schema-fixed paths publish reads, per
// core.#IdentityPackage. There is no marker to scan for — the name is where
// the schema puts the field, and a field not at its path is absent (a
// malformed artifact, not a cue to search wider).
var identityFieldNames = []string{"ModulePath", "Version"}

// conformIdentity unifies the identity package against core's
// #IdentityPackage and returns refusal 10 on failure. CUE's own error is
// surfaced — a procedural expected-versus-found check would be a second
// statement of the contract, and the two drift.
func conformIdentity(a *artifact, schema cue.Value) *Refusal {
	unified := schema.Unify(a.identity)
	err := unified.Validate(cue.Concrete(true))
	if err == nil {
		return nil
	}

	// Concrete validation is what makes CUE name a missing required field
	// ("field is required but not present"). It also flags open fields as
	// incomplete — but openness is D4's gate (refusal 1, fillable by
	// --version), not a conformance failure, so incompleteness errors are
	// dropped here. One cause, one refusal.
	var kept cueerrors.Error
	for _, e := range cueerrors.Errors(err) {
		if strings.Contains(e.Error(), "incomplete value") {
			continue
		}
		kept = cueerrors.Append(kept, e)
	}
	if kept == nil {
		return nil
	}
	return &Refusal{
		Headline: "the identity package does not conform to core's #IdentityPackage",
		Err:      kept,
	}
}

// identityStates classifies each schema-fixed identity field per D4's
// tristate, evaluating with CUE defaults: a defaulted field is concrete with
// the default as its value — that is the committed, release-automation-owned
// declaration.
func identityStates(a *artifact) []IdentityField {
	out := make([]IdentityField, 0, len(identityFieldNames))
	for _, name := range identityFieldNames {
		f := IdentityField{Name: name, State: StateAbsent}
		v := a.identity.LookupPath(cue.ParsePath(name))
		if v.Exists() {
			f.State = StateOpen
			f.Pos = fieldPos(a.dir, v)
			if s, err := v.String(); err == nil {
				f.State, f.Value = StateConcrete, s
				_, f.Defaulted = v.Default()
			}
		}
		out = append(out, f)
	}
	return out
}

// resolveVersion applies D3/D12's --version semantics to the Version field
// and sets the plan's tag: fill an open field (recording the pending
// working-tree write), assert a concrete one, never overwrite. One cause,
// one refusal: an absent field is refusal 10's finding and produces nothing
// here.
func resolveVersion(p *Plan, opts Options) {
	ver := p.identityField("Version")
	if ver == nil || ver.State == StateAbsent {
		return
	}

	flag := opts.VersionFlag
	switch {
	case ver.State == StateConcrete && flag != "" && flag != ver.Value:
		p.refuse(Refusal{
			Headline: "--version disagrees with the version this artifact declares",
			Evidence: [][]string{
				{"--version", flag},
				{"declared", ver.Value, ver.Pos},
			},
			Consequence: "--version fills an open field or asserts a concrete one. It never\noverwrites a value someone decided.",
			Action:      fmt.Sprintf("Change the flag, or:  opm %s version set %s", p.Kind, flag),
		})
	case ver.State == StateConcrete:
		p.Tag = "v" + ver.Value
		p.TagSource = "the artifact's own Version"
		if ver.Defaulted {
			p.TagSource += " (default)"
		}
	case flag != "":
		// Open + flag: the fill writes the working tree (D12) — an in-memory
		// overlay would publish bytes the tree does not hold. Push performs
		// the write; the plan records it.
		ver.Filled = true
		ver.Value = flag
		p.FillVersion = flag
		p.Tag = "v" + flag
		p.TagSource = "--version (fills the open Version)"
	default:
		artifactName := p.DeclaredPath
		if artifactName == "" {
			artifactName = p.Dir
		}
		p.refuse(Refusal{
			Headline: fmt.Sprintf("%s declares an identity field that has no value", artifactName),
			Evidence: [][]string{
				{"Version", ver.Pos, "declared, never filled"},
			},
			Consequence: "This artifact would publish, and every consumer would fail at evaluation\nwith `incomplete value` — a producer's mistake, discoverable only downstream.",
			Action:      fmt.Sprintf("Supply it:  opm %s version set <version>\n        or: opm %s publish --version <version>", p.Kind, p.Kind),
		})
	}
}

// requireConcreteModulePath refuses when ModulePath is declared but unfilled
// (msg 1's shape — --version does not apply to it). Absent is refusal 10's
// finding. Returns the concrete declared path, or "".
func requireConcreteModulePath(p *Plan) string {
	mp := p.identityField("ModulePath")
	if mp == nil || mp.State == StateAbsent {
		return ""
	}
	if mp.State == StateOpen {
		artifactName := p.Dir
		p.refuse(Refusal{
			Headline: fmt.Sprintf("%s declares an identity field that has no value", artifactName),
			Evidence: [][]string{
				{"ModulePath", mp.Pos, "declared, never filled"},
			},
			Consequence: "There is nothing to derive a registry coordinate from; publish reads the\ndeclared path and refuses to invent one.",
			Action:      "Declare it in identity/identity.cue:  ModulePath: \"<host>/<path>@v<major>\"",
		})
		return ""
	}
	return mp.Value
}

// identityField returns the plan's state for the named field, or nil.
func (p *Plan) identityField(name string) *IdentityField {
	for i := range p.Identity {
		if p.Identity[i].Name == name {
			return &p.Identity[i]
		}
	}
	return nil
}
