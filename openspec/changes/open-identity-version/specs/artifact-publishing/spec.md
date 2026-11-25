## ADDED Requirements

### Requirement: The identity check treats an open field as fillable and checks the identity as filled

An open identity field is not a conformance failure; an open field that `--version` does not fill is refused by the gate for open fields. When `--version` fills an open `Version`, publish SHALL run the conformance check on the identity as filled, before any push and in the dry run. The check SHALL tell an open field from a wrong one without reading error text.

#### Scenario: An open Version filled by the flag passes conformance

- **WHEN** the identity declares `Version: string` under core's real `#IdentityPackage` and `opm module publish --version 0.1.0 --dry-run` runs on a `@v0` module path
- **THEN** the plan reports no conformance refusal and shows `Version` as open and filled by `--version`

#### Scenario: A filled Version is checked against the path major

- **WHEN** the identity declares `Version: string` and `opm module publish --version 1.0.0 --dry-run` runs on a `@v0` module path
- **THEN** publish refuses, naming the version major `v1` and the path major `v0`, before any push
