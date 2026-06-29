## MODIFIED Requirements

<!-- enhancement 0002 D6/D8/D9 — X3-deferred to X4 per D-X3.6 (single capability owner). Restates only the requirements whose normative text changes under the rename: the synthetic-instance flow, the `#ModuleInstance` kind, the `opm instance apply` cross-reference, and the instance-identity wording. Unchanged flag-surface/prune/dry-run behavior rides the archive spec-sync. -->

### Requirement: `opm module apply` deploys a module package via the synthetic instance flow

The CLI SHALL provide a `module apply` subcommand (with alias `mod apply`) under the `module` command group that accepts an optional positional argument (the module-package directory, defaulting to `"."`) and deploys the synthesized `#ModuleInstance` to a Kubernetes cluster. The subcommand SHALL accept only directory inputs. <!-- Was: synthetic release flow / #ModuleRelease (0002 D8) -->

The subcommand SHALL produce the same cluster-side effects as `opm instance apply` after the render stage: server-side apply of all rendered resources, inventory-secret read/write keyed on the instance UUID, stale-resource pruning, ownership checks, and dry-run support. <!-- Was: opm release apply, release UUID -->

#### Scenario: Default to current directory

- **WHEN** the user runs `opm module apply` with no positional argument from inside a module package directory
- **THEN** the subcommand SHALL synthesize a `#ModuleInstance` from that directory and apply the result to the cluster

#### Scenario: File argument rejected

- **WHEN** the user runs `opm module apply ./my-module/module.cue`
- **THEN** the subcommand SHALL return an error stating that `module apply` expects a directory
- **AND** SHALL point the user to `opm instance apply <file>` for instance files

### Requirement: Instance identity is derived in CUE, not the CLI

The synthetic instance's `metadata.uuid` SHALL be computed by the catalog's CUE schema (`SHA1(OPMNamespace, "<moduleUUID>:<name>:<namespace>")`) and SHALL NOT be generated, randomized, or persisted by the CLI itself. The CLI SHALL read the computed UUID from the rendered `ModuleInstance` and use it as the `instanceID` passed to the apply workflow. <!-- Was: synthetic release, ModuleRelease, releaseID (0002 D8/D9) -->

#### Scenario: Stable UUID across runs

- **WHEN** the user runs `opm module apply ./foo` twice with identical inputs
- **THEN** both invocations SHALL produce the same instance UUID
- **AND** SHALL access the same inventory record

### Requirement: Apply log surfaces the synthetic instance name

The subcommand SHALL log the resolved synthetic instance name (including the default `<module>-debug` suffix when applicable) prominently in the apply summary so the operator can recognise a synthetic-instance deployment in command-line scrollback or CI logs. <!-- Was: synthetic release name (0002 D9) -->

#### Scenario: Synthetic instance name in apply log

- **WHEN** the user runs `opm module apply ./my-module` against a cluster
- **THEN** the apply log SHALL include a line identifying the instance as `my-module-debug` (or the user-provided `--name` value)

### Requirement: Failures surface validation errors before any cluster contact

The subcommand SHALL validate and synthesize the `#ModuleInstance` before opening any connection to the Kubernetes cluster. Render-time errors SHALL exit with the validation-error exit code and SHALL NOT contact the apiserver. <!-- Was: #ModuleRelease (0002 D8) -->

#### Scenario: Synthesis failure exits before cluster contact

- **WHEN** the module is missing a required `cue.mod/module.cue` catalog dependency
- **AND** the user runs `opm module apply ./broken-module`
- **THEN** the subcommand SHALL exit with the validation-error exit code
- **AND** SHALL NOT issue any apiserver requests
