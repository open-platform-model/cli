# Capability: artifact-publishing

## Purpose

The identity-driven publish pipeline behind `opm module publish` and `opm catalog publish` (enhancement 0011, slice cli-publish-pipeline). Coordinates derive from the artifact's own committed identity — never composed from parts, never written back to match a coordinate — gates refuse in a fixed, actionable shape, the resolved plan doubles as the dry run, and what is published is exactly the committed tree.

## Requirements

### Requirement: One pipeline, two entry points

`opm module publish [dir]` and `opm catalog publish [dir]` SHALL share one implementation: load the artifact tree and its `identity/` subpackage, validate the identity package by unifying it against core's `#IdentityPackage` (surfacing CUE's own diagnostic — never a hand-rolled comparison), derive the registry coordinates by reading and splitting the declared `modulePath` (never composing from parts, never editing the artifact to match a coordinate), run the gates, and push via CUE's module machinery with credentials from the same resolver chain reads use. What is published SHALL be exactly the committed tree — no copied build directory, no generated files in the pushed bytes.

#### Scenario: Clean artifact publishes at its declared version

- **WHEN** a conformant artifact with concrete identity is published
- **THEN** the plan prints, the push targets `<registryRepo>:v<Version>`, and the command exits 0

#### Scenario: Identity is validated by unification

- **WHEN** the identity package has a renamed, mistyped, or missing required field
- **THEN** publish refuses with CUE's unification error, carrying file and line
- **AND** no procedural expected-versus-found message is printed

#### Scenario: A tree that does not load surfaces the load error

- **WHEN** the artifact tree fails to load
- **THEN** publish surfaces the load error verbatim and reports no identity-derived refusal

### Requirement: Gates and refusal catalog

Publish SHALL evaluate every evaluable gate, accumulate all refusals into one list, and refuse with exit code 2 when any gate fails. Each refusal SHALL follow the fixed shape condition → evidence → consequence → action, print disagreeing values in aligned columns naming the file (and line where available) that declares each, and offer a runnable command as the action. One cause SHALL yield one refusal. The gates: identity concreteness (a defaulted field counts as concrete at its default; an absent field or file is never publishable); `metadata` ↔ identity-package derivation; `cue.mod` `module:` ↔ declared path agreement (no `cue.mod` at all refuses, pointing at `opm mod init` — publish never creates or edits `cue.mod`); `source: {kind: "self"}` presence; tag equals the effective declared version and the tag major equals the path major; the module package name equals `metadata.name`; namespace and kind-segment checks on owned domains only; local-override presence; already-published tag.

#### Scenario: Multiple defects reported in one pass

- **WHEN** an artifact has both a coordinate disagreement and tag/version skew
- **THEN** one invocation reports both refusals

#### Scenario: Already published is always a refusal

- **WHEN** the registry already holds the resolved tag
- **THEN** publish refuses, naming the artifact, the version, and the bump command
- **AND** no flag or mode turns this condition into a success

#### Scenario: Unowned domains are not policed

- **WHEN** a third-party artifact under its own domain is published
- **THEN** no namespace or kind-segment gate refuses it

### Requirement: Local overrides are never honored; modules may waive the gate

Publish SHALL always resolve dependencies as published — `cue.mod/local-module.cue` replacements are ignored, never baked in. When the file is present, publish SHALL report each replacement alongside the registry version that supersedes it and refuse. `opm module publish --skip-override-check` SHALL waive the gate (not the ignoring); `opm catalog publish` SHALL have no waiver, and its check SHALL be on file presence regardless of whether replacements resolve.

#### Scenario: Catalog with override always refused

- **WHEN** a catalog tree carries `cue.mod/local-module.cue`
- **THEN** publish refuses even when the flag is supplied

#### Scenario: Module flag waives the gate only

- **WHEN** a module tree carries the file and `--skip-override-check` is supplied
- **THEN** publish proceeds and the published artifact resolves against published dependencies

### Requirement: `--version` fills or asserts, never overwrites

`--version` on either command SHALL fill an open identity `Version` (writing the working tree through the schema-fixed-path rewrite) or assert an already-concrete one; a disagreement with a concrete value SHALL refuse, naming both values and offering `version set`. An absent identity field or file SHALL NOT be fillable by the flag.

