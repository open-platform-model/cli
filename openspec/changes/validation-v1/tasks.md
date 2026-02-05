# Tasks: CLI Validation Commands

## Phase 1: Create validator package structure

- [ ] Create `cli/internal/validator/` package
- [ ] Define `ModuleValidator` struct with validation options
- [ ] Define `ConfigValidator` struct
- [ ] Define shared types: `ValidationResult`, `ConfigResult`, `ValidationError`
- [ ] Add unit tests for struct initialization

## Phase 2: Implement module validation pipeline

- [ ] Implement Phase 1: Load & Build
  - [ ] Configure `load.Config` with module root and package
  - [ ] Load instances via `load.Instances()`
  - [ ] Handle load errors with file:line:col
  - [ ] Build value from instance via `ctx.BuildInstance()`
- [ ] Implement Phase 2: Schema validation
  - [ ] Call `value.Validate()` for constraint checking
  - [ ] Unify with `core.#Module` schema
  - [ ] Extract and count components for summary
  - [ ] Extract and count scopes (if present) for summary
- [ ] Implement Phase 3: Concrete validation (optional)
  - [ ] Only run when `--concrete` flag is set
  - [ ] Call `value.Validate(cue.Concrete(true), cue.Final())`
  - [ ] Report non-concrete fields clearly
- [ ] Add integration tests with fixture modules

## Phase 3: Implement config validation pipeline

- [ ] Implement Phase 1: Load & Build
  - [ ] Configure `load.Config` for config directory
  - [ ] Load config CUE module via `load.Instances()`
  - [ ] Handle load errors with file:line:col
  - [ ] Build value from instance
- [ ] Implement Phase 2: Schema validation
  - [ ] Look up `config` field in value
  - [ ] Call `configValue.Validate()` for constraint checking
  - [ ] Validate against `core.#Config` schema (via CUE unification)
- [ ] Add integration tests with fixture configs

## Phase 4: Custom error formatting

- [ ] Create `cli/internal/validator/errors.go`
- [ ] Parse CUE errors to extract `file:line:col` using `cue/errors` package
- [ ] Format errors with Charm lipgloss styling
- [ ] Implement fail-on-end error aggregation
- [ ] Add suggested fixes for common errors:
  - [ ] Missing required files
  - [ ] Invalid field values (RFC-1123 for namespace)
  - [ ] Unresolved imports
  - [ ] Non-concrete values

## Phase 5: Entity summary output

- [ ] Extract module name from `metadata.name`
- [ ] Count components from `#components`
- [ ] Count scopes from `#scopes` (if present)
- [ ] Display success summary with checkmarks (lipgloss)
- [ ] Display config summary (path, registry)

## Phase 6: Update config templates to use core.#Config

- [ ] Update `cli/internal/config/templates.go`:
  - [ ] Add import for `opmodel.dev/core@v0` in `DefaultConfigTemplate`
  - [ ] Unify config with `core.#Config` for self-validating configs
  - [ ] Update `DefaultModuleTemplate` to add `opmodel.dev/core@v0` dependency
- [ ] Update template to produce:
  ```cue
  import (
      "opmodel.dev/core@v0"
      prov "opmodel.dev/providers@v0"
  )
  
  config: core.#Config & {
      // ... existing fields
  }
  ```
- [ ] Test that `opm config init` produces valid configs that pass `opm config vet`

## Phase 7: Wire up commands

- [ ] Update `cli/internal/cmd/mod/vet.go`:
  - [ ] Use `validator.ModuleValidator` instead of `cue.Vet()`
  - [ ] Add `--debug` flag
  - [ ] Add `--values` flag (multiple)
  - [ ] Add `--package` flag
  - [ ] Keep `--concrete` flag
- [ ] Update `cli/internal/cmd/config/vet.go`:
  - [ ] Use `validator.ConfigValidator`
  - [ ] Add `--registry` flag override
- [ ] Remove `Vet()` function from `cli/internal/cue/binary.go`
- [ ] Update command help text

## Phase 8: Testing

- [ ] Unit tests for `ModuleValidator`
  - [ ] Valid module with all required files
  - [ ] CUE syntax errors
  - [ ] Schema violations
  - [ ] Non-concrete values with `--concrete`
- [ ] Unit tests for `ConfigValidator`
  - [ ] Valid config
  - [ ] Missing config directory
  - [ ] Invalid field values
- [ ] Integration tests
  - [ ] End-to-end `opm mod vet` command
  - [ ] End-to-end `opm config vet` command
  - [ ] `opm config init` followed by `opm config vet` succeeds
- [ ] Error formatting tests
  - [ ] File location extraction
  - [ ] Multiple error aggregation

## Definition of Done

- [ ] All unit tests pass
- [ ] All integration tests pass
- [ ] `opm mod vet` no longer depends on external `cue` binary
- [ ] `opm config vet` no longer depends on external `cue` binary
- [ ] Error messages include file:line:col where available
- [ ] Success output shows entity summary
- [ ] `--debug`, `--values`, `--package`, `--concrete` flags work as specified
- [ ] Documentation updated in CLI help text
- [ ] `core.#Config` schema exists in catalog
- [ ] `opm config init` produces configs that unify with `core.#Config`
