## ADDED Requirements

### Requirement: The identity check treats an open field as fillable and checks the identity as filled

An open identity field is not a conformance failure; an open field that `--version` does not fill is refused by the gate for open fields. When `--version` fills an open `Version`, publish SHALL run the conformance check on the identity as filled, before any push and in the dry run. The check SHALL tell an open field from a wrong one without reading error text.

#### Scenario: An open Version filled by the flag passes conformance

- **WHEN** the identity declares `Version: string` under core's real `#IdentityPackage` and `opm module publish --version 0.1.0 --dry-run` runs on a `@v0` module path
- **THEN** the plan reports no conformance refusal and shows `Version` as open and filled by `--version`

#### Scenario: A filled Version is checked against the path major

- **WHEN** the identity declares `Version: string` and `opm module publish --version 1.0.0 --dry-run` runs on a `@v0` module path
- **THEN** publish refuses, naming the version major `v1` and the path major `v0`, before any push

## MODIFIED Requirements

### Requirement: Publish loads the artifact tree and validates its identity package by unification

`opm module publish [dir]` and `opm catalog publish [dir]` SHALL share one implementation: load the artifact tree and its `identity/` subpackage, and unify the identity package with core's `#IdentityPackage`. A conflict SHALL surface CUE's own diagnostic, never a hand-rolled comparison. An absent `ModulePath` or `Version` SHALL be refused by name, because plain unification does not report it.

#### Scenario: Identity is validated by unification

- **WHEN** the identity package has a renamed or mistyped field
- **THEN** publish refuses with CUE's unification error, carrying file and line
- **AND** no procedural expected-versus-found message is printed
- **WHEN** the identity package does not declare `ModulePath` or `Version`
- **THEN** publish refuses, naming the absent field and `identity/identity.cue`

#### Scenario: A tree that does not load surfaces the load error

- **WHEN** the artifact tree fails to load
- **THEN** publish surfaces the load error verbatim and reports no identity-derived refusal
