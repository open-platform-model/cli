package publish

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"cuelang.org/go/cue"

	"github.com/open-platform-model/cli/pkg/loader"
)

// Owned-domain shapes (D13/D14): the namespace rules bind only where OPM owns
// the domain. Outside opmodel.dev and community.opmodel.dev publish asserts
// nothing about the path — path ownership is domain ownership.
var (
	firstPartyShape = regexp.MustCompile(`^opmodel\.dev/(modules|catalogs|platforms)/[a-z0-9._-]+$`)
	communityShape  = regexp.MustCompile(`^community\.opmodel\.dev/(m|catalogs|p)/[a-z0-9._-]+/[a-z0-9._-]+$`)
)

// Run computes the publish plan for one artifact: load, read identity, derive
// coordinates, evaluate every evaluable gate, and accumulate refusals. It
// never pushes and never writes; Push does both, after the caller has printed
// the plan.
//
// A tree that does not load short-circuits with the load error — nothing
// downstream is evaluable. Everything else accumulates, so one pass fixes
// everything. The only registry round-trip is the already-published lookup;
// an unreachable registry there returns a *ConnectivityError rather than a
// refusal.
func Run(ctx context.Context, opts Options) (*Plan, error) {
	if opts.Context == nil {
		return nil, fmt.Errorf("publish: Options.Context is required")
	}
	if !opts.IdentitySchema.Exists() {
		return nil, fmt.Errorf("publish: Options.IdentitySchema is required")
	}
	absDir, err := statArtifactDir(opts.Dir)
	if err != nil {
		return nil, err
	}

	p := &Plan{Kind: opts.Kind, Dir: absDir}

	// Gate: a tree without cue.mod is not a CUE module (D16, D20). Publish
	// reads the module path from cue.mod and refuses to invent one, so
	// nothing downstream is evaluable — short-circuit.
	if !hasCueMod(absDir) {
		p.refuse(noCueModRefusal(opts, absDir))
		return p, nil
	}

	// Gate: the tree and its identity package must load. The load error is
	// surfaced verbatim — reporting a failed load as "identity absent" would
	// blame the wrong thing.
	a, refusal := loadArtifact(opts)
	if refusal != nil {
		p.refuse(*refusal)
		return p, nil
	}
	if err := a.readCueMod(opts.Context); err != nil {
		p.refuse(Refusal{Headline: "cue.mod/module.cue cannot be read", Err: err})
		return p, nil //nolint:nilerr // the read failure IS the refusal; the plan carries it
	}
	p.CueModPath, p.CueModPos = a.cueModPath, a.cueModPos

	// Gate: identity conforms to #IdentityPackage (D21) — CUE's diagnostic,
	// through the grouped funnel (refusal 10). Downstream gates still read
	// the fields that do resolve.
	if r := conformIdentity(a, opts.IdentitySchema); r != nil {
		p.refuse(*r)
	}

	// Identity states (D4 tristate, defaults are concrete), coordinate
	// derivation (read and split the declared path — never composed), and
	// --version assert/fill (D3/D12).
	p.Identity = identityStates(a)
	p.DeclaredPath = requireConcreteModulePath(p)
	if before, after, ok := strings.Cut(p.DeclaredPath, "@"); ok {
		p.RegistryRepo, p.Major = before, after
	}
	resolveVersion(p, opts)

	// The remaining gates, in dependency order; all evaluable ones run.
	gateSourceSelf(p, a)
	gateCueModAgreement(p, a)
	gateDerivation(p, a)
	gateTagMajor(p)
	gateNamespace(p)
	gateKindSegment(p)
	gatePackageName(p, a)
	gateOverride(p, opts)

	// The catalog-only local gates (D22): enumerate the member tree once,
	// unify every member against the FQN gate and every trait's posture
	// against the optional gate. Both are local — no registry is touched.
	if p.Kind == KindCatalog {
		if !opts.MemberFQNGateSchema.Exists() || !opts.TraitOptionalGateSchema.Exists() {
			return nil, fmt.Errorf("publish: Options.MemberFQNGateSchema and Options.TraitOptionalGateSchema are required for catalog publish")
		}
		var refusals []Refusal
		p.members, refusals = enumerateMembers(opts, absDir)
		for _, r := range refusals {
			p.refuse(r)
		}
		gateMemberFQN(p, opts, a, p.members)
		gateTraitOptional(p, opts, p.members)
	}

	// Gate: the tag must not already be published (D15) — the one registry
	// round-trip before the push. The dry run runs it too: a dry run
	// surfaces rejections, never defers them.
	if err := gateAlreadyPublished(ctx, p, opts); err != nil {
		return p, err
	}

	return p, nil
}

