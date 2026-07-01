## MODIFIED Requirements

### Requirement: Legacy pipeline package

The `internal/legacy/` package SHALL NOT exist; all render-pipeline phase logic SHALL live in `internal/loader/`, `internal/builder/`, `internal/provider/`, and `internal/transformer/`.

**Reason**: Replaced by `internal/pipeline/`. All phase logic now lives in
`internal/loader/`, `internal/builder/`, `internal/provider/`, and
`internal/transformer/`. The monolithic `internal/legacy/` package served as a
transitional holder while those packages were built out; it is no longer needed.

**Migration**: Replace any import of `github.com/open-platform-model/cli/internal/legacy` with
`github.com/open-platform-model/cli/internal/pipeline`. Replace `legacy.NewPipeline(config)`
with `pipeline.NewPipeline(config)`. The `Pipeline` interface, `RenderOptions`,
`RenderResult`, and helper methods (`HasErrors`, `HasWarnings`, `ResourceCount`)
are available at the new import path with identical signatures.

#### Scenario: No files import internal/legacy after this change

- **WHEN** the pipeline orchestration change is fully applied
- **THEN** no Go source file in the repository SHALL import `github.com/open-platform-model/cli/internal/legacy`
- **AND** the `internal/legacy/` directory SHALL NOT exist in the repository
