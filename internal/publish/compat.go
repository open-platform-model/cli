package publish

import (
	"bytes"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/ast"
	"cuelang.org/go/cue/format"
	"cuelang.org/go/cue/load"
	"golang.org/x/mod/semver"

	"github.com/open-platform-model/library/opm/compat"

	"github.com/open-platform-model/cli/internal/output"
)

// gateCompat is D9's compatibility gate: every beta/GA member of the tree is
// compared against the last published build that shipped a member of that
// name at that apiVersion, under 0010 D27's additive-only rule, via the
// library comparator. The predecessor is found by D9's literal rule — scan
// published versions strictly below the effective version, same major,
// prereleases included, newest first — NOT by a stable-preferring selector: a
// beta+ name-and-apiVersion key is a permanent claim on its own history, so
// an incompatible re-introduction after a removal is refused identically, and
// the only escape is the apiVersion bump (a member no build has carried
// passes).
//
// Transformers are excluded structurally (no apiVersion, D44) and alpha
// members by policy (D34), both before any registry work. Any transport
// failure during the walk aborts as a *ConnectivityError with no partial
// verdict — the artifact was never judged; a package absent at a given
// version is the scan's negative signal, never an error.
func gateCompat(p *Plan, opts Options) error {
	if p.Kind != KindCatalog || !p.RegistryChecked {
		return nil // never judged: the lookup's ConnectivityError already reported it
	}
	versions := predecessorVersions(p.publishedVersions, p.Tag, p.Major)
	if err := compatScan(opts, p.RegistryRepo, versions, p.members, &p.CatalogGates, p.refuse); err != nil {
		return err
	}
	p.CompatChecked = true
	return nil
}

// compatScan is the walk itself, shared by the publish gate and
// `opm catalog registry check --compat` (D7): partition the members
// (transformers structurally, alpha by policy — both before any registry
// work), group by package, and probe the given versions newest-first until
// every member found its predecessor or history is exhausted. Counts land in
// g; violations go through refuse.
func compatScan(opts Options, repo string, versions []string, members []Member, g *CatalogGateOutcomes, refuse func(Refusal)) error {
	byPkg, pkgs := eligibleByPackage(members, g)

	// The loads run from a neutral directory: a versioned package pattern
	// resolves against the registry, and running it inside the artifact's own
	// module would put the current tree's cue.mod in play.
	loadDir, err := os.MkdirTemp("", "opm-compat-*")
	if err != nil {
		return fmt.Errorf("creating predecessor-scan work directory: %w", err)
	}
	defer os.RemoveAll(loadDir)

	for _, pkgPath := range pkgs {
		unresolved := byPkg[pkgPath]
		for _, v := range versions {
			if len(unresolved) == 0 {
				break
			}
			pkgVal, found, err := loadPublishedPackage(opts, loadDir, repo+"/"+pkgPath, v)
			if err != nil {
				return err
			}
			if !found {
				continue // absent at this build — the scan's negative signal
			}
			published := membersOfPackage(pkgVal, pkgPath)
			var still []Member
			for _, m := range unresolved {
				prev, ok := findMember(published, m.Name, m.APIVersion)
				if !ok {
					still = append(still, m)
					continue
				}
				refusal, err := compareMember(m, prev, repo, v)
				if err != nil {
					return err
				}
				g.CompatCompared++
				if refusal != nil {
					g.CompatRefused++
					refuse(*refusal)
				}
			}
			unresolved = still
		}
		// History exhausted: whatever remains is new at its key — the
		// apiVersion-bump escape.
		g.CompatNew += len(unresolved)
	}
	return nil
}

// eligibleByPackage partitions the members the compat gate binds to
// (transformers excluded structurally, alpha members by policy, unclassifiable
// apiVersions left to the FQN gate) and groups them by package, keys sorted
// for deterministic walk order.
func eligibleByPackage(members []Member, g *CatalogGateOutcomes) (byPkg map[string][]Member, pkgs []string) {
	byPkg = map[string][]Member{}
	for _, m := range members {
		if m.Kind == "transformers" {
			continue
		}
		_, lvl, ok := compat.ParseLevel(m.APIVersion)
		if !ok {
			continue // no classifiable apiVersion — the FQN gate's finding, not this one's
		}
		if !lvl.Enforced() {
			g.CompatAlpha++
			continue
		}
		if _, seen := byPkg[m.PkgPath]; !seen {
			pkgs = append(pkgs, m.PkgPath)
		}
		byPkg[m.PkgPath] = append(byPkg[m.PkgPath], m)
	}
	sort.Strings(pkgs)
	return byPkg, pkgs
}

// predecessorVersions filters the published tags to the scan's window —
// strictly below the effective tag, within the declared major, prereleases
// included — ordered newest first, so each member resolves against the newest
// build carrying it.
func predecessorVersions(published []string, tag, major string) []string {
	var out []string
	for _, v := range published {
		if !semver.IsValid(v) || semver.Major(v) != major {
			continue
		}
		if tag != "" && semver.Compare(v, tag) >= 0 {
			continue
		}
		out = append(out, v)
	}
	semver.Sort(out)
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out
}

