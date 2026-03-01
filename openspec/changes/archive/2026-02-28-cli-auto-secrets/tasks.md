## 1. Test Fixture

- [x] 1.1 Create `tests/fixtures/valid/secrets-module/cue.mod/module.cue` with `opmodel.dev@v1` v1.0.8 dependency
- [x] 1.2 Create `tests/fixtures/valid/secrets-module/module.cue` with `#Secret` fields in `#config` (two fields sharing `$secretName: "db-creds"`, one with `$secretName: "api-keys"`)
- [x] 1.3 Create `tests/fixtures/valid/secrets-module/values.cue` providing `#SecretLiteral` values for all secret fields
- [x] 1.4 Verify fixture loads correctly with `cue vet -c=false ./...` in the fixture directory

## 2. Core Implementation

- [x] 2.1 Create `internal/builder/autosecrets.go` with `readAutoSecrets()` function — reads `autoSecrets` via `cue.ParsePath("autoSecrets")`, returns `(cue.Value, bool)`
- [x] 2.2 Implement `loadSecretsSchema()` — loads `opmodel.dev/resources/config@v1` via `load.Instances`, extracts `#Secrets`
- [x] 2.3 Implement `buildOpmSecretsComponent()` — FillPath chain (`metadata.name`, `spec.secrets.*.data`), wraps in map, calls `component.ExtractComponents()`
- [x] 2.4 Implement `injectAutoSecrets()` — orchestrates read, collision check, build, inject
- [x] 2.5 Add `injectAutoSecrets()` call to `builder.go` between step 7b and step 8

## 3. Tests

- [x] 3.1 Create `internal/builder/autosecrets_test.go` with `requireRegistry` gating
- [x] 3.2 Test: `TestInjectAutoSecrets_WithSecrets` — secrets-module fixture produces `opm-secrets` component with correct `#resources` FQN and `spec.secrets` structure
- [x] 3.3 Test: `TestInjectAutoSecrets_NoSecrets` — real_module fixture returns components unchanged, no `opm-secrets`
- [x] 3.4 Test: `TestInjectAutoSecrets_NameCollision` — pre-insert dummy `opm-secrets` into components map, expect error containing "reserved"

## 4. Validation

- [x] 4.1 Run `task fmt` — all files formatted
- [x] 4.2 Run `task lint` — golangci-lint passes (no new issues)
- [x] 4.3 Run `task test:unit` — all unit tests pass
