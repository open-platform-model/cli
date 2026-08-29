// Package publish implements the identity-driven publish pipeline shared by
// `opm module publish` and `opm catalog publish` (enhancement 0011, slice
// cli-publish-pipeline).
//
// The pipeline derives every coordinate from what the artifact itself
// declares — decode the tree, read identity/identity.cue, unify it against
// core's #IdentityPackage, split metadata.modulePath into repository and
// major — and never edits the artifact to match a coordinate. What is
// published is exactly the committed tree (D2). Every evaluable gate runs,
// refusals accumulate into one list, and the resolved plan prints before any
// push; --dry-run stops after the plan.
package publish

import (
	"fmt"

	"cuelang.org/go/cue"
	"cuelang.org/go/mod/modregistry"
)

// Kind names the artifact kind being published. The two entry points differ
// by exactly three predicates: where identity's package name matters, whether
// the package-name gate applies, and whether the local-override gate can be
// waived.
type Kind string

// The two artifact kinds the pipeline publishes.
const (
	KindModule  Kind = "module"
	KindCatalog Kind = "catalog"
)

// Options carries one publish invocation's inputs.
type Options struct {
	// Dir is the artifact tree's root directory (holds cue.mod/ and
	// identity/).
	Dir string

	// Kind selects the entry point's differences (package-name gate,
	// override-gate waiver).
	Kind Kind

	// Context is the CUE context every load and lookup runs in.
	Context *cue.Context

	// IdentitySchema is core's #IdentityPackage definition, resolved from the
	// kernel's schema cache. The identity package is validated by unifying
	// against it, so CUE produces the diagnostic (refusal 10).
	IdentitySchema cue.Value

	// MemberFQNGateSchema is core's #CatalogMemberFQNGate definition, resolved
	// beside IdentitySchema. Required for catalog publish; module publish
	// never reads it.
	MemberFQNGateSchema cue.Value

	// TraitOptionalGateSchema is core's #TraitOptionalGate definition,
	// resolved beside IdentitySchema. Required for catalog publish.
	TraitOptionalGateSchema cue.Value

	// Registry is the resolved CUE registry (flag > env > config). Empty
	// inherits the process environment.
	Registry string

	// VersionFlag is --version: fills an open identity Version or asserts a
	// concrete one, never overwrites. Empty means absent.
	VersionFlag string

	// SkipOverrideCheck waives the local-override gate. Module publish only;
	// it never changes resolution — replacements are ignored either way.
	SkipOverrideCheck bool
}

// IdentityFieldState classifies an identity field per D4's tristate.
type IdentityFieldState string

// The three states an identity field can be in. Only concrete is publishable;
// open becomes publishable when --version fills it; absent is a malformed
// artifact (refusal 10 names it), never fillable.
const (
	StateConcrete IdentityFieldState = "concrete"
	StateOpen     IdentityFieldState = "open"
	StateAbsent   IdentityFieldState = "absent"
)

// IdentityField is one identity field's resolved state in the plan.
type IdentityField struct {
	// Name is the schema-fixed field name ("ModulePath", "Version").
	Name string
	// State is the D4 tristate.
	State IdentityFieldState
	// Value is the concrete value, when State is concrete.
	Value string
	// Defaulted reports the value came from a CUE default — concrete, with
	// the default as its value (the committed, release-automation-owned
	// declaration).
	Defaulted bool
	// Filled reports --version is filling this open field on this invocation.
	Filled bool
	// Pos is the field's declaration position (file:line), when known.
	Pos string
}

// Replacement is one cue.mod/local-module.cue replaceWith entry, reported
// beside the registry version that supersedes it.
type Replacement struct {
	// Path is the replaced dependency's module path, major included.
	Path string
	// ReplaceWith is the local directory or fork the working tree points at.
	ReplaceWith string
	// PublishedVersion is the version cue.mod/module.cue pins — what the
	// published artifact will resolve instead. Empty when the dep is not
	// pinned.
	PublishedVersion string
}

// Plan is everything publish resolves before it pushes — 0011's #PublishPlan
// as data. Every field is derived from the artifact or supplied deliberately;
// nothing is invented.
type Plan struct {
	Kind Kind
	Dir  string

	// DeclaredPath is the artifact's own identity ModulePath — the complete
	// CUE module path including the major (read, never composed).
	DeclaredPath string
	// CueModPath is cue.mod/module.cue's module: line; CueModPos its position.
	CueModPath string
	CueModPos  string
	// RegistryRepo and Major are DeclaredPath split at "@".
	RegistryRepo string
	Major        string

	// Tag is what would be pushed ("v" + effective version); TagSource names
	// where the version came from.
	Tag       string
	TagSource string

	// Identity holds the per-field states.
	Identity []IdentityField

	// OverridePresent reports cue.mod/local-module.cue exists with
	// replacements; OverrideWaived that --skip-override-check waived the gate
	// (modules only). Replacements lists the entries either way.
	OverridePresent bool
	OverrideWaived  bool

	// KernelChecked reports that the kernel loader gate ran (msg 12); a
	// module whose identity is open or absent skips it, so the row is
	// omitted rather than misread as a pass. KernelAccepted is its verdict.
	KernelChecked  bool
	KernelAccepted bool
	Replacements   []Replacement

	// FillVersion, when non-empty, is the version Push writes into
	// identity/identity.cue (through internal/cueedit) before zipping — D12:
	// the fill writes the working tree, because the pushed bytes come from
	// disk.
	FillVersion string

	// RegistryChecked reports the already-published lookup completed. When
	// the registry was unreachable it stays false, and a refusal-free plan
	// renders INCOMPLETE rather than a GO the gates never finished earning.
	RegistryChecked bool

	// CatalogGates carries the catalog-only gates' per-gate outcome counts
	// (member/posture/compat). Zero-valued for modules.
	CatalogGates CatalogGateOutcomes

	// CompatChecked reports the compatibility walk completed (catalogs only).
	// A walk aborted mid-flight by a transport failure leaves it false, and a
	// refusal-free plan renders INCOMPLETE — no partial verdict (D35's
	// connectivity contract extends to the member walk).
	CompatChecked bool

	// Refusals is the accumulated refusal list. Empty means GO.
	Refusals []Refusal

	// registryClient is the client the already-published lookup built,
	// carried so Push does not resolve registry configuration and
	// credentials a second time.
	registryClient *modregistry.Client

	// members is the enumerated member list (catalogs only), carried from the
	// local gates to the compatibility gate so the tree is walked once.
	members []Member

	// publishedVersions is what the already-published lookup enumerated,
	// carried so the predecessor scan does not repeat the round-trip.
	publishedVersions []string
}

// Go reports whether every gate passed.
func (p *Plan) Go() bool { return len(p.Refusals) == 0 }

// refuse appends a refusal to the plan.
func (p *Plan) refuse(r Refusal) { p.Refusals = append(p.Refusals, r) }

// ConnectivityError reports the registry could not be reached for a lookup or
// a push. It is not a refusal: the artifact was never judged. Commands map it
// to ExitConnectivityError (3).
type ConnectivityError struct {
	// Op names the registry operation that failed.
	Op  string
	Err error
}

func (e *ConnectivityError) Error() string {
	return fmt.Sprintf("registry unreachable: %s: %v", e.Op, e.Err)
}

func (e *ConnectivityError) Unwrap() error { return e.Err }
