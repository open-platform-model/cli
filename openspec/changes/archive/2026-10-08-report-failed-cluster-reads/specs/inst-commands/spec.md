## ADDED Requirements

### Requirement: Instance diff fails when a rendered resource could not be read

`opm instance diff` SHALL treat a rendered resource whose live read fails with an error other than NotFound, and a rendered resource whose comparison with the live object fails, as a failure. For each failure the command SHALL print an error that names the resource and carries the cause. The command SHALL still compare the resources it can read and SHALL print the differences it found. When at least one failure exists, the command SHALL NOT print `No differences found` and SHALL exit non-zero: with the code of the failures when all of them have the same class (4 when the API server denied the read with Forbidden or Unauthorized, 3 on a server timeout or service unavailable, 1 for any other failure), and 1 when the classes differ. A NotFound answer for a rendered resource SHALL still mean a new resource. With no failure the command SHALL behave as before. A tracked resource that orphan detection could not read stays a warning that does not change the exit code.

#### Scenario: Every rendered resource is unreadable

- **WHEN** `opm instance diff` runs and the read of every rendered resource fails with Forbidden
- **THEN** the command SHALL print one error per resource naming it
- **AND** SHALL NOT print `No differences found`
- **AND** SHALL exit 4

#### Scenario: One unreadable resource beside a modified one

- **WHEN** one rendered resource differs from its live object
- **AND** the read of another rendered resource fails with an internal server error
- **THEN** the command SHALL print the difference of the first resource
- **AND** SHALL print an error naming the second resource
- **AND** SHALL exit 1

#### Scenario: Failures of two classes

- **WHEN** one read fails with Forbidden and another with an internal server error
- **THEN** the command SHALL exit 1

#### Scenario: Unreadable tracked resource that is not rendered

- **WHEN** every rendered resource is read
- **AND** orphan detection cannot read a tracked resource that is not rendered
- **THEN** the command SHALL warn that orphan detection could not check it
- **AND** the exit code SHALL NOT change

#### Scenario: Nothing failed and nothing differs

- **WHEN** every rendered resource is read and equals its live object
- **THEN** the command SHALL print `No differences found` and exit 0
