// Package scaffold implements `opm mod init`'s fetch-based scaffolding: the
// template-reference grammar with its docker-official-image-style shortcuts,
// the baked official-template table that drives both shortcut expansion and
// `opm module template list`, version resolution over the published history,
// the staged-tree copy, and the re-identification pass that rewrites a cloned
// template to the user's identity (enhancement 0011, D20/D25 and the
// cli-template-modules change).
package scaffold

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/open-platform-model/cli/internal/cueedit"
)

// Segment is the reserved registry segment shortcut expansion targets
// (0011 D25). Reserved and gate-curated: only the cli's own release CI
// publishes under it, which is what makes expanding a bare word into it safe
// rather than a typosquat surface.
const Segment = "opmodel.dev/templates"

// DefaultTemplate is the template used when an invocation names none.
const DefaultTemplate = "standard"

// OfficialTemplate is one row of the baked table. The table is the single
// source for shortcut expansion and `opm module template list`: the binary
// that knows it belongs to the release train that published the templates, so
// the two cannot drift (release-coupled by construction).
type OfficialTemplate struct {
	Name         string
	Description  string
	DefaultMajor string
}

// Official is the baked table, in listing order.
var Official = []OfficialTemplate{
	{"minimal", "Smallest useful module - one stateless workload, single file", "v1"},
	{"standard", "Default starting point - exposed stateless workload, separated concerns", "v1"},
	{"advanced", "Showcase - web, api, worker, and a stateful cache with trait attachments", "v1"},
}

// officialByName returns the table row for name, or nil.
func officialByName(name string) *OfficialTemplate {
	for i := range Official {
		if Official[i].Name == name {
			return &Official[i]
		}
	}
	return nil
}

// bareWordRE is the shortcut grammar: a template reference whose head is a
// bare word (letters, digits, underscores) expands into the reserved segment.
// A bare word can never be a valid module path — paths require a dotted first
// segment — so the expansion is collision-free by grammar.
var bareWordRE = regexp.MustCompile(`^[a-z0-9_]+$`)

// majorRE matches a major-float version suffix (@v1).
var majorRE = regexp.MustCompile(`^v\d+$`)

// TemplateRef is a parsed template reference: where to fetch from and which
// version to select.
type TemplateRef struct {
	// Raw is the reference as given.
	Raw string
	// Base is the module path without a major suffix.
	Base string
	// Major is the major to float within ("v1"), empty when the reference
	// pins an exact version or constrains no major.
	Major string
	// Exact is the exact bare SemVer to pin, empty when floating.
	Exact string
	// Shortcut reports that Base was expanded from a bare word into the
	// reserved segment.
	Shortcut bool
}

// LookupPath is the path version resolution enumerates: the base plus the
// major when one constrains the float.
func (r TemplateRef) LookupPath() string {
	if r.Major != "" {
		return r.Base + "@" + r.Major
	}
	return r.Base
}

// ParseTemplateRef classifies a template reference by shape — purely
// syntactically, touching no registry:
//
//   - a bare word (`standard`) expands to the reserved segment; its default
//     major comes from the baked table when the word names an official
//     template, otherwise the float is unconstrained (an unknown word then
//     fails at resolution, inside the segment — never by falling back);
//   - `word@vN` floats within the major, `word@X.Y.Z` pins the exact tag;
//   - a reference containing `/` or `.` is a literal module path, never
//     expanded, with the same optional `@vN` / `@X.Y.Z` suffix semantics.
func ParseTemplateRef(ref string) (TemplateRef, error) {
	if ref == "" {
		return TemplateRef{}, fmt.Errorf("template reference must not be empty")
	}

	head, suffix := ref, ""
	if i := strings.LastIndex(ref, "@"); i >= 0 {
		head, suffix = ref[:i], ref[i+1:]
	}
	if head == "" {
		return TemplateRef{}, fmt.Errorf("template reference %q names no template before the @", ref)
	}

	out := TemplateRef{Raw: ref}
	switch {
	case bareWordRE.MatchString(head):
		out.Base = Segment + "/" + head
		out.Shortcut = true
		if suffix == "" {
			if t := officialByName(head); t != nil {
				out.Major = t.DefaultMajor
			}
		}
	case strings.ContainsAny(head, "/."):
		out.Base = head
	default:
		return TemplateRef{}, fmt.Errorf("template reference %q is neither a template name (letters, digits, underscores) nor a module path", ref)
	}

	switch {
	case suffix == "":
	case majorRE.MatchString(suffix):
		out.Major = suffix
	case cueedit.CheckVersion(suffix) == nil:
		out.Exact = suffix
		out.Major = "v" + strings.SplitN(suffix, ".", 2)[0]
	default:
		return TemplateRef{}, fmt.Errorf("template reference %q carries version %q, which is neither a major (v1) nor an exact SemVer (1.2.3)", ref, suffix)
	}
	return out, nil
}