#### Scenario: Fill then publish

- **WHEN** the identity `Version` is open and `--version 1.3.0` is supplied
- **THEN** the working tree's identity file is updated to `1.3.0` and the publish proceeds at `v1.3.0`

#### Scenario: Assertion failure

- **WHEN** the identity declares `2.4.0` and `--version 2.4.1` is supplied
- **THEN** publish refuses printing both values and the `version set` action

### Requirement: The plan is the dry run

Every publish SHALL print the resolved plan — kind, declared path, `cue.mod` path, registry repository, major, tag and its source, per-field identity state, and the override-gate outcome — before any push. `--dry-run` SHALL stop after the plan, having evaluated every gate including the already-published lookup, exiting 0 on GO and 2 on REFUSED. There SHALL be no separate check mode.

#### Scenario: Dry run surfaces refusals

- **WHEN** `--dry-run` is supplied on a defective artifact
- **THEN** the plan prints REFUSED with the accumulated refusal list and the command exits 2

### Requirement: Exit codes

Publish SHALL exit 0 on success, 2 on any refusal, 3 when a registry cannot be reached (for the core-schema fetch, the already-published lookup, or the push), and 1 on unexpected failure. No consumer contract is made on distinguishing refusal causes by code. When the already-published lookup is what failed, the plan and every locally-derived refusal SHALL still print before the connectivity error — the author's fixable problems are never hidden behind an unreachable registry, and a refusal-free plan renders an incomplete verdict rather than a GO.

#### Scenario: Connectivity is not a refusal

- **WHEN** the registry cannot be reached for the already-published lookup
- **THEN** the command exits 3 and the message names the registry operation that failed

### Requirement: The compatibility gate

`opm catalog publish` SHALL, for every member of the tree at beta or GA level (transformers structurally excluded; alpha members exempt), compare the member against the last published build that shipped a member of that name at that apiVersion — found by scanning published versions strictly below the effective version, within the same major, prereleases included, newest first — under the additive-only rule via the library comparator, over provenance-stripped definition values. A member no published build has carried SHALL pass. Violations SHALL refuse as refusal 9: a header naming the member, its apiVersion, and the compared-against coordinate, followed by path-located violation lines. The gate makes a beta-or-GA name-and-apiVersion key a permanent claim on its own history: an incompatible re-introduction after a removal SHALL be refused identically.

#### Scenario: Field removal refused against the true predecessor

- **WHEN** a beta member removes a field relative to the newest published build that carried it
- **THEN** publish refuses with the violation path and the predecessor coordinate

#### Scenario: Prerelease predecessors count

- **WHEN** the newest build carrying the member is a prerelease newer than the latest stable
- **THEN** the comparison runs against that prerelease, not the stable

#### Scenario: Remove-then-readd refused

- **WHEN** a member absent from the immediate predecessor was shipped incompatibly-reshaped, and an older build carried it
- **THEN** the scan finds the older build and refuses

#### Scenario: apiVersion bump escapes

- **WHEN** a member ships at an apiVersion no build has ever carried
- **THEN** it passes the compatibility gate trivially

### Requirement: Member and posture gates

`opm catalog publish` SHALL unify every member of the tree — all four kinds, all levels — against core's member gate with concrete evaluation, surfacing CUE's error as the refusal (wrong-kind or wrong-depth filing, stale authored fqn, stale catalogVersion, missing declaredAPIVersion). Every trait's `optional` field SHALL be unified against core's posture gate, refusing an unstated or pinned posture. The alpha exemption applies to the compatibility gate only.

#### Scenario: Filing error caught structurally

- **WHEN** a contract member files under the wrong kind or depth
- **THEN** publish refuses with CUE's own conflict diagnostics

#### Scenario: Unstated posture caught only concretely

- **WHEN** a trait states no `optional` posture
- **THEN** publish refuses — the incomplete value is the finding

### Requirement: Connectivity across the member walk

A transport failure during predecessor enumeration or package loading SHALL abort as a connectivity error with no partial verdict — the artifact was never judged. A package absent at a given published version is the scan's negative signal, never an error.

#### Scenario: Mid-walk failure renders no verdict

- **WHEN** the registry becomes unreachable after some members have compared clean
- **THEN** the command exits 3 and no GO or REFUSED verdict is printed
