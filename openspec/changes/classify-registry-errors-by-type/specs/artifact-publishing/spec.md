## MODIFIED Requirements

### Requirement: Connectivity across the member walk

A transport failure during predecessor enumeration or package loading SHALL abort as a connectivity error with no partial verdict — the artifact was never judged. A package absent at a given published version is the scan's negative signal, never an error. Absent SHALL mean that the registry does not hold the probed version, or that the probed version holds no CUE files for the package. A dependency of the predecessor build that the registry does not hold SHALL abort as a connectivity error (exit 3). Source: 0011:D9. The CLI SHALL take these decisions from the library's typed fetch classification and from the fetched build's contents, and SHALL NOT match the error's message text itself. Source: 0021:D8:R12.

#### Scenario: Mid-walk failure renders no verdict

- **WHEN** the registry becomes unreachable after some members have compared clean
- **THEN** the command exits 3 and no GO or REFUSED verdict is printed

#### Scenario: Package absent at a published version

- **WHEN** a predecessor version is published but holds no CUE files for a member's package
- **THEN** the walk moves on to the next older version and reports no error for that version

#### Scenario: A predecessor's dependency is not held

- **WHEN** a predecessor build is published but the registry does not hold one of its dependencies
- **THEN** the command exits 3 and no GO or REFUSED verdict is printed
