## MODIFIED Requirements

### Requirement: One pipeline, two entry points

`opm module publish [dir]` and `opm catalog publish [dir]` SHALL share one implementation: load the artifact tree and its `identity/` subpackage, validate the identity package by unifying it against core's `#IdentityPackage` (surfacing CUE's own diagnostic — never a hand-rolled comparison), derive the registry coordinates by reading and splitting the declared `modulePath` (never composing from parts, never editing the artifact to match a coordinate), run the gates, and push via CUE's module machinery with credentials from the same resolver chain reads use. What is published SHALL be exactly the committed tree — no copied build directory, no generated files in the pushed bytes.

For a module, publish SHALL additionally load the tree through the kernel's module loader, the same loader `opm module build` and the controller use, and SHALL refuse with the loader's own error when the loader refuses. Publish SHALL NOT judge a module loadable by any test the kernel does not apply.

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
