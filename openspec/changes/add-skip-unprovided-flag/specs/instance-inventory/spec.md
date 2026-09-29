## ADDED Requirements

### Requirement: Skipped contracts annotation

When an apply rendered with `--skip-unprovided` and the kernel skipped at least one demand, the CLI SHALL include the annotation `module-instance.opmodel.dev/skipped-contracts` in its spec apply. Its value SHALL be the skipped demands as `<component>=<contract fqn>` pairs, sorted, deduplicated and joined with commas. When a later apply skips nothing, with or without the flag, the CLI SHALL omit the annotation so server-side apply removes it. The annotation is information for whoever inspects the instance: no CLI gate SHALL read it as an authority.

#### Scenario: A skipping apply records its skips

- **WHEN** `opm instance apply --skip-unprovided` renders an instance whose component `db` skips `opmodel.dev/catalogs/opm/traits/backup@v1alpha1`
- **THEN** the ModuleInstance SHALL carry `module-instance.opmodel.dev/skipped-contracts: db=opmodel.dev/catalogs/opm/traits/backup@v1alpha1`

#### Scenario: A complete apply clears the record

- **WHEN** the same instance is applied again against a platform that provides the backup contract
- **THEN** the annotation SHALL no longer be present on the ModuleInstance

#### Scenario: A flag that skipped nothing writes nothing

- **WHEN** `opm instance apply --skip-unprovided` renders an instance with no unprovided demand
- **THEN** the ModuleInstance SHALL NOT carry the annotation
