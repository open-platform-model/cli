## Why

Two first-hour failures name the wrong cause. With no registry configured, `opm module vet` and the publish commands fail as "loading core schema" (exit 3) and never mention `opm config init`. And the publish commands label every failed registry lookup or push "registry unreachable", so a refused credential (401, 403) reads as a network problem and never mentions `opm registry login`.

## What Changes

- When the core schema cannot be loaded and no registry is configured (no `--registry`, no `OPM_REGISTRY`, no `registry` in the config file, and no `CUE_REGISTRY` in the environment), `opm module vet`, `opm module publish` and `opm catalog publish` say "no registry is configured", keep the cause, point to `opm config init`, and exit 2 instead of 3.
- The publish commands classify a failed core-schema fetch, already-published lookup and push with the typed registry classification the cli already uses (`internal/cuemod/connectivity.go`): only no response at all is "registry unreachable"; a 401 or 403 is "the registry refused the credentials" and points to `opm registry login`; any other registry answer is "registry operation failed". The registry's own error text stays in the message in every case.
- The publish exit codes do not change for a registry failure: every one of the three classes still exits 3, because the artifact was never judged.
- Help text and the publish pages name the three classes.

Not changed: the `CUE_REGISTRY` fallback (a run with only `CUE_REGISTRY` set resolves as it does today, and a warm CUE module cache still serves a run with nothing configured); the catalog compatibility walk and `opm catalog registry check`, whose registry failures keep their pinned classification; `opm module vet`'s exit 3 for every other failed core-schema fetch; flags.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `artifact-publishing`: the exit-code requirement names the three registry failure classes and the no-registry case.
- `mod-vet`: the exit-code and registry-loading requirements gain the no-registry case (exit 2, `opm config init`).

## Impact

- Commands: `opm module vet`, `opm module publish`, `opm catalog publish`.
- Packages: `internal/config` (no-registry test and error), `internal/cuemod` (an unauthorized reader beside the connectivity one), `internal/publish` (registry failure classification), `internal/cmdutil` (publish exit mapping and hints), `internal/cmd/module` (vet's schema error, help text), `internal/cmd/catalog` (help text), `docs/site`.
- SemVer: PATCH after GA (error text, and one exit code that moves from 3 to 2 for a case that never succeeded). Beta ships it as the next `beta.N`. Not marked breaking: no successful run changes, and the spec makes no contract on telling causes apart by code beyond the classes it lists.
- No new dependency, flag or command.
