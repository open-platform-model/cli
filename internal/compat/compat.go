package compat

import (
	"bytes"
	"errors"
	"fmt"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/format"
)

// provenanceDenylist is 0010:D30's exact set: the metadata fields that change
// per catalog release by construction (0010:D25 — provenance only) and are
// therefore never contract surface. The comparison walk skips them directly
// under any metadata field at every depth; nothing else is excluded —
// identity fields and labels stay.
var provenanceDenylist = map[string]bool{
	"catalogVersion": true,
	"description":    true,
}

// Violation is one breach of 0010:D27's additive-only rule, located by the
// dotted path from the compared root. Violations are results, not errors
// (opm/errors doctrine): the walk reports every breach it finds and never
// fails. The primitive's name, apiVersion, and predecessor coordinate are
// caller-attached — the walk does not know them.
type Violation struct {
	Path string // dotted path from the compared root, "" for top-level
	Kind string // one of the Kind* constants
	Old  string // rendered prior value; "" when not applicable
	New  string // rendered new value; "" when not applicable
}

// The violation kinds. KindDomainNarrowed carries the CUE subsumption
// diagnostic verbatim in New (no reformatting);
// the default kinds carry the rendered defaults in Old/New.
const (
	KindFieldRemoved      = "field removed"
	KindFieldAddedStrict  = "field added without optional or default"
	KindDefaultChanged    = "default changed"
	KindDefaultRemoved    = "default removed"
	KindDomainNarrowed    = "domain narrowed"
	KindFieldMadeRequired = "field made required"
)

// Sentinel errors for [CheckAtLevel]'s error channel. A gate that cannot
// classify its input has not found an incompatibility — these are failures,
// never violations.
var (
	ErrUnparseableAPIVersion = errors.New("unparseable apiVersion")
	ErrNotStruct             = errors.New("operand is not a struct")
)

// CheckAtLevel is the level-aware entry point (0010:D34): the additive-only
// promise binds at beta and GA only, so an alpha apiVersion returns (nil, nil)
// without evaluating the operands. The apiVersion is the primitive's own — a
// catalog's release version is an independent axis and must not be passed
// here. An apiVersion outside core's #APIVersionType grammar is an error, as
// are non-struct top-level operands.
func CheckAtLevel(apiVersion string, prev, next cue.Value) ([]Violation, error) {
	_, lvl, ok := ParseLevel(apiVersion)
	if !ok {
		return nil, fmt.Errorf("%w: %q", ErrUnparseableAPIVersion, apiVersion)
	}
	if !lvl.Enforced() {
		return nil, nil
	}
	if k := prev.IncompleteKind(); k != cue.StructKind {
		return nil, fmt.Errorf("%w: previous operand is %v", ErrNotStruct, k)
	}
	if k := next.IncompleteKind(); k != cue.StructKind {
		return nil, fmt.Errorf("%w: next operand is %v", ErrNotStruct, k)
	}
	return Check(prev, next), nil
}

// Check reports every violation of 0010:D27's additive-only rule in next relative
// to prev: fields and options may be added and never removed; a newly added
// field must be optional or defaulted; an existing field's default is
// immutable. It is level-blind — see [CheckAtLevel] — and cannot fail given
// two valid values.
//
// The comparison is a field-wise walk, deliberately not a single Subsume call
// in either direction: adding a struct field makes a value more specific while
// adding a disjunct makes it less specific, and 0010:D27 calls both "additive", so
// the rule spans both directions of the lattice while one subsume call tests
// one (measured 10/14 and 8/14 against the 0010:D27 change classes in
// enhancements/0011/experiments/03-d27-compat-gate; the walk is 14/14).
// Structs recurse; leaves get a forward subsume, where it is correct for the
// value domain; defaults are compared explicitly at every level, because
// subsume is blind to them in both directions.
//
// Three rules keep the walk from reporting non-changes (measured on
// catalog_opm PR 51, cli issue 165):
//
//   - 0010:D30's provenance fields are skipped at every depth: catalogVersion
//     and description directly under any field named metadata, so a member
//     reference embedded in another member (appliesTo, composedResources)
//     does not report the referenced member's per-release provenance.
//   - Closed lists of equal length are walked element-wise (paths name[i]),
//     so those embedded references reach the rule above; any other list pair
//     is a leaf.
//   - A leaf whose emitted syntax is byte-identical on both sides reports
//     nothing: it cannot have narrowed, and the forward subsume false-positives
//     on unchanged leaves carrying matchN or a pending comprehension.
func Check(prev, next cue.Value) []Violation {
	return walk("", prev, next, false, nil)
}