// noCueModRefusal is msg 3. When the identity package still loads on its own,
// the action names the declared path; otherwise it shows the placeholder.
func noCueModRefusal(opts Options, absDir string) Refusal {
	modPath := "<module-path>"
	if idVal, _, _, refusal := loadPackage(opts, absDir, "./identity"); refusal == nil {
		if s, err := idVal.LookupPath(cue.ParsePath("ModulePath")).String(); err == nil {
			modPath = s
		}
	}
	return Refusal{
		Headline:    fmt.Sprintf("no cue.mod/module.cue under %s — this is not a CUE module", absDir),
		Consequence: "Publish reads the module path from cue.mod and refuses to invent one.",
		//nolint:misspell // "Initialise" is the drafted message's spelling (0011 06-operational.md, msg 3) — the messages are normative
		Action: fmt.Sprintf("Initialise it:  opm mod init %s", modPath),
	}
}

// gateSourceSelf requires cue.mod's source: {kind: "self"} (push-mechanism
// preflight): `self` selects the whole-directory zip, and CUE's publish
// machinery refuses without it — better a gate in the house shape than a
// late CUE error.
func gateSourceSelf(p *Plan, a *artifact) {
	if a.sourceKind == "self" {
		return
	}
	evidence := [][]string{{"source", "absent", "cue.mod/module.cue"}}
	if a.sourceKind != "" {
		evidence = [][]string{{"source", fmt.Sprintf("{kind: %q}", a.sourceKind), "cue.mod/module.cue"}}
	}
	p.refuse(Refusal{
		Headline:    "cue.mod/module.cue declares no `source: {kind: \"self\"}`",
		Evidence:    evidence,
		Consequence: "CUE's module machinery reads `source:` to decide which bytes form the\nartifact; `self` ships the committed directory as-is, with no VCS machinery.",
		Action:      "Declare it:  cue mod edit --source self",
	})
}

// gateCueModAgreement is msg 2: the artifact's declared path and cue.mod's
// module: line must agree. Publishing under either value leaves the other
// wrong, so publish will not choose.
func gateCueModAgreement(p *Plan, a *artifact) {
	if p.DeclaredPath == "" || a.cueModPath == p.DeclaredPath {
		return
	}
	mp := p.identityField("ModulePath")
	p.refuse(Refusal{
		Headline: fmt.Sprintf("%s disagrees with itself about where it lives", p.DeclaredPath),
		Evidence: [][]string{
			{"declared", p.DeclaredPath, mp.Pos},
			{"cue.mod", a.cueModPath, a.cueModPos},
		},
		Consequence: "cue.mod's `module:` line is what CUE resolves imports against — including\nthis module's own identity import. Publishing under either value leaves the\nother wrong.",
		Action:      "Fix whichever is wrong; publish will not choose for you.",
	})
}

// gateDerivation is msg 7 (D12): metadata.modulePath and metadata.version
// must derive from the identity package. core cannot enforce the wiring — a
// developer who replaces `id.Version` with a literal leaves nothing to
// conflict — so publish compares the evaluated values. The identity side uses
// the effective value, so a literal that would disagree after a --version
// fill is caught now.
func gateDerivation(p *Plan, a *artifact) {
	type pair struct {
		metaPath   string
		identity   string // identity field name
		derivation string
	}
	pairs := []pair{
		{"metadata.modulePath", "ModulePath", "modulePath: id.ModulePath"},
		{"metadata.version", "Version", "version: id.Version"},
	}
	for _, pr := range pairs {
		idField := p.identityField(pr.identity)
		if idField == nil || idField.Value == "" {
			continue // nothing concrete to compare against; refused elsewhere
		}
		metaVal := a.root.LookupPath(cue.ParsePath(pr.metaPath))
		if !metaVal.Exists() {
			continue // shape problems are the schema's finding, not this gate's
		}
		metaStr, err := metaVal.String()
		if err != nil || metaStr == idField.Value {
			continue
		}
		metaPos := fieldPos(a.dir, metaVal)
		file := strings.SplitN(metaPos, ":", 2)[0]
		p.refuse(Refusal{
			Headline: fmt.Sprintf("%s states a %s its identity package does not", file, strings.TrimPrefix(pr.metaPath, "metadata.")),
			Evidence: [][]string{
				{pr.metaPath, metaStr, metaPos},
				{"id." + pr.identity, idField.Value, idField.Pos},
			},
			Action: fmt.Sprintf("Derive it:  %s", pr.derivation),
		})
	}
}

