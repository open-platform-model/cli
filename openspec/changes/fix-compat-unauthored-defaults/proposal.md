## Why

The publish compatibility gate refuses a legal catalog change with a false `default changed`. Making catalog_opm's role subjects optional (`subjects!: [...#RoleSubjectSchema] & [_, ...]` to `subjects?: ...`) is refused as `spec.role.subjects default changed ([{name!: string}] -> [{name!: string}])`, the same text on both sides. CUE gives every open list an implicit default (the list closed at its fixed elements), and `checkDefaults` compares it as if an author had written a `*`. No catalog spelling avoids it and the gate has no override, so catalog_opm's planned `add-subjectless-roles` change (the opm-operator module's unbound ClusterRoles) cannot publish until the cli is fixed and released.

## What Changes

- The default rule compares only authored defaults: a default marked with `*` somewhere in the value's disjunction. The implicit default CUE gives a plain open list is no default for the rule, on either side.
- A required-to-optional marker change on an open list with no authored default reports nothing (it is none of the violation kinds; the made-required rule only fires in the other direction).
- A real change is still caught: an authored list default that changes or is removed, and an open list whose domain narrows (`[...string]` to `[...string] & [_, ...]` keeps its `domain narrowed`, and loses the spurious `default changed` it reported beside it).
- Unit tests reproduce the catalog case and the `[...string] & [_, ...]` probe, and pin the negative cases.

SemVer class: PATCH (a gate bug fix; it removes a false refusal and loosens no rule 0010:D27 names). Ships in the next beta.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `catalog-compatibility`: the Compatibility Comparison requirement states that only authored defaults are compared, with scenarios for the implicit open-list default.

## Impact

- Code: `internal/compat/compat.go` (`checkDefaults` and one helper), `internal/compat/compat_test.go`.
- Consumers: `opm module publish`, `opm catalog publish` and `opm catalog registry check --compat` stop refusing optional-marker changes on open lists. catalog_opm `add-subjectless-roles` is gated on a cli release carrying this fix plus its `.opm-cli-version` bump.
- Known gap left as is: `required()` still treats a regular open-list field as defaulted through the same implicit default, so a newly added regular `xs: [...T] & [_, ...]` is not reported as a strict addition. Changing that would add refusals; it is its own change.
