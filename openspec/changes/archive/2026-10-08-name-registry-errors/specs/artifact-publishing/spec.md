## MODIFIED Requirements

### Requirement: Exit codes

Publish SHALL exit 0 on success, 2 on any refusal, 3 when a registry operation fails (the core-schema fetch, the already-published lookup, or the push), and 1 on unexpected failure. No consumer contract is made on distinguishing refusal causes by code. When the already-published lookup is what failed, the plan and every locally-derived refusal SHALL still print before the registry error: the author's fixable problems are never hidden behind a failed registry operation, and a refusal-free plan renders an incomplete verdict rather than a GO.

A failed registry operation SHALL be named by its cause, and the message SHALL keep the registry operation and the registry's or the transport's own error text:

- no HTTP response at all (a refused connection, a DNS or TLS failure, a timeout) SHALL read "registry unreachable";
- a 401 or 403 answer that reaches the cli as such SHALL read as refused credentials (authentication or permission), SHALL NOT read "unreachable", and SHALL point to `opm registry login`, naming the registry host the operation was routed to when it is known;
- any other failure answer SHALL read "registry operation failed" and SHALL NOT read "unreachable".

A 403 answer to the already-published lookup is not such a failure: the registry client reads it as "this module is not published", so the lookup passes and the push is where a refusal shows. One refusal is not yet named: a registry that hands out bearer tokens and whose token endpoint answers 403 is reported as "registry unreachable" on the push, with the 403 in the text, until the registry error classification the cli builds on reads that form as an answer.

When the core-schema fetch fails and no registry is configured (no `--registry` flag, no `OPM_REGISTRY`, no `registry` in the config file, and no `CUE_REGISTRY` in the environment), publish SHALL exit 2 with a message that says no registry is configured, keeps the cause, and points to `opm config init`.

#### Scenario: Connectivity is not a refusal

- **WHEN** the registry cannot be reached for the already-published lookup
- **THEN** the command exits 3 and the message names the registry operation that failed

#### Scenario: A refused credential is not called unreachable

- **WHEN** the registry answers the already-published lookup with 401, or the push with 401 or 403
- **THEN** the command exits 3
- **AND** the message says the registry refused the credentials, names the registry operation, carries the registry's own error text and points to `opm registry login <host>`
- **AND** the message does not contain "unreachable"

#### Scenario: A forbidden lookup reads as not published

- **WHEN** the registry answers the already-published lookup with 403
- **THEN** the lookup reports no published version and no error

#### Scenario: Another registry answer is not called unreachable

- **WHEN** the registry answers the already-published lookup with 429 or 503
- **THEN** the command exits 3 and the message says "registry operation failed" with the registry's own error text
- **AND** the message does not contain "unreachable"

#### Scenario: No registry is configured

- **WHEN** publish runs with no `--registry`, no `OPM_REGISTRY`, no `registry` in the config file and no `CUE_REGISTRY`, and the core schema cannot be loaded
- **THEN** the command exits 2
- **AND** the message says no registry is configured and points to `opm config init`
