## Why

CUE modules can declare sensitive configuration fields using `schemas.#Secret` in their `#config`. At evaluation time, `#ModuleRelease` auto-discovers these secrets via the `_autoSecrets` hidden field and groups them by `$secretName`/`$dataKey`. However, the CLI currently ignores `_autoSecrets` entirely — it never reads the field and never generates the `opm-secrets` component needed for the transformer pipeline to produce Kubernetes Secret and ExternalSecret resources.

This means module authors who use `#Secret` fields get no secret management output from `opm mod build`. The CUE side is complete (discovery, grouping, schema validation), but the Go side has a gap: reading `_autoSecrets` and injecting the synthesized component before matching.

## What Changes

- The builder SHALL read the `_autoSecrets` hidden field from the evaluated `#ModuleRelease`
- When `_autoSecrets` is non-empty, the builder SHALL construct an `opm-secrets` component by unifying auto-discovered secret data with the `resources/config.#Secrets` schema
- The synthesized component SHALL be injected into the components map before the matching phase
- The component name `opm-secrets` SHALL be reserved — user-defined components with that name SHALL cause a build error
- A new test fixture module with `#Secret` fields SHALL be created for validation

## Capabilities

### New Capabilities

- `auto-secrets-injection`: The builder's ability to read `_autoSecrets` from a `#ModuleRelease`, construct an `opm-secrets` component via FillPath with the `#Secrets` schema, and inject it into the component map for transformer matching

### Modified Capabilities

- `release-building`: The build phase gains a new step (7c) between component extraction and release construction that conditionally injects the auto-secrets component

## Impact

### New Files

- `internal/builder/autosecrets.go` — Auto-secrets reading, component building, and injection logic
- `internal/builder/autosecrets_test.go` — Unit tests for the auto-secrets pipeline
- `tests/fixtures/valid/secrets-module/` — Test fixture module with `#Secret` fields in `#config`

### Modified Files

- `internal/builder/builder.go` — Three-line insertion calling `injectAutoSecrets()` between step 7b and step 8

### Dependencies

**Existing (reused):**

- `cue.Hid()` — For accessing hidden CUE fields
- `load.Instances()` — For loading `opmodel.dev/resources/config@v1`
- `component.ExtractComponents()` — For extracting the synthesized component
- `cue.Value.FillPath()` — For building the component via CUE unification

**CUE catalog (already complete):**

- `core/module_release.cue` `_autoSecrets` field
- `schemas/config.cue` `#AutoSecrets`, `#DiscoverSecrets`, `#GroupSecrets`
- `resources/config/secret.cue` `#Secrets`, `#SecretsResource`

### SemVer Classification

**MINOR** — New capability, no breaking changes, no existing behavior modified. Modules without `#Secret` fields are unaffected.

### Complexity Justification (Principle VII)

**No new flags or commands.** The injection is fully automatic — when `_autoSecrets` is non-empty, the component is created. When empty, nothing changes. This aligns with Principle IV (declarative intent): module authors declare secrets in `#config`, the CLI handles the rest.

**Estimated scope:** ~150-200 LOC in `autosecrets.go`, ~100-150 LOC in tests, ~50 LOC in test fixture. The `builder.go` change is 3 lines.
