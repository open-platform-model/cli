## 1. Verify Two-Phase Loading Implementation

- [x] 1.1 Review `internal/config/loader.go` for registry regex extraction
- [x] 1.2 Verify bootstrap handles missing config file gracefully (returns empty string)
- [x] 1.3 Verify bootstrap handles config without registry field (returns empty string)
- [x] 1.4 Add table-driven tests for bootstrap extraction edge cases
- [x] 1.5 Verify CUE_REGISTRY is set before full CUE evaluation in loader

## 2. Verify Precedence Chain Implementation

- [x] 2.1 Review precedence resolution in `internal/config/resolver.go`
- [x] 2.2 Verify flag > env > config > default ordering for registry
- [x] 2.3 Verify precedence ordering for kubeconfig, context, namespace
- [x] 2.4 Add table-driven tests for precedence resolution scenarios
- [x] 2.5 Verify resolution tracking captures source and shadowed values

## 3. Verify Config Init Command

- [x] 3.1 Review `internal/cmd/config_init.go` implementation
- [x] 3.2 Verify directory creation with 0700 permissions for `~/.opm/` and `~/.opm/cue.mod/`
- [x] 3.3 Verify file creation with 0600 permissions for config.cue and module.cue
- [x] 3.4 Verify --force flag overwrites existing configuration
- [x] 3.5 Verify error message and hint when config already exists
- [x] 3.6 Verify generated config.cue includes provider import template
- [x] 3.7 Verify generated module.cue enables import resolution
- [x] 3.8 Add table-driven tests for init command scenarios

## 4. Implement Config Vet Command

- [x] 4.1 Create `internal/cmd/config_vet.go` implementation
- [x] 4.2 Verify validation checks config file existence
- [x] 4.3 Verify validation checks cue.mod/module.cue existence
- [x] 4.4 Verify CUE syntax validation with actionable error messages
- [x] 4.5 Verify --config flag and OPM_CONFIG env override paths
- [x] 4.6 Verify registry passed to CUE evaluation for import resolution
- [x] 4.7 Add table-driven tests for vet command scenarios

## 5. Verify Error Handling Patterns

- [x] 5.1 Verify fail-fast when providers configured without registry
- [x] 5.2 Verify error includes actionable hint for registry resolution
- [x] 5.3 Verify invalid CUE config includes hint to run `opm config vet`
- [x] 5.4 Add table-driven tests for error scenarios

## 6. Verify Filesystem Paths

- [x] 6.1 Verify tilde expansion via `os.UserHomeDir()` in `internal/config/paths.go`
- [x] 6.2 Verify no hardcoded paths (use filepath.Join)
- [x] 6.3 Verify OPM_CONFIG environment variable overrides default path
- [x] 6.4 Add tests for path resolution on different platforms

## 7. Validation Gates

- [x] 7.1 Run `task fmt` - verify all Go files formatted
- [x] 7.2 Run `task lint` - verify golangci-lint passes
- [x] 7.3 Run `task test` - verify all tests pass
