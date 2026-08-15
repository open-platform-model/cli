# instance-inventory — Delta

## MODIFIED Requirements

### Requirement: Spec Writing (CLI-executor mode)

On apply in CLI-executor mode, the CLI SHALL server-side-apply the CR spec with field manager `opm-cli`, containing: `spec.owner: cli`, `spec.module.path` and `spec.module.version` set to the module's declared identity **read verbatim from core-v2 metadata** — `spec.module.path` is `metadata.modulePath` as-is (the complete major-suffixed registry address; no composition from a parent prefix and a name), and `spec.module.version` is `metadata.version` normalized to the `v`-prefixed registry-tag form — and `spec.values` set to the single unified values blob that the render consumed. The pair applies for local-directory and locally-replaced module resolution as well; the CR MUST NOT contain a filesystem path. The declared pair is verified against fetched coordinates wherever it later meets a registry (handoff verification, operator acquire), not at write time.

#### Scenario: Local-path apply writes the declared reference

- **WHEN** applying from a local module directory whose `module.cue` declares `modulePath: "opmodel.dev/modules/podinfo@v0"` and `version: "0.1.4"`
- **THEN** `spec.module.path` SHALL be `opmodel.dev/modules/podinfo@v0` and `spec.module.version` SHALL be `v0.1.4`

#### Scenario: No address arithmetic

- **WHEN** the written path is compared to the module's declared `metadata.modulePath`
- **THEN** they SHALL be byte-identical — no major tag, leaf, or prefix is computed by the CLI
