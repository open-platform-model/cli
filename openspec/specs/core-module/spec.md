# Core Module

## Purpose

Defines the `Module` and `ModuleMetadata` types in `pkg/module/`. The dead `pkgName` field and stale comments have been removed.

---

## Requirements

### Requirement: Module type location
The `Module` and `ModuleMetadata` types SHALL be defined in `pkg/module/` (moved from `internal/core/module/`). The dead `pkgName` field and its stale comment SHALL be removed.

#### Scenario: Module is importable from pkg/module
- **WHEN** code imports `github.com/opmodel/cli/pkg/module`
- **THEN** `module.Module` and `module.ModuleMetadata` are accessible

#### Scenario: Dead pkgName field removed
- **WHEN** code accesses `module.Module.pkgName`
- **THEN** compilation fails — the field no longer exists
