// Package k8sgen builds Kubernetes resource-definition manifests from
// loaded OPM modules: Kubernetes CustomResourceDefinition (CRD) and
// Crossplane v2 CompositeResourceDefinition (XRD). The module's #config
// definition drives the emitted OpenAPI v3 schema; surrounding metadata
// (group, version, names) is derived from the module's metadata.name and
// metadata.version plus caller-supplied options.
package k8sgen

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

// Names holds the Kubernetes CRD name fields derived from a module's
// metadata.name. The derivation is intentionally naive (see DeriveNames) and
// is expected to evolve; callers that need overrides (irregular plurals,
// acronym preservation) should expose flags on top of this.
type Names struct {
	Kind     string
	ListKind string
	Singular string
	Plural   string
}

// kindPattern is the validation regex for a CRD kind. Kubernetes accepts a
// stricter DNS-like pattern for names, but kind specifically must match
// ^[A-Z][a-zA-Z0-9]*$.
var kindPattern = regexp.MustCompile(`^[A-Z][a-zA-Z0-9]*$`)

// DeriveNames computes CRD name fields from an OPM module's metadata.name.
//
// Rules (POC — will change when we handle irregular cases):
//   - Split on '-' and '_'; consecutive separators and leading/trailing
//     separators are collapsed.
//   - PascalCase each segment: first rune upper, remaining runes lower.
//   - Concatenate segments to form Kind.
//   - ListKind = Kind + "List".
//   - Singular = strings.ToLower(Kind).
//   - Plural   = Singular + "s" (naive; does not handle -y/-s/-ch/-sh).
//
// Returns an error if the resulting Kind does not match ^[A-Z][a-zA-Z0-9]*$
// (e.g., the module name starts with a digit or contains characters other
// than ASCII letters, digits, '-', '_').
func DeriveNames(moduleName string) (Names, error) {
	trimmed := strings.TrimSpace(moduleName)
	if trimmed == "" {
		return Names{}, fmt.Errorf("module name is empty")
	}

	segments := strings.FieldsFunc(trimmed, func(r rune) bool {
		return r == '-' || r == '_'
	})
	if len(segments) == 0 {
		return Names{}, fmt.Errorf("module name %q contains only separators", moduleName)
	}

	var b strings.Builder
	for _, seg := range segments {
		runes := []rune(seg)
		b.WriteRune(unicode.ToUpper(runes[0]))
		for _, r := range runes[1:] {
			b.WriteRune(unicode.ToLower(r))
		}
	}
	kind := b.String()

	if !kindPattern.MatchString(kind) {
		return Names{}, fmt.Errorf(
			"module name %q produces invalid CRD kind %q (must match %s)",
			moduleName, kind, kindPattern,
		)
	}

	singular := strings.ToLower(kind)
	return Names{
		Kind:     kind,
		ListKind: kind + "List",
		Singular: singular,
		Plural:   singular + "s",
	}, nil
}

// DeriveVersion maps an OPM module's metadata.version to a CRD version
// string.
//
// POC rule (will change — single mapping is a starting point):
//   - An optional leading "v" is tolerated.
//   - SemVer prerelease ("-...") and build metadata ("+...") are stripped
//     before inspecting the major component. No alpha/beta distinction is
//     produced for 1.x+ prereleases: "1.0.0-beta" maps to "v1".
//   - Major == 0 → "v1alpha1".
//   - Major >= 1 → "v" + major   (e.g. "2.3.1" → "v2").
//   - Empty or unparseable major → error.
func DeriveVersion(moduleVersion string) (string, error) {
	v := strings.TrimSpace(moduleVersion)
	v = strings.TrimPrefix(v, "v")
	if v == "" {
		return "", fmt.Errorf("module version is empty")
	}

	if i := strings.IndexAny(v, "-+"); i != -1 {
		v = v[:i]
	}
	if v == "" {
		return "", fmt.Errorf("module version %q has no major component", moduleVersion)
	}

	majorStr, _, _ := strings.Cut(v, ".")
	if majorStr == "" {
		return "", fmt.Errorf("module version %q has no major component", moduleVersion)
	}

	major, err := strconv.Atoi(majorStr)
	if err != nil {
		return "", fmt.Errorf("module version %q has non-numeric major: %w", moduleVersion, err)
	}
	if major < 0 {
		return "", fmt.Errorf("module version %q has negative major", moduleVersion)
	}

	if major == 0 {
		return "v1alpha1", nil
	}
	return fmt.Sprintf("v%d", major), nil
}
