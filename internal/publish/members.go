package publish

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"cuelang.org/go/cue"
)

// Member is one catalog member as the walk found it: which kind directory it
// files under, what its metadata declares, and the loaded definition value the
// gates evaluate.
type Member struct {
	// Kind is the kind directory the member files under ("resources",
	// "traits", "blueprints", "transformers") — the filing, not the
	// discriminator. The FQN gate takes it as its kind input, which is what
	// makes wrong-kind filing surface as CUE conflicts.
	Kind string
	// Discriminator is the member's own concrete kind field ("Resource",
	// "Trait", …) — what identified the definition as a member. The posture
	// gate selects traits by it, so a trait filed under the wrong kind is
	// still posture-gated.
	Discriminator string
	// DefName is the definition's label ("#ConfigMapsResource"), for display.
	DefName string
	// Name is metadata.name, when concrete ("" otherwise — the FQN gate
	// refuses the absence).
	Name string
	// APIVersion is metadata.apiVersion, when concrete; "" for transformers
	// (structurally, D44) and for members that fail to state one.
	APIVersion string
	// Value is the loaded definition value.
	Value cue.Value
	// PkgPath is the member's package directory relative to the module root,
	// slash-separated ("resources/v1beta1", "transformers") — the subpackage
	// the predecessor scan probes at each published build.
	PkgPath string
}

// memberKindDirs are the contract kind directories the walk enumerates;
// transformers file flat and are walked separately.
var memberKindDirs = []string{"resources", "traits", "blueprints"}

// memberDiscriminators maps a member definition's concrete kind field to the
// kind directory it belongs under. The discriminator is what separates members
// from the fragments and schemas defined beside them (D22): a definition
// without one is not a member and is never gated.
var memberDiscriminators = map[string]string{
	"Resource":             "resources",
	"Trait":                "traits",
	"Blueprint":            "blueprints",
	"ComponentTransformer": "transformers",
}

// enumerateMembers walks the catalog tree's member packages — every
// `<kind>/<apiVersion>/` directory, a kind directory's own package when it
// holds files directly (a flat-filed contract member must reach the gate, not
// hide from it), and the flat `transformers/` package — and returns every
// member definition found. The walk is filesystem-driven because a value walk
// cannot enumerate the tree: #Catalog exposes only #transformers, which reach
// about half the contract members and no blueprints (measured, design.md). A
// directory walk also visits only this catalog's own definitions, so
// cross-catalog references (0010 D17) are excluded structurally.
//
// A member package that fails to load is a refusal — the tree's root loaded,
// so a broken subpackage is a finding about the member tree, not a load
// short-circuit.
func enumerateMembers(opts Options, dir string) ([]Member, []Refusal) {
	var members []Member
	var refusals []Refusal

	walkPkg := func(pkgPath string) {
		val, _, _, refusal := loadPackage(opts, dir, "./"+pkgPath)
		if refusal != nil {
			refusal.Headline = fmt.Sprintf("the %s package does not load, so its members cannot be gated", pkgPath)
			refusals = append(refusals, *refusal)
			return
		}
		members = append(members, membersOfPackage(val, pkgPath)...)
	}

	for _, kind := range memberKindDirs {
		kindDir := filepath.Join(dir, kind)
		if hasCueFiles(kindDir) {
			walkPkg(kind)
		}
		entries, err := os.ReadDir(kindDir)
		if err != nil {
			continue // no such kind directory — nothing of that kind to gate
		}
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			sub := filepath.Join(kindDir, e.Name())
			if hasCueFiles(sub) {
				walkPkg(path.Join(kind, e.Name()))
			}
		}
	}

	if hasCueFiles(filepath.Join(dir, "transformers")) {
		walkPkg("transformers")
	}

	return members, refusals
}

// membersOfPackage iterates a loaded package's top-level definitions and
// returns those carrying a concrete member kind discriminator. The kind
// recorded on each member is the package's kind directory — where the member
// actually files — never the discriminator, so a member filed under a foreign
// kind carries that filing into the FQN gate.
func membersOfPackage(val cue.Value, pkgPath string) []Member {
	dirKind, _, _ := strings.Cut(pkgPath, "/")

	var out []Member
	iter, err := val.Fields(cue.Definitions(true))
	if err != nil {
		return nil
	}
	for iter.Next() {
		sel := iter.Selector()
		if !sel.IsDefinition() {
			continue
		}
		def := iter.Value()
		disc, err := def.LookupPath(cue.ParsePath("kind")).String()
		if err != nil {
			continue // no concrete discriminator — a fragment or schema, not a member
		}
		if _, ok := memberDiscriminators[disc]; !ok {
			continue // some other kinded definition (e.g. a Component wrapper)
		}
		m := Member{
			Kind:          dirKind,
			Discriminator: disc,
			DefName:       sel.String(),
			Value:         def,
			PkgPath:       pkgPath,
		}
		if s, err := def.LookupPath(cue.ParsePath("metadata.name")).String(); err == nil {
			m.Name = s
		}
		if s, err := def.LookupPath(cue.ParsePath("metadata.apiVersion")).String(); err == nil {
			m.APIVersion = s
		}
		out = append(out, m)
	}
	return out
}

// hasCueFiles reports whether dir directly contains at least one .cue file.
func hasCueFiles(dir string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".cue") {
			return true
		}
	}
	return false
}