// walk compares one position. underMetadata is true when path names a direct
// child of a field called metadata, which is where 0010:D30's denylist applies.
func walk(path string, prev, next cue.Value, underMetadata bool, acc []Violation) []Violation {
	// Defaults first, at every level — no subsume direction can see them.
	acc = checkDefaults(path, prev, next, acc)

	// A disjunction of structs has StructKind but is not field-iterable
	// content (Fields silently yields nothing on it) — it is walked as a
	// leaf, so branch removal surfaces as domain narrowing and branch-
	// internal field edits are judged by subsumption alone.
	if isWalkableStruct(prev) && isWalkableStruct(next) {
		pit, perr := prev.Fields(cue.All())
		nit, nerr := next.Fields(cue.All())
		if perr == nil && nerr == nil {
			return walkStruct(path, pit, nit, next, underMetadata, acc)
		}
	}

	if out, ok := walkList(path, prev, next, acc); ok {
		return out
	}

	// An open list is a leaf, and its implicit default is not compared, so
	// the defaults its fixed elements carry are compared here.
	acc = openListDefaults(path, prev, next, acc)

	// An unchanged leaf cannot have narrowed. Syntax(cue.All()) expands
	// references (a changed definition behind a reference renders
	// differently), so byte equality is a sound "unchanged" signal; rendering
	// failure falls through to the subsume.
	if leafIdentical(prev, next) {
		return acc
	}

	// Leaf: the new domain must accept everything the old one did — widening
	// passes, narrowing is refused. cue.Schema() and cue.Raw() are
	// load-bearing measured options (see the option-pinning tests): Schema
	// keeps optional-field edits inside disjunction branches from reporting
	// spuriously; Raw keeps the check default-sensitive so a domain narrowed
	// to its own default is not waved through.
	if err := next.Subsume(prev, cue.Schema(), cue.Raw()); err != nil {
		acc = append(acc, Violation{Path: path, Kind: KindDomainNarrowed, New: err.Error()})
	}
	return acc
}

// isWalkableStruct reports whether v is a plain struct the walk may iterate —
// StructKind and not a disjunction.
func isWalkableStruct(v cue.Value) bool {
	if v.IncompleteKind() != cue.StructKind {
		return false
	}
	op, _ := v.Expr()
	return op != cue.OrOp
}

func walkStruct(path string, pit, nit *cue.Iterator, next cue.Value, underMetadata bool, acc []Violation) []Violation {
	// The new side, by name, so the prior-side loop can compare selectors
	// (constraint markers live on the selector, not the value) and the
	// addition loop below needs no second iteration. Keyed by name on
	// purpose: `y`, `y?` and `y!` are three constraint flavours of one label
	// and must find each other, which a LookupPath by selector does not do
	// (a plain selector misses optional and required fields).
	type field struct {
		sel cue.Selector
		val cue.Value
	}
	added := map[string]field{}
	var order []string
	for nit.Next() {
		sel := nit.Selector()
		if sel.LabelType() == cue.HiddenLabel {
			continue
		}
		name := fieldName(sel)
		added[name] = field{sel: sel, val: nit.Value()}
		order = append(order, name)
	}

	for pit.Next() {
		sel := pit.Selector()
		if sel.LabelType() == cue.HiddenLabel {
			continue // hidden fields are not contract surface
		}
		name := fieldName(sel)
		if underMetadata && provenanceDenylist[name] {
			continue // 0010:D30: per-release provenance, never contract surface
		}
		nf, present := added[name]
		delete(added, name)
		if !present {
			acc = append(acc, Violation{Path: join(path, name), Kind: KindFieldRemoved})
			continue
		}
		// An optional field that becomes required breaks every consumer
		// that omitted it (0010:D27). Judged on the selectors: the leaf
		// subsume below compares value domains and cannot see a constraint
		// marker. A defaulted field that loses its default is the default
		// rule's finding, not this one's.
		if sel.ConstraintType()&cue.OptionalConstraint != 0 && required(nf.sel, nf.val) {
			acc = append(acc, Violation{Path: join(path, name), Kind: KindFieldMadeRequired})
		}
		acc = walk(join(path, name), pit.Value(), nf.val, name == "metadata", acc)
	}

	for _, name := range order {
		nf, ok := added[name]
		if !ok || (underMetadata && provenanceDenylist[name]) {
			continue
		}
		// An added field must be optional or carry a default — a new
		// required field breaks every existing consumer.
		if required(nf.sel, nf.val) {
			acc = append(acc, Violation{Path: join(path, name), Kind: KindFieldAddedStrict})
		}
	}
	return acc
}

