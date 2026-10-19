# Capability: catalog-compatibility

## Purpose

Publish-side catalog compatibility, owned by the CLI: the additive-only comparison walk, the contract-level classification and the provenance strip of enhancement 0010, and the float selector template resolution uses to pick a version. Consumed by the publish gate of enhancement 0011 (`opm module publish`, `opm catalog publish`), by `opm catalog registry check --compat`, and by template scaffolding. Pure logic — no I/O, no registry access, no schema-cache dependency; callers compose enumeration, fetching and gate policy around it.

## Requirements

### Requirement: `internal/compat.Check` reports every violation of the additive-only rule
The CLI SHALL provide a pure comparison (`internal/compat.Check`) that, given a prior and a new definition of the same contract as `cue.Value`s, reports every violation of the additive-only rule of enhancement 0010: fields and options MAY be added and MUST NOT be removed; a newly added field MUST be optional or defaulted; an existing field's default MUST NOT change; an existing optional or defaulted field MUST NOT become required.

#### Scenario: Field removal reported

- **WHEN** the new definition lacks a field the prior definition declares
- **THEN** `Check` reports one violation at that field's path with kind `field removed`

#### Scenario: Strict field addition reported

- **WHEN** the new definition adds a required field with no default
- **THEN** `Check` reports kind `field added without optional or default` at that path
- **AND** adding the same field as optional or defaulted reports nothing

#### Scenario: Additive widening passes

- **WHEN** the new definition only adds an option to a disjunction or adds an optional field
- **THEN** `Check` reports no violations

### Requirement: `Check` is a field-wise walk
The comparison (`internal/compat.Check`) SHALL be a field-wise walk: struct recursion for the removed-field, optional-or-defaulted and made-required rules, forward subsumption at leaves for the value domain, and explicit default comparison at every level, and SHALL NOT be implemented as a single subsumption call in either direction. (Requirement: `internal/compat.Check` reports every violation of the additive-only rule)

#### Scenario: Domain narrowing reported at a leaf

- **WHEN** a leaf's value domain no longer accepts a value the prior domain accepted
- **THEN** `Check` reports kind `domain narrowed` at that path carrying the CUE subsumption diagnostic verbatim

### Requirement: Each violation carries the dotted path from the compared root and a stable kind discriminator
Each violation SHALL carry the dotted path from the compared root and a stable kind discriminator; default-change violations SHALL carry the rendered old and new values. (Requirement: `internal/compat.Check` reports every violation of the additive-only rule)

#### Scenario: Default change reported with both values

- **WHEN** an existing field's default changes
- **THEN** `Check` reports kind `default changed` at that path carrying the old and new rendered values
- **AND** neither direction of whole-value subsumption would have detected it

### Requirement: A field that was optional and is required in the new definition reports `field made required`
A field is "required" for this rule when it carries the required constraint (`!`) or is a regular field with no default. A field that carried the optional constraint (`?`) in the prior definition and is required in the new one SHALL report kind `field made required` at its path, independently of whether its value domain also changed. (Requirement: `internal/compat.Check` reports every violation of the additive-only rule)

#### Scenario: Field made required reported

- **WHEN** a field declared `y?: string` in the prior definition is declared `y!: string` in the new one
- **THEN** `Check` reports kind `field made required` at `y`
- **AND** declaring it `y: string` (regular, no default) reports the same kind
- **AND** declaring it `y!: =~"^[a-z]"` reports both `field made required` and `domain narrowed` at `y`
- **AND** the reverse (`y!: string` → `y?: string`, or `y: string` → `y: string | *"z"`) reports no `field made required`

### Requirement: A defaulted field that loses its default and the reverse transition do not report `field made required`
A defaulted field that loses its default is the default rule's finding (`default removed`), not this one's. The reverse transition (required to optional, or an optional field gaining a default) SHALL NOT report this kind. (Requirement: A field that was optional and is required in the new definition reports `field made required`)

#### Scenario: A defaulted field that loses its default reports `default removed`, not `field made required`

- **WHEN** a defaulted field loses its default in the new definition
- **THEN** `Check` reports `default removed` at its path, not `field made required`

### Requirement: A default is compared only where an author wrote one, and the implicit default of a plain open list is not compared
A default is compared only where an author wrote one: a value whose disjunction marks a default with `*`. The implicit default CUE gives a plain open list (the list closed at its fixed elements: `[]` for `[...T]`, `[T]` for `[...T] & [_, ...]`) SHALL NOT count as a default on either side: it SHALL NOT be compared, its absence SHALL NOT report `default removed`, and a constraint-marker change on such a list (required to optional) SHALL report nothing. (Requirement: `Check` is a field-wise walk)

#### Scenario: Implicit open-list default is not compared

- **WHEN** a field declared `subjects!: [...#Subject] & [_, ...]` in the prior definition, where `#Subject` is a disjunction of structs, is declared `subjects?: [...#Subject] & [_, ...]` in the new one
- **THEN** `Check` reports no violations
- **AND** the same holds for `xs!: [...string] & [_, ...]` to `xs?: [...string] & [_, ...]`, and for a plain `xs: [...string]` made `xs?: [...string]`

