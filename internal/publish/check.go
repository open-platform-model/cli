package publish

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"cuelang.org/go/cue"
	"cuelang.org/go/mod/modconfig"
	"cuelang.org/go/mod/module"
	"golang.org/x/mod/semver"
)

// ErrNotPublished reports that the coordinate handed to RegistryCheck names
// no published build — a reachable registry answered, and the answer was no.
var ErrNotPublished = errors.New("not published")

// CheckOptions carries one `opm catalog registry check` invocation's inputs.
type CheckOptions struct {
	// Coordinate is the <path@version> argument — the repository path and the
	// exact published version to fetch ("example.com/catalogs/demo@v1.2.0";
	// the "v" may be omitted).
	Coordinate string

	// Compat additionally runs the predecessor comparison (D9's rule) for
	// the fetched build, exactly as publish would have.
	Compat bool

	// Context is the CUE context every load runs in.
	Context *cue.Context

	// Registry is the resolved CUE registry. Empty inherits the process
	// environment.
	Registry string
}

// CheckReport is what the check resolved: the fetched build's declared
// identity, its member inventory, and every finding. Findings reuse the
// publish Refusal shape — the check is the consumer's side of the same
// contract — but the command frames them as findings: this is an aid, and
// nothing was refused (D35, enforcement exists only at publish).
type CheckReport struct {
	// Repo and Version are the fetched coordinate, split.
	Repo    string
	Version string

	// DeclaredPath and DeclaredVersion are what the pulled catalog's
	// metadata states ("" when not concrete — itself a finding).
	DeclaredPath    string
	DeclaredVersion string

	// Members is the fetched build's member inventory.
	Members []Member

	// Gates carries the compat counts when Compat ran.
	Gates CatalogGateOutcomes

	// CompatRan reports the --compat walk completed.
	CompatRan bool

	// Findings is everything the check surfaced. Empty means clean.
	Findings []Refusal
}

// Clean reports whether the check surfaced nothing.
func (r *CheckReport) Clean() bool { return len(r.Findings) == 0 }

func (r *CheckReport) finding(f Refusal) { r.Findings = append(r.Findings, f) }

// RegistryCheck pulls a published catalog by path@version and verifies, out
// of band, what a consumer verifies at materialize (D7): the declared
// identity is concrete and its modulePath/version agree with the coordinate
// the build was fetched by — read from the pulled catalog's metadata, like
// the library's materialize-time twin; a published catalog's identity package
// is never evaluated as a package by any consumer. It reports the member
// inventory per kind and apiVersion, and with Compat set it runs the
// predecessor comparison for the fetched build.
//
// Transport failures return *ConnectivityError; a coordinate no build
// answers to returns ErrNotPublished.
func RegistryCheck(ctx context.Context, opts CheckOptions) (*CheckReport, error) {
	if opts.Context == nil {
		return nil, fmt.Errorf("registry check: CheckOptions.Context is required")
	}
	repo, version, ok := strings.Cut(opts.Coordinate, "@")
	if !ok || repo == "" || version == "" {
		return nil, fmt.Errorf("registry check: %q is not a <path@version> coordinate", opts.Coordinate)
	}
	if !strings.HasPrefix(version, "v") {
		version = "v" + version
	}
	if !semver.IsValid(version) {
		return nil, fmt.Errorf("registry check: %q is not a semantic version", version)
	}
	report := &CheckReport{Repo: repo, Version: version}
	lopts := Options{Context: opts.Context, Registry: opts.Registry}

	// One fetch through CUE's own module cache hands back the extracted tree;
	// root and member packages load from it exactly as they load from a
	// working tree. (The root cannot be loaded by import path: its package
	// name is the author's, not the path's last segment.)
	extractDir, err := fetchPublishedTree(ctx, opts.Registry, repo, version)
	if err != nil {
		return nil, err
	}

	root, _, _, refusal := loadPackage(lopts, extractDir, ".")
	if refusal != nil {
		return nil, fmt.Errorf("the published build's root package does not load: %w", refusal.Err)
	}
	checkDeclaredIdentity(report, root)

	// The member inventory, by the same walk publish runs over a working
	// tree. A member package that does not load is a finding — the build is
	// published; a consumer imports what it can and hits the rest.
	members, refusals := enumerateMembers(lopts, extractDir)
	report.Members = members
	for _, r := range refusals {
		report.finding(r)
	}

	if opts.Compat {
		client, err := NewRegistryClient(opts.Registry)
		if err != nil {
			return nil, err
		}
		modPath := repo + "@" + semver.Major(version)
		versions, err := client.ModuleVersions(ctx, modPath)
		if err != nil {
			return nil, &ConnectivityError{Op: fmt.Sprintf("listing published versions of %s", modPath), Err: err}
		}
		preds := predecessorVersions(versions, version, semver.Major(version))
		if err := compatScan(lopts, repo, preds, report.Members, &report.Gates, report.finding); err != nil {
			return nil, err
		}
		report.CompatRan = true
	}

	return report, nil
}

