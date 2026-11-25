## RENAMED Requirements

- FROM: `### Requirement: Non-test cli code reads error text only in four listed cases and a test fails on any other error-text predicate`
- TO: `### Requirement: Non-test cli code reads error text only in three listed cases and a test fails on any other error-text predicate`

## MODIFIED Requirements

### Requirement: Non-test cli code reads error text only in three listed cases and a test fails on any other error-text predicate

Non-test cli code MAY read error text only in these cases, none of which is a fetch or resolution failure:
- the "module is not tidy" answer of the embedded `cue mod tidy`;
- dropping CUE's disjunction summary line from grouped validation output;
- splitting the library's duplicate-identities message into a header and its rows.
A test SHALL fail when non-test cli code gains an error-text predicate outside that list.

**Source**: enhancement 0021 ("A beta line reaches GA only when its exit criteria hold").

#### Scenario: A new text match is refused

- **WHEN** non-test cli code tests an error's message with a string predicate outside the listed
  cases
- **THEN** the guard test fails and names the file and line

#### Scenario: A listed text read is allowed

- **WHEN** non-test cli code splits the library's duplicate-identities message into a header and its rows
- **THEN** it MAY read the error text, because that is one of the listed cases and none of the listed cases is a fetch or resolution failure
