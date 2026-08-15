# kernel-render — Delta

## ADDED Requirements

### Requirement: Local-replacement renders warn (0010 D19)

When the effective module context of a render carries a local-path replacement — the main module (instance-file path) or the module directory (module path) has a `cue.mod/local-module.cue` with at least one local `replaceWith` — the render SHALL emit a warning that demanded keys may not correspond to published bytes. The check SHALL be a deterministic, offline file test (no registry query, no new fields); it SHALL NOT block the render or alter its output. Both render entry points SHALL perform it.

#### Scenario: Replaced dependency warns

- **WHEN** an instance render's main module carries `cue.mod/local-module.cue` replacing a dependency with a local checkout
- **THEN** a warning is emitted naming the local-replacement condition
- **AND** the render proceeds unchanged

#### Scenario: Clean context stays silent

- **WHEN** no `local-module.cue` with a local `replaceWith` is present
- **THEN** no such warning is emitted
