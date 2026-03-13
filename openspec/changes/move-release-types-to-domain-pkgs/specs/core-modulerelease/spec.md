## MODIFIED Requirements

### Requirement: ModuleRelease type location and structure
The `Release` and `ReleaseMetadata` types SHALL be defined in `pkg/module/` (in `release.go`). `Release` SHALL expose components via typed accessor methods instead of raw public fields.

#### Scenario: Release is importable from pkg/module
- **WHEN** code imports `github.com/opmodel/cli/pkg/module`
- **THEN** `module.Release` and `module.ReleaseMetadata` are accessible

#### Scenario: Schema and DataComponents are not public fields
- **WHEN** code accesses `release.Schema` or `release.DataComponents` directly
- **THEN** compilation fails — use `release.MatchComponents()` or `release.ExecuteComponents()` instead

#### Scenario: Consistent value/pointer embedding
- **WHEN** `Release` is constructed
- **THEN** metadata fields use pointer convention (`*ReleaseMetadata`) and embedded types use value convention consistently
