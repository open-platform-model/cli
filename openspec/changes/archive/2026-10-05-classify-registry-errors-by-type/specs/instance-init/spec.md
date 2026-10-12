## ADDED Requirements

### Requirement: Only a registry that gave no response counts as unreachable

While `opm instance init` acquires the module archive and while it resolves the staged package's dependency closure, a registry failure SHALL count as unreachable (exit 3) when no HTTP response was received: a refused connection, a DNS or TLS failure, or a timeout. A registry answer that a dependency of the staged package is not held SHALL NOT count as unreachable: init exits 1 naming the failure. Reading the module's own module file, between resolution and acquire, stays exit 3 for every failure. Source: enhancement 0016 ("`opm instance init` mirrors `opm module init`, takes a major-free module path, and floats to the highest core-compatible major when no version is given"). The CLI SHALL take the unreachable decision from the library's typed fetch classification and SHALL NOT match the registry error's message text itself. Source: enhancement 0021 ("A beta line reaches GA only when its exit criteria hold").

#### Scenario: A dependency the registry does not hold

- **WHEN** the registry answers that a dependency of the staged package is not held
- **THEN** init exits 1, not 3, and nothing is left at the target path or beside it
