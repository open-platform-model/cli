// Package modref resolves a published OPM module: a major-free module path
// plus an optional version selector become one exact published version
// (0016:D5). Every command that takes a published module resolves through
// it, so the CLI has one grammar and one notion of "newest".
package modref

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"cuelang.org/go/mod/modconfig"
	"cuelang.org/go/mod/modfile"
	"cuelang.org/go/mod/module"
	"golang.org/x/mod/semver"

	"github.com/open-platform-model/cli/internal/publish"
)

// Source is the registry surface resolution needs: the published versions of
// a module path across every major, and one version's module file.
// modconfig's cached registry satisfies it.
type Source interface {
	ModuleVersions(ctx context.Context, path string) ([]string, error)
	ModFile(ctx context.Context, mv module.Version) (*modfile.File, error)
}

// NewSource builds a Source over the CLI's registry mapping. Module files it
// reads land in CUE's own module cache.
func NewSource(registry string) (Source, error) {
	src, err := modconfig.NewRegistry(&modconfig.Config{CUERegistry: registry})
	if err != nil {
		return nil, fmt.Errorf("resolving registry configuration: %w", err)
	}
	return src, nil
}

// Route names where the registry mapping sends path, for messages
// ("ghcr.io/open-platform-model/opmodel.dev/modules/web_app"). It falls back
// to the mapping itself when the mapping cannot be resolved.
func Route(registry, path string) string {
	resolver, err := modconfig.NewResolver(&modconfig.Config{CUERegistry: registry})
	if err != nil {
		return registry
	}
	loc, ok := resolver.ResolveToLocation(path, "")
	if !ok {
		return registry
	}
	return loc.Host + "/" + loc.Repository
}

// RefusalError carries a refusal the user must fix (exit 2) through the
// house refusal funnel.
type RefusalError struct {
	Refusal publish.Refusal
}

func (e *RefusalError) Error() string { return e.Refusal.Headline }

func refuse(r publish.Refusal) error { return &RefusalError{Refusal: r} }

// pathElement is one element of a published module path.
var pathElement = regexp.MustCompile(`^[a-z0-9._-]+$`)

// ParsePath validates a major-free published module path: at least two
// elements, a dot in the first, lowercase letters, digits, ".", "_" and "-"
// only. A major suffix is refused with the --version spelling it meant, a URL
// scheme with the reminder that the registry configuration routes paths.
// Pure, no I/O (0016:D5:R1).
func ParsePath(arg string) (string, error) {
	if strings.Contains(arg, "://") {
		return "", refuse(publish.Refusal{
			Headline:    fmt.Sprintf("%q is a URL; a module is named by its module path", arg),
			Consequence: "The registry configuration (--registry, OPM_REGISTRY, config registry)\nroutes a module path to its registry.",
			Action:      "Name the module:  opmodel.dev/modules/web_app",
		})
	}
	if path, suffix, ok := strings.Cut(arg, "@"); ok {
		return "", refuse(publish.Refusal{
			Headline: "module path must not carry a major",
			Action:   fmt.Sprintf("Use:  %s --version %s", path, suffixSelector(suffix)),
		})
	}
	elems := strings.Split(arg, "/")
	if len(elems) < 2 || !strings.Contains(elems[0], ".") {
		return "", invalidPath(arg, "a module path has a domain first element and at least two elements")
	}
	for _, e := range elems {
		if !pathElement.MatchString(e) {
			return "", invalidPath(arg, fmt.Sprintf("element %q may hold only lowercase letters, digits, '.', '_' and '-'", e))
		}
	}
	return arg, nil
}

func invalidPath(arg, reason string) error {
	return refuse(publish.Refusal{
		Headline: fmt.Sprintf("%q is not a module path: %s", arg, reason),
		Action:   "Name the module:  opmodel.dev/modules/web_app",
	})
}

// suffixSelector spells an @-suffix as the --version value that means the
// same: "v1" stays a major, "v1.0.4" becomes the pin "1.0.4".
func suffixSelector(suffix string) string {
	if majorSelector.MatchString(suffix) {
		return suffix
	}
	if exact := strings.TrimPrefix(suffix, "v"); exactSelector(exact) {
		return exact
	}
	return "<vN | X.Y.Z>"
}

// Selector is a parsed --version value: Major ("v1") floats within that
// major, Exact ("1.0.4") pins that tag, both empty selects the highest
// core-compatible major.
type Selector struct{ Major, Exact string }

var majorSelector = regexp.MustCompile(`^v(0|[1-9]\d*)$`)

// exactSelector reports whether s is a full SemVer X.Y.Z with an optional
// prerelease and no build metadata.
func exactSelector(s string) bool {
	if s == "" || s[0] == 'v' {
		return false
	}
	v := "v" + s
	return semver.IsValid(v) && semver.Canonical(v) == v
}

// ParseSelector parses a --version value (0016:D5:R2). Empty is the
// no-selector case. Pure, no I/O.
func ParseSelector(s string) (Selector, error) {
	switch {
	case s == "":
		return Selector{}, nil
	case majorSelector.MatchString(s):
		return Selector{Major: s}, nil
	case exactSelector(s):
		return Selector{Exact: s}, nil
	}
	return Selector{}, refuse(publish.Refusal{
		Headline: fmt.Sprintf("--version %q must be vN (float) or X.Y.Z (pin), e.g. v1 or 1.0.4", s),
	})
}
