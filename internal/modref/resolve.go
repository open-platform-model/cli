package modref

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"cuelang.org/go/mod/module"
	"golang.org/x/mod/semver"

	"github.com/open-platform-model/cli/internal/publish"
)

// corePath is the module path, without major, whose major a published module
// must share with this CLI to be selected by the no-selector walk.
const corePath = "opmodel.dev/core"

// Strategy is how a Resolution chose its version.
type Strategy int

const (
	// Exact pinned the tag the selector named.
	Exact Strategy = iota
	// FloatMajor took the newest release within the named major.
	FloatMajor
	// HighestCompatibleMajor walked the majors from the highest down to the
	// first one on the CLI's core major.
	HighestCompatibleMajor
)

// Skipped is a major the no-selector walk passed over, with why.
type Skipped struct{ Major, Reason string }

// Resolution is one published module version chosen by Resolve.
type Resolution struct {
	Path      string // "opmodel.dev/modules/web_app"
	Major     string // "v1"
	Version   string // "v1.0.4"
	Strategy  Strategy
	CoreMajor string    // the CLI's core major, "v2"
	Skipped   []Skipped // majors above Major the walk passed over, highest first
}

// Import returns the major-qualified module path ("…/web_app@v1").
func (r *Resolution) Import() string { return r.Path + "@" + r.Major }

// Request is what Resolve resolves: a path from ParsePath, a selector from
// ParseSelector, the CLI's core major, and the registry to name in messages.
type Request struct {
	Path      string
	Selector  Selector
	CoreMajor string
	Registry  string
}

// Resolve lists the module's published versions across every major and picks
// one by the selector (0016:D5:R2/R3). A transport failure is a
// *publish.ConnectivityError; anything the user must fix is a *RefusalError.
func Resolve(ctx context.Context, src Source, req Request) (*Resolution, error) {
	versions, err := src.ModuleVersions(ctx, req.Path)
	if err != nil {
		return nil, &publish.ConnectivityError{
			Op:  fmt.Sprintf("listing published versions of %s (registry %s)", req.Path, req.Registry),
			Err: err,
		}
	}
	if len(versions) == 0 {
		return nil, refuse(publish.Refusal{
			Headline: fmt.Sprintf("%s has no published versions", req.Path),
			Evidence: [][]string{{"registry", req.Registry}},
			Action:   "Check the module path, or the registry configuration that routes it.",
		})
	}
	byMajor := groupByMajor(versions)

	switch {
	case req.Selector.Exact != "":
		return resolveExact(req, versions)
	case req.Selector.Major != "":
		return resolveFloat(req, byMajor)
	}
	return resolveHighestCompatible(ctx, src, req, byMajor)
}

func resolveExact(req Request, versions []string) (*Resolution, error) {
	want := "v" + req.Selector.Exact
	if !slices.Contains(versions, want) {
		return nil, refuse(publish.Refusal{
			Headline: fmt.Sprintf("%s has no published version %s", req.Path, req.Selector.Exact),
			Evidence: [][]string{{"registry", req.Registry}, {"published", displayVersions(versions)}},
			Action:   "Pin a published version, or float within a major with --version vN.",
		})
	}
	return &Resolution{
		Path:      req.Path,
		Major:     semver.Major(want),
		Version:   want,
		Strategy:  Exact,
		CoreMajor: req.CoreMajor,
	}, nil
}

func resolveFloat(req Request, byMajor map[string][]string) (*Resolution, error) {
	major := req.Selector.Major
	inMajor := byMajor[major]
	if len(inMajor) == 0 {
		return nil, refuse(publish.Refusal{
			Headline: fmt.Sprintf("%s has no published version in major %s", req.Path, major),
			Evidence: [][]string{{"registry", req.Registry}, {"majors", strings.Join(majorsDescending(byMajor), ", ")}},
			Action:   "Float within a published major with --version vN.",
		})
	}
	v := Newest(inMajor)
	if v == "" {
		return nil, refuse(publish.Refusal{
			Headline: fmt.Sprintf("major %s of %s holds only development builds", major, req.Path),
			Evidence: [][]string{{"registry", req.Registry}, {"published", displayVersions(inMajor)}},
			Action:   "A float never selects a development build; pin one with --version X.Y.Z.",
		})
	}
	return &Resolution{
		Path:      req.Path,
		Major:     major,
		Version:   v,
		Strategy:  FloatMajor,
		CoreMajor: req.CoreMajor,
	}, nil
}

