## MODIFIED Requirements

### Requirement: `opm instance diff` exits non-zero and prints no `No differences found` when a failure exists
When at least one failure exists, the `opm instance diff` command SHALL NOT print `No differences found` and SHALL exit non-zero: with the code of the failures when all of them have the same class (4 for a denied read, 3 for a connectivity failure, 1 for any other failure, as `cmdutil.ExitCodeFromK8sError` classes them), and 1 when the classes differ. (Requirement: `opm instance diff` treats a rendered resource it could not read or compare as a failure)

#### Scenario: Every rendered resource is unreadable

- **WHEN** `opm instance diff` runs and the read of every rendered resource fails with Forbidden
- **THEN** the command SHALL print one error per resource naming it
- **AND** SHALL NOT print `No differences found`
- **AND** SHALL exit 4

#### Scenario: Failures of two classes

- **WHEN** one read fails with Forbidden and another with an internal server error
- **THEN** the command SHALL exit 1
