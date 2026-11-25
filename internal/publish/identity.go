package publish

import (
	"fmt"

	"cuelang.org/go/cue"
)

// identityFieldNames are the schema-fixed paths publish reads, per
// core.#IdentityPackage. There is no marker to scan for — the name is where
// the schema puts the field, and a field not at its path is absent (a
// malformed artifact, not a cue to search wider).
var identityFieldNames = []string{"ModulePath", "Version"}

// identityFile is the file of the identity package that declares the
// identity fields, relative to the artifact root.
const identityFile = "identity/identity.cue"

// conformIdentity checks the identity package against core's
// #IdentityPackage and returns refusal 10 on failure. fill, when non-empty,
// is the --version value for an open Version: the identity is checked as
// filled, so a fill that conflicts with the schema is refused before any push.
//
// Plain validation reports conflicts only: an open field, and anything core
// derives from it, is incomplete rather than wrong, and passes, because
// openness is the gate for open fields (fillable by --version). CUE's own
// error is surfaced: a procedural expected-versus-found check would be a
// second statement of the contract, and the two drift.
//
// Plain validation does not report an absent required field, so the absent
// schema-fixed fields are read from the identity value and refused by name.
func conformIdentity(a *artifact, schema cue.Value, fill string) []Refusal {
	id := a.identity
	if fill != "" {
		id = id.FillPath(cue.ParsePath("Version"), fill)
	}
	if err := schema.Unify(id).Validate(); err != nil {
		return []Refusal{{
			Headline: "the identity package does not conform to core's #IdentityPackage",
			Err:      err,
		}}
	}
	var out []Refusal
	for _, name := range identityFieldNames {
		if id.LookupPath(cue.ParsePath(name)).Exists() {
			continue
		}
		out = append(out, absentIdentityFieldRefusal(name))
	}
	return out
}

// absentIdentityFieldRefusal refuses an identity package that does not
// declare the named schema-fixed field. It names the field and the file that
// declares the identity.
func absentIdentityFieldRefusal(name string) Refusal {
	example := `Version: "<major>.<minor>.<patch>"`
	if name == "ModulePath" {
		example = `ModulePath: "<host>/<path>@v<major>"`
	}
	return Refusal{
		Headline: fmt.Sprintf("the identity package does not declare %s, which core's #IdentityPackage requires", name),
		Evidence: [][]string{
			{name, identityFile, "absent"},
		},
		Consequence: "Publish reads the field at the path the schema fixes and searches no wider.",
		Action:      fmt.Sprintf("Declare it in %s:  %s", identityFile, example),
	}
}

// identityStates classifies each schema-fixed identity field per the
// tristate of enhancement 0011, evaluating with CUE defaults: a defaulted field is concrete with
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

// resolveVersion applies the --version semantics of enhancement 0011 to the Version field
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
		// Open + flag: the fill writes the working tree (enhancement 0011) — an in-memory
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
			Action:      "Declare it in " + identityFile + ":  ModulePath: \"<host>/<path>@v<major>\"",
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
