## ADDED Requirements

### Requirement: Shared CUE string-map extraction helper

The `loader` package SHALL provide an internal `extractCUEStringMap` function that
extracts a `map[string]string` from a named field of a CUE value.

The function SHALL accept:
- A `cue.Value` to read from
- A field name string

The function SHALL return `(map[string]string, error)`.

The function MUST implement the following behavior:
- Look up the named field on the provided CUE value.
- If the field does not exist or is not a struct, return an empty (non-nil) map and
  no error (absence is not an error).
- Iterate the field's subfields using `Fields()`.
- For each subfield: extract the unquoted selector as the key and the string value
  as the value.
- If a subfield value cannot be decoded as a string, return a wrapped error indicating
  the field path and CUE error.
- Return the populated map.

The function SHALL be used by `loader/module.go`, `loader/provider.go`, and
`builder/builder.go` in place of their individually inlined copies of this logic.

#### Scenario: Field exists with string entries

- **WHEN** the CUE value has a struct field containing only string-valued subfields
- **THEN** `extractCUEStringMap` returns a `map[string]string` with all key-value pairs

#### Scenario: Field is absent from the CUE value

- **WHEN** the named field does not exist on the CUE value
- **THEN** `extractCUEStringMap` returns an empty non-nil map and a nil error

#### Scenario: Subfield value is not a concrete string

- **WHEN** a subfield of the named struct is not a concrete string (e.g., still a CUE
  open type or bottom value)
- **THEN** `extractCUEStringMap` returns a non-nil error wrapping the CUE error
