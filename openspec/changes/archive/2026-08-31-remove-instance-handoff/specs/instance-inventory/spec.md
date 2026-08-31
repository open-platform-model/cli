## MODIFIED Requirements

### Requirement: Spec write contents

On apply in CLI-executor mode, the CLI SHALL server-side-apply the CR spec with field manager `opm-cli`, containing: `spec.owner: cli`, `spec.module.path` and `spec.module.version` set to the module's declared identity **read verbatim from core-v2 metadata** — `spec.module.path` is `metadata.modulePath` as-is (the complete major-suffixed registry address; no composition from a parent prefix and a name), and `spec.module.version` is `metadata.version` normalized to the `v`-prefixed registry-tag form — and `spec.values` set to the single unified values blob that the render consumed. The pair applies for local-directory and locally-replaced module resolution as well; the CR MUST NOT contain a filesystem path. The declared pair is verified against fetched coordinates wherever it later meets a registry (operator acquire; any future ownership transfer), not at write time.

#### Scenario: Local-path apply writes the declared reference

- **WHEN** applying from a local module directory whose `module.cue` declares `modulePath: "opmodel.dev/modules/podinfo@v0"` and `version: "0.1.4"`
- **THEN** `spec.module.path` SHALL be `opmodel.dev/modules/podinfo@v0` and `spec.module.version` SHALL be `v0.1.4`

#### Scenario: No address arithmetic

- **WHEN** the written path is compared to the module's declared `metadata.modulePath`
- **THEN** they SHALL be byte-identical — no major tag, leaf, or prefix is computed by the CLI

#### Scenario: Values are the unified blob

- **WHEN** applying with multiple `--values` files
- **THEN** `spec.values` SHALL contain the single unified result the render consumed, not the individual layers

### Requirement: Render provenance annotation

When the applied render's module bytes did not come from pure registry resolution — the main module is a local directory, or the main module's `cue.mod/local-module.cue` contains any local-path `replaceWith` — the CLI SHALL include the annotation `module-instance.opmodel.dev/source: local` in its spec apply. When a later apply resolves fully from registries, the CLI SHALL omit the annotation so server-side apply removes it. The annotation is a fail-closed provenance signal: the thin-editor apply path refuses a module that resolves from local bytes, and any future transfer of an instance to operator ownership reads it as a refusal; no CLI-executor-mode gate SHALL read it as an authority.

#### Scenario: Local render stamps the annotation

- **WHEN** an apply renders from a local module directory
- **THEN** the CR SHALL carry `module-instance.opmodel.dev/source: local`

#### Scenario: Replacement in effect stamps the annotation

- **WHEN** an apply's main module has a `cue.mod/local-module.cue` with a local-path `replaceWith`
- **THEN** the CR SHALL carry `module-instance.opmodel.dev/source: local`

#### Scenario: Registry apply clears the annotation

- **WHEN** an instance carrying the annotation is re-applied with fully registry-resolved modules
- **THEN** the annotation SHALL no longer be present on the CR