// checkDeclaredIdentity is the consumer twin's assertion: the pulled
// catalog's declared modulePath and version are concrete and name the
// coordinate the build was fetched by. Disagreement reports both values —
// which bytes a tag serves is permanent, so a build that misstates its own
// identity misleads every consumer resolving it.
func checkDeclaredIdentity(report *CheckReport, root cue.Value) {
	mp := root.LookupPath(cue.ParsePath("metadata.modulePath"))
	if s, err := mp.String(); err == nil {
		report.DeclaredPath = s
	} else {
		report.finding(Refusal{
			Headline:    "the published catalog does not declare a concrete metadata.modulePath",
			Consequence: "A consumer verifies the declared identity against the coordinate it\nfetched by; there is nothing here to verify.",
		})
	}
	ver := root.LookupPath(cue.ParsePath("metadata.version"))
	if s, err := ver.String(); err == nil {
		report.DeclaredVersion = s
	} else {
		report.finding(Refusal{
			Headline:    "the published catalog does not declare a concrete metadata.version",
			Consequence: "A consumer verifies the declared identity against the coordinate it\nfetched by; there is nothing here to verify.",
		})
	}

	if report.DeclaredPath != "" {
		declaredRepo, declaredMajor, _ := strings.Cut(report.DeclaredPath, "@")
		if declaredRepo != report.Repo || declaredMajor != semver.Major(report.Version) {
			report.finding(Refusal{
				Headline: "the published catalog declares a path other than the coordinate it was fetched by",
				Evidence: [][]string{
					{"declared", report.DeclaredPath},
					{"fetched", report.Repo + "@" + semver.Major(report.Version)},
				},
			})
		}
	}
	if report.DeclaredVersion != "" && "v"+report.DeclaredVersion != report.Version {
		report.finding(Refusal{
			Headline: "the published catalog declares a version other than the tag it was fetched by",
			Evidence: [][]string{
				{"declared", report.DeclaredVersion},
				{"fetched", strings.TrimPrefix(report.Version, "v")},
			},
		})
	}
}

// fetchPublishedTree fetches a published build through CUE's own module
// machinery — the same on-disk cache every package load hits, so the build is
// fetched once — and returns the extracted tree's directory. A build the
// registry answers "no" to is ErrNotPublished; a registry that cannot answer
// is a *ConnectivityError.
func fetchPublishedTree(ctx context.Context, registry, repo, version string) (string, error) {
	reg, err := modconfig.NewRegistry(&modconfig.Config{Env: registryEnv(registry)})
	if err != nil {
		return "", fmt.Errorf("building module registry resolver: %w", err)
	}
	mv, err := module.NewVersion(repo+"@"+semver.Major(version), version)
	if err != nil {
		return "", fmt.Errorf("forming module version from %s and %s: %w", repo, version, err)
	}
	loc, err := reg.Fetch(ctx, mv)
	if err != nil {
		// Measured: an absent module version reports "module not found";
		// transport failures report the failed request.
		if strings.Contains(err.Error(), "not found") {
			return "", fmt.Errorf("%s@%s: %w", repo, version, ErrNotPublished)
		}
		return "", &ConnectivityError{Op: fmt.Sprintf("fetching %s", mv), Err: err}
	}
	osfs, ok := loc.FS.(module.OSRootFS)
	if !ok {
		return "", fmt.Errorf("fetched module source for %s has no filesystem root (%T)", mv, loc.FS)
	}
	return filepath.Join(osfs.OSRoot(), loc.Dir), nil
}

// Render prints the report: coordinate, declared identity, the member
// inventory per package, the compat outcome when it ran, and the verdict.
func (r *CheckReport) Render() string {
	var b strings.Builder
	row := func(label, value string) {
		if value == "" {
			value = "—"
		}
		fmt.Fprintf(&b, "  %-15s %s\n", label, value)
	}
	row("coordinate", r.Repo+"@"+r.Version)
	row("declaredPath", r.DeclaredPath)
	row("declaredVersion", r.DeclaredVersion)

	byPkg := map[string][]string{}
	var pkgs []string
	for _, m := range r.Members {
		if _, seen := byPkg[m.PkgPath]; !seen {
			pkgs = append(pkgs, m.PkgPath)
		}
		byPkg[m.PkgPath] = append(byPkg[m.PkgPath], m.Name)
	}
	sort.Strings(pkgs)
	fmt.Fprintf(&b, "  %-15s %d\n", "members", len(r.Members))
	for _, p := range pkgs {
		names := byPkg[p]
		sort.Strings(names)
		fmt.Fprintf(&b, "    %-16s %s\n", p, strings.Join(names, ", "))
	}
	if r.CompatRan {
		g := r.Gates
		row("compat", fmt.Sprintf("%d compared, %d violating, %d alpha-exempt, %d new",
			g.CompatCompared, g.CompatRefused, g.CompatAlpha, g.CompatNew))
	}

	if r.Clean() {
		fmt.Fprintf(&b, "\n  CLEAN — the declared identity matches the fetched coordinate\n")
	} else {
		noun := "finding"
		if len(r.Findings) != 1 {
			noun = "findings"
		}
		fmt.Fprintf(&b, "\n  %d %s\n", len(r.Findings), noun)
	}
	return b.String()
}
