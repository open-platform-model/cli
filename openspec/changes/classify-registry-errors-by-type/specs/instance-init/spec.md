## ADDED Requirements

### Requirement: Only a registry that gave no response counts as unreachable

While `opm instance init` acquires the module and while it resolves the staged package's dependency closure, a registry failure SHALL count as unreachable (exit 3) exactly when no HTTP response was received: a refused connection, a DNS or TLS failure, or a timeout. A registry that answered, whatever it answered (the module or a dependency is not held, the credentials are refused, a rate limit, a server error), SHALL NOT count as unreachable, and init SHALL exit 1 naming the failure. The CLI SHALL take this decision from the library's typed fetch classification and SHALL NOT match the error's message text itself. Source: 0021:D8:R12.

#### Scenario: Refused connection while resolving dependencies

- **WHEN** the module was acquired, and the registry refuses the connection while the staged package's dependency closure is resolved
- **THEN** init exits 3 naming the registry, and nothing is left at the target path or beside it

#### Scenario: A dependency the registry does not hold

- **WHEN** the registry answers that a dependency of the staged package is not held
- **THEN** init exits 1, not 3, and nothing is left at the target path or beside it

#### Scenario: A server error while acquiring the module

- **WHEN** the registry answers the module fetch with status 503 after the version was resolved
- **THEN** init exits 1, not 3
