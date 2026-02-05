# Proposal: CLI Validation Commands

## Intent

Replace the stub `opm mod vet` and `opm config vet` implementations (which shell out to `cue vet`) with native Go CUE SDK implementations. This enables custom error formatting, programmatic control, and integration with the CLI's config system.

## Context

The current implementation in `cli/internal/cue/binary.go` simply wraps the external `cue vet` command:

```go
func Vet(dir string, concrete bool, registry string) error {
    args := []string{"vet", "./..."}
    if concrete {
        args = append(args, "--concrete")
    }
    return RunCUECommand(dir, args, registry)
}
```

This approach has limitations:
- No custom error formatting (Charm ecosystem integration)
- Limited programmatic control over validation phases
- No entity summary output
- Missing flags: `--debug`, `--values`, `--package`
- No fail-on-end error aggregation

## Scope

**In scope:**
- Replace `cue.Vet()` stub with Go CUE SDK implementation
- 4-phase validation pipeline for `mod vet` (structure → load → schema → concrete)
- 4-phase validation pipeline for `config vet` (structure → load → schema → providers)
- Custom error formatting with file:line:col via Charm ecosystem
- Entity summary output on success
- Flags: `--debug`, `--values`, `--package`, `--concrete`
- Fail-on-end error aggregation

**Out of scope:**
- Bundle validation (future)
- Auto-fix capabilities
- Watch mode
- Custom validation rules beyond CUE schema

## Approach

1. Create `cli/internal/validator/` package with `ModuleValidator` and `ConfigValidator` types
2. Implement 4-phase pipeline using `cuelang.org/go/cue/load` for instance loading
3. Use `cue.Value.Validate()` with options like `cue.Concrete(true)` for concrete validation
4. Format errors with Charm lipgloss styling, extracting file:line:col from CUE errors
5. Aggregate all errors before exit (fail-on-end pattern)
6. Remove dependency on external `cue` binary for validation