// loadPublishedPackage loads one published subpackage by
// <importPath>@<version> — measured loadable standalone through
// load.Instances with only a registry env; the module zip is fetched once per
// build and CUE-cached on disk, so probing several packages of one build
// costs one fetch. Returns found=false on the loader's module-not-found
// error (the package or the whole build is absent at that version — both are
// the scan's negative signal); any other failure is a *ConnectivityError.
func loadPublishedPackage(opts Options, dir, importPath, version string) (cue.Value, bool, error) {
	pattern := importPath + "@" + version
	cfg := &load.Config{
		Dir: dir,
		Env: registryEnv(opts.Registry),
	}
	insts := load.Instances([]string{pattern}, cfg)
	if len(insts) == 0 {
		return cue.Value{}, false, &ConnectivityError{Op: "loading " + pattern, Err: fmt.Errorf("no instance returned")}
	}
	if err := insts[0].Err; err != nil {
		// Measured against CUE v0.17: an absent module version and an absent
		// package within an existing version both report "cannot find module
		// providing package"; transport failures report "cannot fetch".
		if strings.Contains(err.Error(), "cannot find module providing package") {
			return cue.Value{}, false, nil
		}
		return cue.Value{}, false, &ConnectivityError{Op: "loading " + pattern, Err: err}
	}
	v := opts.Context.BuildInstance(insts[0])
	if err := v.Err(); err != nil {
		return cue.Value{}, false, fmt.Errorf("evaluating published package %s: %w", pattern, err)
	}
	return v, true, nil
}

// findMember locates the member with the given name and apiVersion — D9's
// key — in a published package's member list.
func findMember(members []Member, name, apiVersion string) (Member, bool) {
	for _, m := range members {
		if m.Name == name && m.APIVersion == apiVersion {
			return m, true
		}
	}
	return Member{}, false
}

// provenancePaths is 0010 D30's denylist — the two metadata fields that
// change per catalog release by construction (D25) and are therefore exempt
// from the comparison. Direct children of metadata only, exactly the scope
// the library's StripProvenance names.
var provenancePaths = map[string]bool{
	"metadata.catalogVersion": true,
	"metadata.description":    true,
}

// compareMember runs the level-aware comparator with the member's own
// apiVersion and applies D30's provenance exemption by dropping violations at
// the two denylisted paths. Violations render as refusal 9; a comparator
// failure (unclassifiable input) is an error, never a verdict.
//
// The D30 exemption is applied as a violation filter rather than through the
// library's StripProvenance: measured against the real catalog, the strip's
// InlineImports round-trip cannot rebuild a member typed against core — the
// emitted syntax references core's hidden #KebabToPascal helper (via
// metadata.#definitionName) without inlining it, and the rebuild fails. The
// filter is the same rule at the same scope, applied to the walk's output
// instead of its input. Recorded for the library follow-up.
func compareMember(next, prev Member, repo, predVersion string) (*Refusal, error) {
	// Fast path, and a correct rule in its own right: a member whose emitted
	// syntax is identical modulo the provenance fields cannot have changed
	// shape — it is the same definition restamped by a release. Measured
	// necessity: on byte-identical operands the walk's leaf subsume still
	// reports "domain narrowed" wherever the schema carries a matchN
	// validator or a pending comprehension (#Image's if-guards), so without
	// this every unchanged real member false-positives. Recorded for the
	// library follow-up.
	if equalModuloProvenance(prev.Value, next.Value) {
		return nil, nil
	}
	violations, err := compat.CheckAtLevel(next.APIVersion, prev.Value, next.Value)
	if err != nil {
		return nil, fmt.Errorf("comparing %s@%s against %s: %w", next.Name, next.APIVersion, predVersion, err)
	}
	kept := violations[:0]
	for _, v := range violations {
		if !provenancePaths[v.Path] {
			kept = append(kept, v)
		}
	}
	if len(kept) == 0 {
		return nil, nil
	}
	r := compatRefusal(next, repo, predVersion, kept)
	return &r, nil
}

// equalModuloProvenance reports whether two member values emit identical
// syntax once the D30 provenance fields are scrubbed. Both operands come
// from the same source conventions in the same cue.Context, so identical
// definitions format identically; provenance is scrubbed on the AST — the
// scrub half of the library strip, without the rebuild that fails on
// core-typed members.
func equalModuloProvenance(a, b cue.Value) bool {
	sa, okA := scrubbedSyntax(a)
	sb, okB := scrubbedSyntax(b)
	return okA && okB && bytes.Equal(sa, sb)
}

// scrubbedSyntax renders a value's full syntax with metadata.catalogVersion
// and metadata.description removed.
func scrubbedSyntax(v cue.Value) ([]byte, bool) {
	syn := v.Syntax(cue.All())
	scrubProvenanceNode(syn)
	out, err := format.Node(syn)
	if err != nil {
		return nil, false
	}
	return out, true
}