#### Scenario: Authored list default is still compared

- **WHEN** a field declared `xs: *["a"] | [...string]` is declared `xs: *["b"] | [...string]` in the new definition
- **THEN** `Check` reports `default changed` at `xs`
- **AND** declaring it `xs: [...string]` instead reports `default removed` at `xs`
- **AND** declaring `xs: *[] | [...string]` as `xs: *["a"] | [...string]` reports `default changed` at `xs`
- **AND** adding a default, `xs: [...string]` to `xs: *["a"] | [...string]`, reports nothing

### Requirement: An open list's value domain is still judged at the leaf
An open list's value domain SHALL still be judged at the leaf, so narrowing it keeps reporting `domain narrowed`. (Requirement: A default is compared only where an author wrote one, and the implicit default of a plain open list is not compared)

#### Scenario: Open list narrowed reports only the narrowing

- **WHEN** a field declared `xs: [...string]` is declared `xs: [...string] & [_, ...]` in the new definition
- **THEN** `Check` reports `domain narrowed` at `xs`
- **AND** no `default changed` at `xs`

### Requirement: The default rule is applied to each pair of an open list's fixed elements
Defaults written inside an open list's fixed elements (`[*"a" | string, ...string]`, `[{a: *1 | int}, ...]`) are authored: when both sides are plain open lists with the same number of fixed elements, the default rule SHALL be applied to each pair of fixed elements and everything below them, at paths `name[i]`. (Requirement: A default is compared only where an author wrote one, and the implicit default of a plain open list is not compared)

#### Scenario: Default inside an open list's fixed element is compared

- **WHEN** a field declared `xs: [*"a" | string, ...string]` is declared `xs: [string, ...string]` in the new definition
- **THEN** `Check` reports `default removed` at `xs[0]`
- **AND** declaring `xs: [{a: *1 | int}, ...]` as `xs: [{a: int}, ...]` reports `default removed` at `xs[0].a`
- **AND** declaring it `xs: [{a: *2 | int}, ...]` reports `default changed` at `xs[0].a`

### Requirement: The walk applies the provenance denylist at every depth
The walk SHALL apply the provenance denylist of enhancement 0010 at every depth: the direct children `catalogVersion` and `description` of any field named `metadata` reached by the walk SHALL be neither compared nor reported, so a member reference embedded in another member (`appliesTo`, `composedResources`, `composedTraits`) does not report the referenced member's per-release provenance. (Requirement: `Check` is a field-wise walk)

#### Scenario: Provenance inside a member reference is ignored

- **WHEN** a trait's `appliesTo` list references a resource whose `metadata.catalogVersion` default differs between the two builds and nothing else differs
- **THEN** `Check` reports no violations

### Requirement: Lists whose two sides have equal length are walked element-wise and lists of unequal length are judged as a leaf
Lists whose two sides have equal length SHALL be walked element-wise so their elements reach that rule; lists of unequal length SHALL be judged as a leaf. (Requirement: The walk applies the provenance denylist at every depth)

#### Scenario: Equal-length lists are walked

- **WHEN** a blueprint's `composedResources` list has the same length on both sides and one element removed a field
- **THEN** `Check` reports `field removed` at `composedResources[<i>].<field>`

#### Scenario: Unequal-length lists are a leaf

- **WHEN** the new definition drops an element from a list
- **THEN** `Check` reports `domain narrowed` at the list's path

Known limitation (measured 2026-08-16, cue v0.17.1; narrowed 2026-08-26): a leaf that both changed and carries a `matchN` validator or a pending comprehension is judged by the forward subsume alone, which cannot evaluate those constructs: narrowing `matchN`'s alternatives is not detected and widening them reports `domain narrowed` spuriously. Unchanged leaves no longer report. A comparator-level fix for the changed-leaf residue is unowned.

### Requirement: A leaf whose emitted syntax is identical on both sides reports nothing
A leaf whose emitted syntax is identical on both sides SHALL report nothing. (Requirement: `Check` is a field-wise walk)

#### Scenario: Unchanged guarded leaf is silent

- **WHEN** a field typed by a struct carrying `if`-guards over not-yet-concrete siblings (the `#Image` shape) or a `matchN` validator is byte-identical on both sides
- **THEN** `Check` reports no violations at that path

### Requirement: The CLI classifies a primitive's `apiVersion` on the `vNalphaM | vNbetaM | vN` ladder, exposes whether a level is enforced and pins the grammar against core
The CLI SHALL classify a primitive's `apiVersion` on the `vNalphaM | vNbetaM | vN` ladder (`internal/compat.ParseLevel`) using the exact grammar of core's `#APIVersionType`, and SHALL expose whether a level is enforced (`Enforced`: beta and GA yes, alpha no). The grammar SHALL be pinned against core's own declaration by a parity test, so a core change that widens or narrows `#APIVersionType` fails here rather than silently diverging.

