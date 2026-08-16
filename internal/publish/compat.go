package publish

import (
	"fmt"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"cuelang.org/go/cue"
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

	byPkg := map[string][]Member{}
	var pkgs []string
	for _, m := range p.members {
		if m.Kind == "transformers" {
			continue
		}
		_, lvl, ok := compat.ParseLevel(m.APIVersion)
		if !ok {
			continue // no classifiable apiVersion — the FQN gate's finding, not this one's
		}
		if !lvl.Enforced() {
			p.CatalogGates.CompatAlpha++
			continue
		}
		if _, seen := byPkg[m.PkgPath]; !seen {
			pkgs = append(pkgs, m.PkgPath)
		}
		byPkg[m.PkgPath] = append(byPkg[m.PkgPath], m)
	}
	sort.Strings(pkgs)

	versions := predecessorVersions(p.publishedVersions, p.Tag, p.Major)

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
			pkgVal, found, err := loadPublishedPackage(opts, loadDir, p.RegistryRepo+"/"+pkgPath, v)
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
				refusal, err := compareMember(m, prev, p.RegistryRepo, v)
				if err != nil {
					return err
				}
				p.CatalogGates.CompatCompared++
				if refusal != nil {
					p.CatalogGates.CompatRefused++
					p.refuse(*refusal)
				}
			}
			unresolved = still
		}
		// History exhausted: whatever remains is new at its key — the
		// apiVersion-bump escape.
		p.CatalogGates.CompatNew += len(unresolved)
	}

	p.CompatChecked = true
	return nil
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

// compareMember strips provenance from both operands (D30 — catalogVersion
// differs between any two builds by construction) and runs the level-aware
// comparator with the member's own apiVersion. Violations render as refusal
// 9; a comparator failure (unclassifiable input) is an error, never a
// verdict.
func compareMember(next, prev Member, repo, predVersion string) (*Refusal, error) {
	prevStripped, err := compat.StripProvenance(prev.Value)
	if err != nil {
		return nil, fmt.Errorf("preparing %s@%s from %s for comparison: %w", next.Name, next.APIVersion, predVersion, err)
	}
	nextStripped, err := compat.StripProvenance(next.Value)
	if err != nil {
		return nil, fmt.Errorf("preparing %s@%s for comparison: %w", next.Name, next.APIVersion, err)
	}
	violations, err := compat.CheckAtLevel(next.APIVersion, prevStripped, nextStripped)
	if err != nil {
		return nil, fmt.Errorf("comparing %s@%s against %s: %w", next.Name, next.APIVersion, predVersion, err)
	}
	if len(violations) == 0 {
		return nil, nil
	}
	r := compatRefusal(next, repo, predVersion, violations)
	return &r, nil
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
var apiVersionParts = regexp.MustCompile(`^v([0-9]+)(alpha|beta)?([0-9]+)?$`)

// nextAPIVersion suggests the apiVersion the closing action names: the next
// counter within the level (v1beta1 -> v1beta2), or the next major for GA
// (v1 -> v2).
func nextAPIVersion(apiVersion string) string {
	parts := apiVersionParts.FindStringSubmatch(apiVersion)
	if parts == nil {
		return "<next-apiVersion>"
	}
	major, _ := strconv.Atoi(parts[1])
	if parts[2] == "" {
		return fmt.Sprintf("v%d", major+1)
	}
	n, _ := strconv.Atoi(parts[3])
	return fmt.Sprintf("v%d%s%d", major, parts[2], n+1)
}
