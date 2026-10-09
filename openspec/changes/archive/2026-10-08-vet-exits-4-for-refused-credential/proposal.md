## Why

`opm module publish` and `opm catalog publish` exit 4 when the registry refuses the
credentials on the core schema fetch. `opm module vet` fetches the same schema through the
same registry and still exits 3 for that answer, which a pipeline reads as "registry
unreachable, try again". The owner decided to align vet with publish.

## What Changes

- `opm module vet` exits 4, with the `opm registry login <host>` hint, when the registry
  refuses the credentials on the core schema fetch (a 401, and a 403 that reaches the cli as
  a refusal). It reuses the classification publish uses; one shared helper maps the failed
  fetch for both.
- "No registry is configured" keeps exit 2 and an unreachable registry keeps exit 3. The
  message of the exit-3 cases gains the prefix publish prints (`registry unreachable:` or
  `registry operation failed:`).
- The help of `opm module vet` states its exit codes.
- Known limit, pinned by a test: CUE's registry client reports a 403 answer to the schema's
  tag lookup as "not found", so that refusal exits 3, as it does for publish.

A script that treats exit 3 from `module vet` as "bad credentials" now sees 4. The code moves
to its documented meaning (4 is "permission denied"), so this is a fix, not a breaking change.

SemVer class after GA: PATCH. Beta ships it as the next `1.0.0-beta.N`.

## Capabilities

### New Capabilities

### Modified Capabilities

- `mod-vet`: the requirement "mod vet exit codes" gains code 4 for a refused registry
  credential on the core schema fetch.

## Impact

- Command: `opm module vet`. No other command than the two publish commands fetches the core
  schema through this path.
- Packages: `internal/cmdutil` (`CoreSchemaError`), `internal/cmd/module`.
- No new dependency.