// gateTagMajor is msg 6's evaluable half plus #TagRef's binding: the tag is
// constructed from the effective version, so tag == version holds by
// construction, and what remains checkable is that the tag's major names the
// major the path declares (D18).
func gateTagMajor(p *Plan) {
	if p.Tag == "" || p.Major == "" {
		return
	}
	tagMajor := "v" + strings.SplitN(strings.TrimPrefix(p.Tag, "v"), ".", 2)[0]
	if tagMajor == p.Major {
		return
	}
	mp := p.identityField("ModulePath")
	p.refuse(Refusal{
		Headline: "the tag would not name a version within the major this artifact's path declares",
		Evidence: [][]string{
			{"tag", p.Tag, p.TagSource},
			{"path major", p.Major, mp.Pos},
		},
		Consequence: fmt.Sprintf("Published this way, the registry would serve %s tags from a repository\nwhose path promises %s — a pin on the path's major would never resolve it.", tagMajor, p.Major),
		Action:      fmt.Sprintf("Publish a %s.N.N version, or move the path to its next major first.", p.Major),
	})
}

// gateNamespace enforces the owned-domain path shapes (D13): exact inside
// opmodel.dev and community.opmodel.dev, silent everywhere else — a vanity
// domain or a self-hosted registry may use any valid CUE module path.
// opmodel.dev/core is the fixed path outside the kind pattern;
// testing.opmodel.dev is a separate domain and deliberately not caught.
func gateNamespace(p *Plan) {
	repo := p.RegistryRepo
	if repo == "" || !ownedDomain(repo) || repo == "opmodel.dev/core" {
		return
	}
	shape, allowed := firstPartyShape, "opmodel.dev/(modules|catalogs|platforms)/<name>"
	if strings.HasPrefix(repo, "community.opmodel.dev/") {
		shape, allowed = communityShape, "community.opmodel.dev/(m|catalogs|p)/<owner>/<name>"
	}
	if shape.MatchString(repo) {
		return
	}
	mp := p.identityField("ModulePath")
	p.refuse(Refusal{
		Headline: fmt.Sprintf("%s does not fit the namespace this domain publishes", repo),
		Evidence: [][]string{
			{"declared", p.DeclaredPath, mp.Pos},
			{"expected", allowed},
		},
		Consequence: "Inside the domains OPM owns the path shape is exact; fixtures belong under\ntesting.opmodel.dev, and third-party domains are unconstrained.",
		Action:      "Declare a path matching the shape above, or publish under your own domain.",
	})
}

// gateKindSegment: inside the owned domains the kind segment must agree with
// what is being published — one implementation with two entry points means
// nothing else stops `opm module publish` landing where consumers look for
// catalogs. No platforms arm: nothing publishes a platform today (D14).
func gateKindSegment(p *Plan) {
	repo := p.RegistryRepo
	if repo == "" || !ownedDomain(repo) || repo == "opmodel.dev/core" {
		return
	}
	segments := strings.Split(repo, "/")
	if len(segments) < 2 {
		return
	}
	seg := segments[1]
	agrees := false
	switch p.Kind {
	case KindModule:
		agrees = seg == "modules" || seg == "m"
	case KindCatalog:
		agrees = seg == "catalogs"
	}
	if agrees {
		return
	}
	mp := p.identityField("ModulePath")
	p.refuse(Refusal{
		Headline: fmt.Sprintf("a %s cannot publish under the %s/ segment", p.Kind, seg),
		Evidence: [][]string{
			{"kind", string(p.Kind)},
			{"declared", p.DeclaredPath, mp.Pos},
		},
		Consequence: "The kind segment is where consumers look for this artifact kind; publishing\nacross it files the artifact where nothing will find it.",
		Action:      fmt.Sprintf("Declare a path under the %s segment for this kind, or publish with the\nmatching command.", expectedSegment(p.Kind, repo)),
	})
}

func ownedDomain(repo string) bool {
	return strings.HasPrefix(repo, "opmodel.dev/") || strings.HasPrefix(repo, "community.opmodel.dev/")
}

func expectedSegment(kind Kind, repo string) string {
	if kind == KindCatalog {
		return "catalogs/"
	}
	if strings.HasPrefix(repo, "community.opmodel.dev/") {
		return "m/"
	}
	return "modules/"
}

