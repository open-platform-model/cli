## MODIFIED Requirements

### Requirement: One pipeline, two entry points

`opm module publish [dir]` and `opm catalog publish [dir]` SHALL share one implementation: load the artifact tree and its `identity/` subpackage, validate the identity package by unifying it against core's `#IdentityPackage` (surfacing CUE's own diagnostic — never a hand-rolled comparison), derive the registry coordinates by reading and splitting the declared `modulePath` (never composing from parts, never editing the artifact to match a coordinate), run the gates, and push via CUE's module machinery with credentials from the same resolver chain reads use. What is published SHALL be exactly the committed tree — no copied build directory, no generated files in the pushed bytes.

For a module, publish SHALL additionally load the tree through the kernel's module loader, the same loader `opm module build` and the operator use, and SHALL refuse with the loader's own error when the loader refuses. Publish SHALL NOT judge a module loadable by any test the kernel does not apply.

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

#### Scenario: A module the kernel refuses is refused by publish

- **WHEN** a module tree loads as CUE but the kernel's module loader refuses it (for example `metadata.version` declared as a defaulted disjunction, or `kind` not `"Module"`)
- **THEN** `opm module publish` and `opm module publish --dry-run` refuse with exit code 2, quoting the loader's error and naming the loader's failure class
- **AND** `opm module vet` reports the same refusal
- **AND** the same tree with the defect corrected passes the gate

### Requirement: Gates and refusal catalog

Publish SHALL evaluate every evaluable gate, accumulate all refusals into one list, and refuse with exit code 2 when any gate fails. Each refusal SHALL follow the fixed shape condition → evidence → consequence → action, print disagreeing values in aligned columns naming the file (and line where available) that declares each, and offer a runnable command as the action. One cause SHALL yield one refusal. The gates: kernel loadability (modules only: the kernel's module loader accepts the tree; its refusal is surfaced verbatim and is the only refusal that cause yields); identity concreteness (a defaulted field counts as concrete at its default for tag derivation; an absent field or file is never publishable); `metadata` ↔ identity-package derivation; `cue.mod` `module:` ↔ declared path agreement (no `cue.mod` at all refuses, pointing at `opm mod init` — publish never creates or edits `cue.mod`); `source: {kind: "self"}` presence; tag equals the effective declared version and the tag major equals the path major; the module package name equals `metadata.name`; namespace and kind-segment checks on owned domains only; local-override presence; already-published tag.

#### Scenario: Multiple defects reported in one pass

- **WHEN** an artifact has both a coordinate disagreement and tag/version skew
- **THEN** one invocation reports both refusals

#### Scenario: Kernel refusal is one refusal

- **WHEN** a module's identity `Version` is a defaulted disjunction
- **THEN** publish reports exactly one refusal for it, the kernel loader's, and the identity gates still derive the tag from the default so the rest of the plan prints

#### Scenario: Already published is always a refusal

- **WHEN** the registry already holds the resolved tag
- **THEN** publish refuses, naming the artifact, the version, and the bump command
- **AND** no flag or mode turns this condition into a success

#### Scenario: Unowned domains are not policed

- **WHEN** a third-party artifact under its own domain is published
- **THEN** no namespace or kind-segment gate refuses it