// required reports 0010:D27's posture of a field: a `!` field, or a regular
// field with no default, is required — a consumer must supply it. A `?` field
// or a defaulted one is optional.
func required(sel cue.Selector, v cue.Value) bool {
	ct := sel.ConstraintType()
	if ct&cue.OptionalConstraint != 0 {
		return false
	}
	if ct&cue.RequiredConstraint != 0 {
		return true
	}
	_, hasDefault := v.Default()
	return !hasDefault
}

// walkList walks two closed lists of equal length element-wise, paths
// name[i], and reports ok=false for any other pair (open lists, differing
// lengths, non-lists), which the caller judges as a leaf. Element-wise is the
// member-reference case (appliesTo, composedResources); 0010:D27 states no list
// semantics, so addition and removal keep their subsume verdict.
func walkList(path string, prev, next cue.Value, acc []Violation) ([]Violation, bool) {
	if prev.IncompleteKind() != cue.ListKind || next.IncompleteKind() != cue.ListKind {
		return acc, false
	}
	if prev.Allows(cue.AnyIndex) || next.Allows(cue.AnyIndex) {
		return acc, false // open list: no fixed elements to pair
	}
	pn, perr := prev.Len().Int64()
	nn, nerr := next.Len().Int64()
	if perr != nil || nerr != nil || pn != nn {
		return acc, false
	}
	pit, perr := prev.List()
	nit, nerr := next.List()
	if perr != nil || nerr != nil {
		return acc, false
	}
	for i := 0; pit.Next() && nit.Next(); i++ {
		acc = walk(fmt.Sprintf("%s[%d]", path, i), pit.Value(), nit.Value(), false, acc)
	}
	return acc, true
}

// openListDefaults compares the authored defaults inside the fixed elements
// of two plain open lists with the same number of fixed elements, at paths
// name[i]. CUE folds those defaults into the open list's implicit default,
// which [authoredDefault] skips, and the leaf subsume does not see a removed
// one, so without this walk `[*"a" | string, ...string]` to
// `[string, ...string]` would report nothing. Only defaults are judged here;
// the list's value domain stays the leaf subsume's.
func openListDefaults(path string, prev, next cue.Value, acc []Violation) []Violation {
	pe, pok := openListElements(prev)
	ne, nok := openListElements(next)
	if !pok || !nok || len(pe) != len(ne) {
		return acc
	}
	for i := range pe {
		acc = walkDefaults(fmt.Sprintf("%s[%d]", path, i), pe[i], ne[i], false, acc)
	}
	return acc
}

// openListElements is the fixed elements of a plain open list (see
// [implicitListDefault]); ok is false for any other value.
func openListElements(v cue.Value) ([]cue.Value, bool) {
	if !implicitListDefault(v) {
		return nil, false
	}
	return listElements(v)
}

// walkDefaults applies the default rule alone through one position and
// everything below it: struct fields present on both sides (by name, hidden
// fields and 0010:D30's provenance skipped as in [walkStruct]), the elements
// of equal-length closed lists, and the fixed elements of open lists.
// Removed and added fields are not its concern.
func walkDefaults(path string, prev, next cue.Value, underMetadata bool, acc []Violation) []Violation {
	acc = checkDefaults(path, prev, next, acc)
	if isWalkableStruct(prev) && isWalkableStruct(next) {
		return walkStructDefaults(path, prev, next, underMetadata, acc)
	}
	if pe, ne, ok := closedListPairs(prev, next); ok {
		for i := range pe {
			acc = walkDefaults(fmt.Sprintf("%s[%d]", path, i), pe[i], ne[i], false, acc)
		}
		return acc
	}
	return openListDefaults(path, prev, next, acc)
}

// walkStructDefaults is [walkDefaults] over the fields two structs share.
func walkStructDefaults(path string, prev, next cue.Value, underMetadata bool, acc []Violation) []Violation {
	pit, perr := prev.Fields(cue.All())
	nit, nerr := next.Fields(cue.All())
	if perr != nil || nerr != nil {
		return acc
	}
	nextByName := map[string]cue.Value{}
	for nit.Next() {
		if nit.Selector().LabelType() != cue.HiddenLabel {
			nextByName[fieldName(nit.Selector())] = nit.Value()
		}
	}
	for pit.Next() {
		sel := pit.Selector()
		name := fieldName(sel)
		if sel.LabelType() == cue.HiddenLabel || (underMetadata && provenanceDenylist[name]) {
			continue
		}
		if nv, ok := nextByName[name]; ok {
			acc = walkDefaults(join(path, name), pit.Value(), nv, name == "metadata", acc)
		}
	}
	return acc
}