// scrubProvenanceNode walks the emitted syntax and removes the denylisted
// direct children from the value of every field named metadata — the same
// scope as the library strip: metadata.catalogVersion and
// metadata.description, never a deeper occurrence.
func scrubProvenanceNode(n ast.Node) {
	switch x := n.(type) {
	case *ast.File:
		for _, d := range x.Decls {
			scrubProvenanceNode(d)
		}
	case *ast.StructLit:
		for _, e := range x.Elts {
			scrubProvenanceNode(e)
		}
	case *ast.Field:
		if name, _, err := ast.LabelName(x.Label); err == nil && name == "metadata" {
			scrubProvenanceValue(x.Value)
		}
		scrubProvenanceNode(x.Value)
	case *ast.EmbedDecl:
		scrubProvenanceNode(x.Expr)
	case *ast.BinaryExpr:
		scrubProvenanceNode(x.X)
		scrubProvenanceNode(x.Y)
	case *ast.ParenExpr:
		scrubProvenanceNode(x.X)
	case *ast.ListLit:
		for _, e := range x.Elts {
			scrubProvenanceNode(e)
		}
	}
}

// scrubProvenanceValue deletes the denylisted fields from every struct
// literal contributing to a metadata value.
func scrubProvenanceValue(e ast.Expr) {
	switch x := e.(type) {
	case *ast.StructLit:
		kept := x.Elts[:0]
		for _, el := range x.Elts {
			if f, ok := el.(*ast.Field); ok {
				if name, _, err := ast.LabelName(f.Label); err == nil && provenanceFields[name] {
					continue
				}
			}
			kept = append(kept, el)
		}
		x.Elts = kept
	case *ast.BinaryExpr:
		scrubProvenanceValue(x.X)
		scrubProvenanceValue(x.Y)
	case *ast.ParenExpr:
		scrubProvenanceValue(x.X)
	}
}

// provenanceFields is the same denylist as provenancePaths, keyed by bare
// field name for the AST scrub.
var provenanceFields = map[string]bool{
	"catalogVersion": true,
	"description":    true,
}

// compatRefusal is refusal 9 (0011 06-operational.md, msg 9): the
// caller-attached header names the member, its apiVersion, and the
// compared-against coordinate; the violation lines are path-located by the
// comparator's walk; the closing action is fixed.
func compatRefusal(m Member, repo, predVersion string, violations []compat.Violation) Refusal {
	pred := strings.TrimPrefix(predVersion, "v")

	rows := make([][]string, 0, len(violations))
	for _, v := range violations {
		path := v.Path
		if path == "" {
			path = "(root)"
		}
		rows = append(rows, []string{path, violationDetail(v)})
	}

	consequence := output.AlignColumns("  ", rows) + "\n\n" +
		fmt.Sprintf("At %s and above a contract may gain fields and options, never lose them,\nand a default may not move. Modules compiled against %s match on this key.", m.APIVersion, pred)

	return Refusal{
		Headline:    fmt.Sprintf("%s would break a contract it already published", repo),
		Evidence:    [][]string{{m.DefName, m.APIVersion, "compared against " + repo + "@" + pred}},
		Consequence: consequence,
		Action:      fmt.Sprintf("Make the change additive, or ship it alongside at a new apiVersion (%s).", nextAPIVersion(m.APIVersion)),
	}
}

// violationDetail renders one violation's kind with its values: defaults show
// old -> new, a narrowed domain carries CUE's subsumption diagnostic
// verbatim, and removals stand on the path alone.
func violationDetail(v compat.Violation) string {
	switch {
	case v.Old != "" && v.New != "":
		return fmt.Sprintf("%s (%s -> %s)", v.Kind, v.Old, v.New)
	case v.New != "":
		return fmt.Sprintf("%s (%s)", v.Kind, v.New)
	case v.Old != "":
		return fmt.Sprintf("%s (was %s)", v.Kind, v.Old)
	default:
		return v.Kind
	}
}

// apiVersionParts splits a #APIVersionType value into its major and optional
// prerelease level counter.
var apiVersionParts = regexp.MustCompile(`^v(\d+)(alpha|beta)?(\d+)?$`)

// nextAPIVersionUnknown is the action's placeholder when the apiVersion does
// not parse — the FQN gate owns that finding.
const nextAPIVersionUnknown = "<next-apiVersion>"

// nextAPIVersion suggests the apiVersion the closing action names: the next
// counter within the level (v1beta1 -> v1beta2), or the next major for GA
// (v1 -> v2).
func nextAPIVersion(apiVersion string) string {
	parts := apiVersionParts.FindStringSubmatch(apiVersion)
	if parts == nil {
		return nextAPIVersionUnknown
	}
	major, err := strconv.Atoi(parts[1])
	if err != nil {
		return nextAPIVersionUnknown
	}
	if parts[2] == "" {
		return fmt.Sprintf("v%d", major+1)
	}
	n, err := strconv.Atoi(parts[3])
	if err != nil {
		return nextAPIVersionUnknown
	}
	return fmt.Sprintf("v%d%s%d", major, parts[2], n+1)
}
