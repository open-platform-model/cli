## 1. Test Fixture

- [ ] 1.1 Create `tests/fixtures/valid/secrets-module/cue.mod/module.cue` with `opmodel.dev@v1` dependency
- [ ] 1.2 Create `tests/fixtures/valid/secrets-module/module.cue` with `#Secret` fields in `#config` (two fields sharing `$secretName: "db-creds"`, one with `$secretName: "api-keys"`)
- [ ] 1.3 Create `tests/fixtures/valid/secrets-module/components.cue` with a web component that wires secrets via `env.from:`
- [ ] 1.4 Create `tests/fixtures/valid/secrets-module/values.cue` providing `#SecretLiteral` values for all secret fields
- [ ] 1.5 Verify fixture loads correctly with `cue vet ./...` in the fixture directory

## 2. Core Implementation

- [ ] 2.1 Create `internal/builder/autosecrets.go` with `readAutoSecrets()` function — reads `_autoSecrets` via `cue.Hid("_autoSecrets", "opmodel.dev/core@v1")`, returns `(cue.Value, bool)`
- [ ] 2.2 Implement `loadSecretsSchema()` — loads `opmodel.dev/resources/config@v1` via `load.Instances`, extracts `#Secrets`
- [ ] 2.3 Implement `buildOpmSecretsComponent()` — FillPath chain (`metadata.name`, `spec.secrets.*.data`), wraps in map, calls `component.ExtractComponents()`
- [ ] 2.4 Implement `injectAutoSecrets()` — orchestrates read, collision check, build, inject
- [ ] 2.5 Add `injectAutoSecrets()` call to `builder.go` between step 7b and step 8

## 3. Tests

- [ ] 3.1 Create `internal/builder/autosecrets_test.go` with `requireRegistry` gating
- [ ] 3.2 Test: `TestInjectAutoSecrets_WithSecrets` — secrets-module fixture produces `opm-secrets` component with correct `#resources` FQN and `spec.secrets` structure
- [ ] 3.3 Test: `TestInjectAutoSecrets_NoSecrets` — multi-values-module fixture returns components unchanged, no `opm-secrets`
- [ ] 3.4 Test: `TestInjectAutoSecrets_NameCollision` — pre-insert dummy `opm-secrets` into components map, expect error containing "reserved"

## 4. Validation

- [ ] 4.1 Run `task fmt` — all files formatted
- [ ] 4.2 Run `task lint` — golangci-lint passes
- [ ] 4.3 Run `task test` — all tests pass (unit + integration)
