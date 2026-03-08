# Core ModuleRelease

## Purpose

Defines the `ModuleRelease` and `ReleaseMetadata` types in `pkg/modulerelease/`. `ModuleRelease` exposes components via typed accessor methods rather than raw public fields.

---

## Requirements

### Requirement: ModuleRelease type location and structure
The `ModuleRelease` and `ReleaseMetadata` types SHALL be defined in `pkg/modulerelease/` (moved from `internal/core/modulerelease/`). `ModuleRelease` SHALL expose components via typed accessor methods instead of raw public fields.

#### Scenario: ModuleRelease is importable from pkg/modulerelease
- **WHEN** code imports `github.com/opmodel/cli/pkg/modulerelease`
- **THEN** `modulerelease.ModuleRelease` and `modulerelease.ReleaseMetadata` are accessible

#### Scenario: Schema and DataComponents are not public fields
- **WHEN** code accesses `release.Schema` or `release.DataComponents` directly
- **THEN** compilation fails — use `release.MatchComponents()` or `release.ExecuteComponents()` instead

#### Scenario: Consistent value/pointer embedding
- **WHEN** `ModuleRelease` is constructed
- **THEN** metadata fields use pointer convention (`*ReleaseMetadata`) and embedded types use value convention consistently
