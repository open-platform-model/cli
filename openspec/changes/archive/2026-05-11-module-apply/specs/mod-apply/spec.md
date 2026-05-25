## ADDED Requirements

### Requirement: `opm module apply` deploys a module package via the synthetic release flow

The CLI SHALL provide a `module apply` subcommand (with alias `mod apply`) under the `module` command group that accepts an optional positional argument (the module-package directory, defaulting to `"."`) and deploys the synthesized `#ModuleRelease` to a Kubernetes cluster. The subcommand SHALL accept only directory inputs.

The subcommand SHALL produce the same cluster-side effects as `opm release apply` after the render stage: server-side apply of all rendered resources, inventory-secret read/write keyed on the release UUID, stale-resource pruning, ownership checks, and dry-run support.

#### Scenario: Default to current directory

- **WHEN** the user runs `opm module apply` with no positional argument from inside a module package directory
- **THEN** the subcommand SHALL synthesize a `#ModuleRelease` from that directory and apply the result to the cluster

#### Scenario: Explicit module directory

- **WHEN** the user runs `opm module apply ./my-module`
- **THEN** the subcommand SHALL synthesize a `#ModuleRelease` from `./my-module` and apply the result to the cluster

#### Scenario: File argument rejected

- **WHEN** the user runs `opm module apply ./my-module/module.cue`
- **THEN** the subcommand SHALL return an error stating that `module apply` expects a directory
- **AND** SHALL point the user to `opm release apply <file>` for release files

#### Scenario: Successful apply writes inventory

- **WHEN** the user runs `opm module apply ./my-module` against a cluster
- **AND** the apply succeeds
- **THEN** an inventory Secret SHALL be written keyed on the synthesized release UUID
- **AND** the inventory Secret SHALL record the module-directory absolute path, the resolved synthetic release name and namespace, and the module's metadata.version (when present)

#### Scenario: Re-apply with same inputs reuses inventory

- **WHEN** the user runs `opm module apply ./my-module` twice in succession with no source or flag changes
- **THEN** the second invocation SHALL read the inventory written by the first invocation
- **AND** SHALL skip any resources that are already up-to-date
- **AND** SHALL increment the inventory revision

#### Scenario: Re-apply after resource removal prunes

- **WHEN** a previous `opm module apply ./my-module` recorded an inventory with N entries
- **AND** the user modifies the module so that fewer resources render and runs `opm module apply ./my-module` again
- **THEN** the subcommand SHALL prune the stale resources by default
- **AND** SHALL update the inventory to reflect the new resource set

### Requirement: Flag surface matches `opm release apply` plus `--name`

The `module apply` subcommand SHALL accept the following flags with the listed behavior:

| Flag | Type | Default | Behavior |
| --- | --- | --- | --- |
| `-f`, `--values` | repeatable string | empty | Values files overriding the module's `debugValues` |
| `--provider` | string | from config | Provider override |
| `--name` | string | `<module>-debug` | Synthetic `metadata.name` override |
| `-n`, `--namespace` | string | from config | Target namespace (also propagates to synthetic `metadata.namespace`) |
| `--kubeconfig` | string | from env/config | Path to kubeconfig file |
| `--context` | string | current-context | Kubernetes context to use |
| `--dry-run` | bool | false | Server-side dry-run; no cluster changes |
| `--create-namespace` | bool | false | Create the target namespace if it does not exist |
| `--no-prune` | bool | false | Skip pruning of stale resources |
| `--force` | bool | false | Allow a 0-resource render to prune previously tracked resources |

#### Scenario: Values files override debugValues

- **WHEN** the user runs `opm module apply ./my-module -f overrides.cue`
- **THEN** the subcommand SHALL use `overrides.cue` as the source of values
- **AND** SHALL NOT fall back to the module's `debugValues`

#### Scenario: Namespace flag participates in release identity

- **WHEN** the user runs `opm module apply ./foo -n staging`
- **AND** the user later runs `opm module apply ./foo -n production`
- **THEN** the two invocations SHALL produce two distinct release UUIDs
- **AND** SHALL write two independent inventory Secrets in their respective namespaces

#### Scenario: Name flag participates in release identity

- **WHEN** the user runs `opm module apply ./foo --name myapp`
- **AND** the user later runs `opm module apply ./foo` (no `--name`)
- **THEN** the two invocations SHALL produce two distinct release UUIDs
- **AND** SHALL not interfere with each other's inventory