// gatePackageName is D1's module package-name rule: the module's root package
// must bind to its metadata.name, because a consumer's bare import binds the
// package name. The catalog entry point skips it — a catalog's importable
// surface is its subpackages.
func gatePackageName(p *Plan, a *artifact) {
	if p.Kind != KindModule || a.rootPkgName == "" {
		return
	}
	nameVal := a.root.LookupPath(cue.ParsePath("metadata.name"))
	name, err := nameVal.String()
	if err != nil || name == "" || a.rootPkgName == name {
		return
	}
	importPath := p.DeclaredPath
	if importPath == "" {
		importPath = "<module-path>"
	}
	p.refuse(Refusal{
		Headline: fmt.Sprintf("%s's package does not bind to its name", filepath.Base(p.Dir)),
		Evidence: [][]string{
			{"package", a.rootPkgName, a.rootPkgPos},
			{"name", name, fieldPos(a.dir, nameVal) + " — via metadata.name"},
		},
		Consequence: fmt.Sprintf("A consumer's bare `import %q` binds the package\nname; when they differ every consumer must alias the import.", importPath),
		Action:      fmt.Sprintf("Rename the package:  package %s", name),
	})
}

// gateOverride is D6: local overrides are never honored — CUE strips the file
// and the artifact always resolves as published. What is in question is
// whether what was tested matches what is shipped. The condition is the
// file's presence; --skip-override-check waives the gate for modules only and
// changes nothing else. A catalog's divergence propagates into the key space
// of everything built against it, so it has no waiver.
func gateOverride(p *Plan, opts Options) {
	moduleRoot := loader.ModuleRootFrom(p.Dir)
	p.OverridePresent = loader.HasLocalModuleReplacement(moduleRoot)
	if !p.OverridePresent {
		return
	}
	p.Replacements = readReplacements(opts.Context, p.Dir)

	evidence := make([][]string, 0, len(p.Replacements))
	for _, r := range p.Replacements {
		resolves := "(not pinned in cue.mod — the published artifact will not resolve it)"
		if r.PublishedVersion != "" {
			resolves = fmt.Sprintf("(would resolve to %s when published)", r.PublishedVersion)
		}
		evidence = append(evidence, []string{r.Path, "→ " + r.ReplaceWith, resolves})
	}

	if p.Kind == KindCatalog {
		p.refuse(Refusal{
			Headline:    "cue.mod/local-module.cue is present; a catalog cannot be published from a tree configured for local development",
			Evidence:    evidence,
			Consequence: "A catalog's dependency divergence propagates into the key space of every\nmodule built against it, so there is no override for this one.",
			Action:      "Remove the file and re-vet before publishing.",
		})
		return
	}

	if opts.SkipOverrideCheck {
		p.OverrideWaived = true
		return
	}
	p.refuse(Refusal{
		Headline:    "cue.mod/local-module.cue is present; this module was validated against local checkouts, not against what a consumer will resolve",
		Evidence:    evidence,
		Consequence: "The replacements are ignored either way — CUE strips the file and the\nartifact always resolves as published. What is in question is whether what\nyou tested matches what you are shipping.",
		Action:      "Remove the file, or publish anyway with --skip-override-check if the\nsubstitution above is immaterial.",
	})
}

// readReplacements lists local-module.cue's replaceWith entries beside the
// version cue.mod/module.cue pins for the same dependency — the substitution
// the author sees, rather than a rule citation.
func readReplacements(ctx *cue.Context, dir string) []Replacement {
	data, err := os.ReadFile(filepath.Join(dir, "cue.mod", "local-module.cue"))
	if err != nil {
		return nil
	}
	pins := map[string]string{}
	if modData, err := os.ReadFile(filepath.Join(dir, "cue.mod", "module.cue")); err == nil {
		modVal := ctx.CompileBytes(modData)
		if deps := modVal.LookupPath(cue.ParsePath("deps")); deps.Exists() {
			if iter, err := deps.Fields(); err == nil {
				for iter.Next() {
					if v, err := iter.Value().LookupPath(cue.ParsePath("v")).String(); err == nil {
						pins[iter.Selector().Unquoted()] = v
					}
				}
			}
		}
	}

	var out []Replacement
	v := ctx.CompileBytes(data)
	deps := v.LookupPath(cue.ParsePath("deps"))
	if !deps.Exists() {
		return nil
	}
	iter, err := deps.Fields()
	if err != nil {
		return nil
	}
	for iter.Next() {
		depPath := iter.Selector().Unquoted()
		rw := iter.Value().LookupPath(cue.ParsePath("replaceWith"))
		if !rw.Exists() {
			continue
		}
		target, err := rw.String()
		if err != nil {
			target = "<replacement>"
		}
		out = append(out, Replacement{
			Path:             depPath,
			ReplaceWith:      target,
			PublishedVersion: pins[depPath],
		})
	}
	return out
}
