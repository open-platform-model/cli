## ADDED Requirements

### Requirement: Instance diff fails when an object could not be read

`opm instance diff` SHALL treat each of the following as a failure: a rendered resource whose live read fails with an error other than NotFound, a rendered resource whose comparison with the live object fails, an instance record whose read fails with an error other than NotFound, and a tracked resource that orphan detection could not read. For each failure the command SHALL print an error that names the object and carries the cause; for a failed record read it SHALL say that orphan detection did not run. The command SHALL still compare the objects it can read and SHALL print the differences it found. When at least one failure exists, the command SHALL NOT print `No differences found` and SHALL exit non-zero: with the code of the failures when all of them have the same class (4 when the API server denied the read with Forbidden or Unauthorized, 3 on a server timeout or service unavailable, 1 for any other failure), and 1 when the classes differ. A NotFound answer for a rendered resource SHALL still mean a new resource. With no failure the command SHALL behave as before.

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

#### Scenario: Unreadable instance record

- **WHEN** the read of the instance's `ModuleInstance` record fails with Forbidden
- **AND** every rendered resource equals its live object
- **THEN** the command SHALL print an error naming the record and saying that orphan detection did not run
- **AND** SHALL NOT print `No differences found`
- **AND** SHALL exit 4

#### Scenario: Failures of two classes

- **WHEN** one read fails with Forbidden and another with an internal server error
- **THEN** the command SHALL exit 1

#### Scenario: Nothing failed and nothing differs

- **WHEN** every rendered resource is read and equals its live object
- **THEN** the command SHALL print `No differences found` and exit 0