// closedListPairs is the elements of two closed lists of equal length; ok is
// false for any other pair.
func closedListPairs(prev, next cue.Value) (pe, ne []cue.Value, ok bool) {
	if prev.IncompleteKind() != cue.ListKind || next.IncompleteKind() != cue.ListKind ||
		prev.Allows(cue.AnyIndex) || next.Allows(cue.AnyIndex) {
		return nil, nil, false
	}
	pe, pok := listElements(prev)
	ne, nok := listElements(next)
	if !pok || !nok || len(pe) != len(ne) {
		return nil, nil, false
	}
	return pe, ne, true
}

// listElements is the fixed elements of the list v.
func listElements(v cue.Value) ([]cue.Value, bool) {
	it, err := v.List()
	if err != nil {
		return nil, false
	}
	var out []cue.Value
	for it.Next() {
		out = append(out, it.Value())
	}
	return out, true
}

// leafIdentical reports whether two leaves emit byte-identical syntax under
// cue.All(). Both operands come from the same conventions, so an unchanged
// leaf formats identically; a rendering failure is treated as "different".
func leafIdentical(prev, next cue.Value) bool {
	pb, err := format.Node(prev.Syntax(cue.All()))
	if err != nil {
		return false
	}
	nb, err := format.Node(next.Syntax(cue.All()))
	if err != nil {
		return false
	}
	return bytes.Equal(pb, nb)
}

// checkDefaults enforces default immutability (0010:D27). Only authored
// defaults are compared (see [authoredDefault]), and only when the prior build
// has one — adding a default where none existed is additive. When either
// side's default is non-concrete, equality is judged by mutual subsumption so
// a merely-reordered disjunction does not report.
func checkDefaults(path string, prev, next cue.Value, acc []Violation) []Violation {
	pd, phas := authoredDefault(prev)
	if !phas {
		return acc
	}
	nd, nhas := authoredDefault(next)
	if !nhas {
		return append(acc, Violation{Path: path, Kind: KindDefaultRemoved, Old: render(pd)})
	}
	if pd.IsConcrete() && nd.IsConcrete() {
		if !pd.Equals(nd) {
			return append(acc, Violation{Path: path, Kind: KindDefaultChanged, Old: render(pd), New: render(nd)})
		}
		return acc
	}
	if pd.Subsume(nd, cue.Schema(), cue.Raw()) != nil || nd.Subsume(pd, cue.Schema(), cue.Raw()) != nil {
		return append(acc, Violation{Path: path, Kind: KindDefaultChanged, Old: render(pd), New: render(nd)})
	}
	return acc
}

// authoredDefault is v's default when an author marked one with `*`. CUE
// also reports a default for every plain open list — the list closed at its
// fixed elements: [] for [...T], [T] for [...T] & [_, ...] — which nobody
// wrote and which is not contract surface. Compared, it reports "default
// changed" with the same rendering on both sides whenever the field's
// constraint marker changes (measured on catalog_opm's role subjects, cue
// v0.17.1). Defaults written inside an open list's fixed elements are folded
// into that implicit default; [openListDefaults] compares them element-wise.
func authoredDefault(v cue.Value) (cue.Value, bool) {
	d, ok := v.Default()
	if !ok || implicitListDefault(v) {
		return d, false
	}
	return d, true
}

// implicitListDefault reports whether v is a plain open list, whose own
// default is CUE's implicit one (defaults written in its fixed elements are
// [openListDefaults]' concern). Len is the discriminator: it evaluates on a
// list value only (a disjunction, which is where an authored default at this
// level lives, has no length) and is non-concrete exactly when the list is
// open. The authored-list-default test cases pin this behaviour of Len.
func implicitListDefault(v cue.Value) bool {
	if v.IncompleteKind() != cue.ListKind {
		return false
	}
	n := v.Len()
	return n.Err() == nil && !n.IsConcrete()
}

// fieldName is the clean field name for paths and dedup — no ?/! markers, no
// quotes on ident-safe labels.
func fieldName(sel cue.Selector) string {
	if sel.LabelType() == cue.StringLabel {
		return sel.Unquoted()
	}
	return sel.String()
}

func render(v cue.Value) string { return fmt.Sprintf("%v", v) }

func join(p, n string) string {
	if p == "" {
		return n
	}
	return p + "." + n
}