func resolveHighestCompatible(ctx context.Context, src Source, req Request, byMajor map[string][]string) (*Resolution, error) {
	var skipped []Skipped
	for _, major := range majorsDescending(byMajor) {
		v := Newest(byMajor[major])
		if v == "" {
			skipped = append(skipped, Skipped{Major: major, Reason: "holds only development builds"})
			continue
		}
		coreMajor, err := coreMajorOf(ctx, src, req, major, v)
		if err != nil {
			return nil, err
		}
		switch coreMajor {
		case "":
			skipped = append(skipped, Skipped{Major: major, Reason: "declares no " + corePath + " dependency"})
			continue
		case req.CoreMajor:
			return &Resolution{
				Path:      req.Path,
				Major:     major,
				Version:   v,
				Strategy:  HighestCompatibleMajor,
				CoreMajor: req.CoreMajor,
				Skipped:   skipped,
			}, nil
		}
		skipped = append(skipped, Skipped{Major: major, Reason: "requires core " + coreMajor})
	}
	evidence := [][]string{{"registry", req.Registry}}
	for _, s := range skipped {
		evidence = append(evidence, []string{s.Major, s.Reason})
	}
	return nil, refuse(publish.Refusal{
		Headline: fmt.Sprintf("no published major of %s builds on core %s", req.Path, req.CoreMajor),
		Evidence: evidence,
		Action:   "Pin a version with --version, or use an opm release on the core line the module requires.",
	})
}

// coreMajorOf reads one release's module file and returns the major of its
// opmodel.dev/core dependency, "" when it declares none.
func coreMajorOf(ctx context.Context, src Source, req Request, major, version string) (string, error) {
	mv, err := module.NewVersion(req.Path+"@"+major, version)
	if err != nil {
		return "", fmt.Errorf("forming module version %s@%s: %w", req.Path, version, err)
	}
	mf, err := src.ModFile(ctx, mv)
	if err != nil {
		return "", &publish.ConnectivityError{
			Op:  fmt.Sprintf("reading the module file of %s (registry %s)", mv, req.Registry),
			Err: err,
		}
	}
	for dep := range mf.Deps {
		if p, m, ok := strings.Cut(dep, "@"); ok && p == corePath {
			return m, nil
		}
	}
	return "", nil
}

// Newest is the float rule (0016:D5:R2): the newest stable version, else the
// newest prerelease that is not a development build. A float never selects a
// development build; "" when versions holds nothing else. versions is
// SemVer-sorted ascending, as the registry lists them.
func Newest(versions []string) string {
	var pre string
	for i := len(versions) - 1; i >= 0; i-- {
		v := versions[i]
		switch {
		case semver.Prerelease(v) == "":
			return v
		case pre == "" && !publish.IsDevTag(v):
			pre = v
		}
	}
	return pre
}

// groupByMajor splits the registry's ascending version list by major,
// keeping each major's list ascending.
func groupByMajor(versions []string) map[string][]string {
	out := map[string][]string{}
	for _, v := range versions {
		if semver.IsValid(v) {
			out[semver.Major(v)] = append(out[semver.Major(v)], v)
		}
	}
	return out
}

// majorsDescending returns the majors highest first.
func majorsDescending(byMajor map[string][]string) []string {
	majors := make([]string, 0, len(byMajor))
	for m := range byMajor {
		majors = append(majors, m)
	}
	slices.SortFunc(majors, func(a, b string) int { return semver.Compare(b, a) })
	return majors
}

// displayVersions renders tags without their "v", as a user pins them.
func displayVersions(versions []string) string {
	out := make([]string, len(versions))
	for i, v := range versions {
		out[i] = strings.TrimPrefix(v, "v")
	}
	return strings.Join(out, ", ")
}