#### Scenario: Dry-run makes no cluster changes

- **WHEN** the user runs `opm module apply ./my-module --dry-run`
- **THEN** the subcommand SHALL perform a server-side dry-run apply
- **AND** SHALL NOT write or modify any inventory Secret
- **AND** SHALL NOT prune any resources
- **AND** SHALL log a summary of resources that would be applied

#### Scenario: Create-namespace auto-creates the target namespace

- **WHEN** the user runs `opm module apply ./my-module -n new-ns --create-namespace`
- **AND** namespace `new-ns` does not exist
- **THEN** the subcommand SHALL create the `new-ns` namespace before applying resources

#### Scenario: No-prune preserves stale resources

- **WHEN** a previous inventory recorded resources that are no longer rendered
- **AND** the user runs `opm module apply ./my-module --no-prune`
- **THEN** the stale resources SHALL be left in the cluster
- **AND** the inventory SHALL still be updated to reflect the new resource set

#### Scenario: Force allows 0-resource render to prune all

- **WHEN** a previous inventory has N>0 entries
- **AND** the user runs `opm module apply ./my-module` with a module that now renders 0 resources
- **AND** `--force` is NOT provided
- **THEN** the subcommand SHALL refuse to proceed and SHALL return an error explaining the situation
- **WHEN** the same conditions hold and `--force` IS provided
- **THEN** the subcommand SHALL prune all previously tracked resources

### Requirement: Release identity is derived in CUE, not the CLI

The synthetic release's `metadata.uuid` SHALL be computed by the catalog's CUE schema (`SHA1(OPMNamespace, "<moduleUUID>:<name>:<namespace>")`) and SHALL NOT be generated, randomized, or persisted by the CLI itself. The CLI SHALL read the computed UUID from the rendered `ModuleRelease` and use it as the `releaseID` passed to the apply workflow.

#### Scenario: Stable UUID across runs

- **WHEN** the user runs `opm module apply ./foo` twice with identical inputs
- **THEN** both invocations SHALL produce the same release UUID
- **AND** SHALL access the same inventory record

#### Scenario: UUID differs across `--name` values

- **WHEN** the user runs `opm module apply ./foo --name a`
- **AND** the user runs `opm module apply ./foo --name b`
- **THEN** the two invocations SHALL produce different release UUIDs

### Requirement: Inventory ChangeDescriptor reflects module-directory provenance

When the subcommand writes an inventory record, the embedded `ChangeDescriptor` SHALL record:
- `Path`: the absolute path of the module directory.
- `Local`: `true` (the module was loaded from disk, not from a registry).
- `Version`: the module's `metadata.version` value as decoded, which MAY be empty during local development.

#### Scenario: Inventory records module-directory path

- **WHEN** the user runs `opm module apply /workspace/my-module`
- **AND** the apply succeeds
- **THEN** the inventory Secret SHALL record `Path = "/workspace/my-module"`
- **AND** SHALL record `Local = true`

### Requirement: Apply log surfaces the synthetic release name

The subcommand SHALL log the resolved synthetic release name (including the default `<module>-debug` suffix when applicable) prominently in the apply summary so the operator can recognise a synthetic-release deployment in command-line scrollback or CI logs.

#### Scenario: Synthetic release name in apply log

- **WHEN** the user runs `opm module apply ./my-module` against a cluster
- **THEN** the apply log SHALL include a line identifying the release as `my-module-debug` (or the user-provided `--name` value)

### Requirement: Failures surface validation errors before any cluster contact

The subcommand SHALL validate and synthesize the `#ModuleRelease` before opening any connection to the Kubernetes cluster. Render-time errors SHALL exit with the validation-error exit code and SHALL NOT contact the apiserver.

#### Scenario: Synthesis failure exits before cluster contact

- **WHEN** the module is missing a required `cue.mod/module.cue` catalog dependency
- **AND** the user runs `opm module apply ./broken-module`
- **THEN** the subcommand SHALL exit with the validation-error exit code
- **AND** SHALL NOT issue any apiserver requests

#### Scenario: Module without debugValues and no `-f` flag

- **WHEN** the user runs `opm module apply ./no-debug-module` and the module defines no `debugValues`
- **AND** no `-f` flag is provided
- **THEN** the subcommand SHALL exit with an actionable error telling the user to define `debugValues` or supply values via `-f`
- **AND** SHALL NOT contact the cluster