#### Scenario: Beta and GA are gated

- **WHEN** `CheckAtLevel` is called with `"v1beta1"`, `"v1"`, or `"v2"` and incompatible operands
- **THEN** it returns the same violations `Check` reports

#### Scenario: Grammar parity with core is pinned

- **WHEN** the apiVersion pattern this package accepts is compared with core's `#APIVersionType` declaration
- **THEN** a test fails if the two have diverged

### Requirement: The level-aware entry point `CheckAtLevel` returns no violations for an alpha `apiVersion` and an error for an `apiVersion` the grammar rejects
The level-aware entry point (`CheckAtLevel`) SHALL return no violations and no error for an alpha `apiVersion` without evaluating the operands, and SHALL return an error, not a violation, for an `apiVersion` the grammar rejects.

#### Scenario: Alpha is not gated

- **WHEN** `CheckAtLevel` is called with `apiVersion: "v1alpha1"` and operands that differ incompatibly
- **THEN** it returns no violations and no error

#### Scenario: Unparseable apiVersion is an error

- **WHEN** `CheckAtLevel` is called with an `apiVersion` core's `#APIVersionType` rejects (e.g. `"v1alpha"`, `"1.2.0"`)
- **THEN** it returns a non-nil error and no violations

### Requirement: The CLI provides a total, transitive ordering over valid `apiVersion` strings
The CLI SHALL provide a total, transitive ordering over valid `apiVersion` strings (`CompareAPIVersions`).

#### Scenario: Ordering is transitive

- **WHEN** `CompareAPIVersions` orders `v1alpha1`, `v2`, and `v10`
- **THEN** the resulting order is the same regardless of input order

### Requirement: The CLI provides a stable-preferring version selector over a published-version list that is pure
The CLI SHALL provide a stable-preferring version selector over a published-version list, in the package that resolves template versions: given the registry's `v`-prefixed SemVer-ascending list, it SHALL return the highest stable (non-prerelease) version, skipping unparseable entries, and SHALL fall back to the highest version overall when no stable version exists. Selection SHALL be pure: enumeration and fetching are the caller's.

#### Scenario: Stable preferred over higher prerelease

- **WHEN** the list contains `v1.2.0` and `v1.3.0-alpha.1`
- **THEN** the selector returns `v1.2.0`

#### Scenario: Prerelease-only fallback

- **WHEN** every entry is a prerelease
- **THEN** the selector returns the highest entry

#### Scenario: Unparseable entry is skipped

- **WHEN** the list ends with an entry SemVer cannot parse
- **THEN** the selector returns the highest parseable stable entry instead

### Requirement: The stable-preferring version selector is not exported and is not the compatibility gate's predecessor selection
The selector SHALL NOT be exported: template resolution is its only caller. This selector is NOT the compatibility gate's predecessor selection, and SHALL NOT ship in the compatibility package. (Requirement: The CLI provides a stable-preferring version selector over a published-version list that is pure)

#### Scenario: The selector is not exported and is not in the compatibility package

- **WHEN** the stable-preferring version selector is looked up
- **THEN** the selector SHALL NOT be exported, template resolution is its only caller, and it SHALL NOT ship in the compatibility package

### Requirement: The publish gate's predecessor is found by the literal backward scan of the published history
Enhancement 0011 ("Predecessor selection is D9's literal rule: a backward scan of the published history, prereleases included") (amending the earlier predecessor decision of the same enhancement) settled that the publish gate's predecessor SHALL be found by the literal rule: scan published versions strictly below the effective version, same major, prereleases included, newest first, each member resolving against the newest build carrying its `name` at its `apiVersion`.

#### Scenario: The gate does not use it

- **WHEN** the publish gate selects the predecessor build to compare against
- **THEN** it applies the literal backward scan of the compatibility gate, not this selector

### Requirement: The literal predecessor rule is implemented in `internal/publish` over the registry's version enumeration
Enhancement 0011 ("Predecessor selection is D9's literal rule: a backward scan of the published history, prereleases included") (amending the earlier predecessor decision of the same enhancement) settled that the publish gate's predecessor SHALL be found by the literal rule, implemented in `internal/publish` over the registry's version enumeration.

#### Scenario: The literal predecessor rule is implemented in `internal/publish`

- **WHEN** the publish gate finds the predecessor build
- **THEN** the literal rule SHALL be implemented in `internal/publish` over the registry's version enumeration

### Requirement: The documentation of the selector states how it differs from the predecessor selection of the publish gate
A stable-preferring selector coincides with that rule only on a prerelease-only history and would miss breaks that prerelease pinners (blessed by enhancement 0010: a subscription names exactly one catalog build) can see. The documentation of the selector SHALL state this distinction, because the two were confused once already. (Requirement: The publish gate's predecessor is found by the literal backward scan of the published history)

#### Scenario: The selector documentation states the distinction

- **WHEN** the documentation of the selector is read
- **THEN** it SHALL state that this selector is not the compatibility gate's predecessor selection, because the two were confused once already
