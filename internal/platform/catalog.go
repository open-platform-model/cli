package platform

import (
	"context"
	"fmt"
	"strings"

	"golang.org/x/mod/semver"

	"github.com/open-platform-model/cli/internal/config"
	"github.com/open-platform-model/cli/internal/publish"
)

// RefusalError carries a publish.Refusal through an error return, so a
// catalog-resolution failure prints through the house refusal funnel and
// exits 2 (the shape scaffold.RefusalError already established).
type RefusalError struct {
	Refusal publish.Refusal
}

func (e *RefusalError) Error() string { return e.Refusal.Headline }

// ResolveCatalogVersion returns the bare SemVer a Platform subscription to
// modulePath should pin: the highest published release, or with prerelease
// the highest published prerelease. modulePath carries its major, which
// scopes the listing to that major, so a build from another major can never
// be selected.
//
// Development builds are never candidates (see selectCatalogVersion). A
// listing that answers with nothing selectable is a *RefusalError naming the
// action that would select something; a listing that never completes is a
// *publish.ConnectivityError, because no version was judged.
func ResolveCatalogVersion(ctx context.Context, registry, modulePath string, prerelease bool) (string, error) {
	client, err := publish.NewRegistryClient(registry)
	if err != nil {
		return "", err
	}
	return resolveCatalogVersion(ctx, client, registry, modulePath, prerelease)
}

// versionLister is the one registry capability catalog resolution needs, named
// as an interface so the resolution rules are testable without a registry.
type versionLister interface {
	ModuleVersions(ctx context.Context, modulePath string) ([]string, error)
}

// resolveCatalogVersion is ResolveCatalogVersion over an already-built lister.
func resolveCatalogVersion(ctx context.Context, lister versionLister, registry, modulePath string, prerelease bool) (string, error) {
	versions, err := lister.ModuleVersions(ctx, modulePath)
	if err != nil {
		return "", &publish.ConnectivityError{
			Op:  fmt.Sprintf("listing published versions of %s (registry %s)", modulePath, registry),
			Err: err,
		}
	}

	selected, highest := selectCatalogVersion(versions, prerelease)
	if selected == "" {
		return "", &RefusalError{catalogRefusal(registry, modulePath, highest, prerelease)}
	}
	return strings.TrimPrefix(selected, "v"), nil
}

// selectCatalogVersion picks from published, which is the registry's list of
// v-prefixed tags for one major. It returns the selected tag ("" when nothing
// qualifies) and the highest valid tag seen at all, which the refusal reports
// as evidence.
//
// Release mode takes the highest tag with an empty prerelease. Prerelease mode
// takes the highest tag whose prerelease begins with a non-numeric identifier.
// That predicate is the development-build filter: branch-publish pipelines tag
// `X.Y.Z-0.dev.<count>.g<sha>`, whose leading numeric identifier makes SemVer
// (2.0.0 §11.4.3) rank it below every named prerelease. Reading the property
// the tag scheme was built to have rejects an unknown future shape, where
// matching the literal "dev" would silently accept it.
//
// Ordering is computed rather than assumed: the caller's list arrives sorted,
// but the selection must not depend on that.
func selectCatalogVersion(published []string, prerelease bool) (selected, highest string) {
	for _, v := range published {
		if !semver.IsValid(v) {
			continue
		}
		if highest == "" || semver.Compare(v, highest) > 0 {
			highest = v
		}
		if !qualifies(v, prerelease) {
			continue
		}
		if selected == "" || semver.Compare(v, selected) > 0 {
			selected = v
		}
	}
	return selected, highest
}

// qualifies reports whether v is selectable in the requested mode.
func qualifies(v string, prerelease bool) bool {
	pre := strings.TrimPrefix(semver.Prerelease(v), "-")
	if !prerelease {
		return pre == ""
	}
	if pre == "" {
		return false
	}
	first, _, _ := strings.Cut(pre, ".")
	return !isNumericIdentifier(first)
}

// isNumericIdentifier reports whether a SemVer prerelease identifier is
// numeric, which is what marks a development build.
func isNumericIdentifier(id string) bool {
	if id == "" {
		return false
	}
	for _, r := range id {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// catalogRefusal states what was looked for, where, what was there instead,
// and the one flag that changes the answer.
func catalogRefusal(registry, modulePath, highest string, prerelease bool) publish.Refusal {
	kind := "release"
	action := "Subscribe to the newest prerelease instead:  opm operator install --catalog-prerelease\n" +
		"Or install without seeding a Platform:      opm operator install --skip-platform"
	if prerelease {
		kind = "prerelease"
		action = "Install without seeding a Platform:  opm operator install --skip-platform"
	}

	seen := highest
	if seen == "" {
		seen = "(no published versions)"
	}

	return publish.Refusal{
		Headline: fmt.Sprintf("%s has no published %s", modulePath, kind),
		Evidence: [][]string{
			{"catalog", modulePath},
			{"registry", registry},
			{"highest published", seen},
		},
		Consequence: "A cluster Platform pins exactly one published catalog build. Nothing is\ninferred from a neighboring version, and no development build is ever\nsubscribed to.",
		Action:      action,
	}
}

// DefaultCatalogPath is the catalog `opm operator install` seeds a
// single-catalog Platform with, re-exported here so callers resolving a
// version and callers building a subscription name the same value.
var DefaultCatalogPath = config.DefaultCatalogPath
