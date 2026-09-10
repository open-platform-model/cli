## MODIFIED Requirements

### Requirement: Local-replacement renders warn (0010 D19)

Every render SHALL enable the kernel's local replacements, so a `cue.mod/local-module.cue` in the module context (the main module of an instance-file render, or the module directory of a module render) or in the platform module directory reaches the render build under the kernel's precedence: the platform's replacements whole, the module's only for paths the platform's dependency list does not name. The CLI SHALL word the kernel's replacement rows as warnings, one per honoured replacement, naming the replaced path, the directory or module it was served from and whether the module or the platform supplied it, and stating that the rendered bytes may not correspond to any published build. For each `replaceWith` in the module context's own `cue.mod/local-module.cue` that no row honours, the CLI SHALL warn that the platform names that path and that the redirect belongs in the platform module's `cue.mod/local-module.cue`. The warnings SHALL NOT block the render or alter its output. A context without the file SHALL produce no such warning. Both render entry points SHALL perform this. The provenance signal (`SourceLocal`, the `module-instance.opmodel.dev/source: local` annotation) keeps its file-presence rule.

#### Scenario: Replaced dependency warns

- **WHEN** an instance render's main module carries `cue.mod/local-module.cue` replacing a dependency the platform does not name with a local checkout
- **THEN** the rendered objects reflect the checkout's bytes
- **AND** a warning names the path, the checkout directory and the module as its source, with the published-bytes caveat
- **AND** the render proceeds

#### Scenario: Platform replacement of its catalog is honoured and warns

- **WHEN** a render's platform module directory (`--platform <dir>` or the local default platform module) carries `cue.mod/local-module.cue` replacing its catalog path with a checkout
- **THEN** the rendered objects reflect the checkout's transformer bytes
- **AND** a warning names the catalog path, the checkout directory and the platform as its source

#### Scenario: Module-side replacement of a platform path is reported inert

- **WHEN** a module render's module directory carries `cue.mod/local-module.cue` replacing a path the platform module's dependency list names
- **THEN** the rendered objects reflect the platform's pinned bytes
- **AND** a warning names the path and says the redirect belongs in the platform module's `cue.mod/local-module.cue`

#### Scenario: Clean context stays silent

- **WHEN** neither the module context nor the platform module carries a `local-module.cue` with a `replaceWith`
- **THEN** no such warning is emitted and the output is identical to a render without the opt-in

#### Scenario: A malformed local file is a validation failure

- **WHEN** the module context's `cue.mod/local-module.cue` does not parse, or a replacement directory's `cue.mod/module.cue` declares a different module path than the replaced one
- **THEN** the command fails as a validation error with the kernel's message naming the file or the two paths, and no objects are rendered
