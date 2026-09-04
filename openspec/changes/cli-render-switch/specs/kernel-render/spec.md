## MODIFIED Requirements

### Requirement: All renders go through the library kernel

The CLI SHALL render instances exclusively through `github.com/open-platform-model/library`'s kernel: one `Kernel.Render` call per render, taking a source-carrying instance and a source-carrying platform module, the CLI's runtime identity and the resolved skew policy; matching, execution and diagnostics happen inside that single build. The CLI MUST NOT carry its own component-matching, transformer-execution, or render-finalization implementation, MUST NOT hold any built platform value between renders, and MUST NOT import `opm-operator` Go packages (0006 D13). The rendered resource set is the render result's compiled objects; the result's warnings (unhandled optional traits, version skew under the warn policy) SHALL be surfaced to the user as warnings, never dropped.

#### Scenario: Instance apply renders via the kernel

- **WHEN** `opm instance apply <file.cue>` runs
- **THEN** the rendered resource set SHALL be produced by `Kernel.Render`'s output, not by any `pkg/render` code

#### Scenario: No CLI-side match implementation

- **WHEN** the project is compiled
- **THEN** the packages `pkg/render` and `pkg/provider` SHALL NOT exist
- **AND** `pkg/loader` SHALL NOT contain component-to-transformer matching code

#### Scenario: Render warnings reach the user

- **WHEN** a render's module requires a newer catalog build than the platform pins and the skew policy is warn
- **THEN** the command prints the kernel's skew warning naming the path and both versions, and the render proceeds

#### Scenario: Skew refusal is a validation failure

- **WHEN** the skew policy is refuse and a module requires a newer OPM-namespace build than the platform pins
- **THEN** the command fails before evaluation as a validation error naming the path, the module's required version and the platform's version

#### Scenario: Typed render causes keep their exit codes

- **WHEN** a render fails with unresolved demands or unmatched components
- **THEN** the command exits as a validation failure with the kernel's message; a transform error or an over-subscribed provider contract exits the same way, with the diagnostics printed beside the refusal

### Requirement: CLI entry points map onto kernel entry points

Each CLI loading path SHALL use the corresponding kernel entry point: instance files via `AcquireInstanceFromDir`, with any `-f` values files layered through the kernel's values option so the instance the render imports already carries them; local module packages via `LoadModulePackage` + `SynthesizeInstance`; registry references via `AcquireModuleFromRegistry` + `SynthesizeInstance`; every platform source via `AcquirePlatformFromDir`. One `kernel.Kernel` SHALL be constructed per command invocation and threaded through the workflow; packages needing a `*cue.Context` SHALL receive the kernel's. The CLI MUST NOT fill values into an evaluated instance from Go.

#### Scenario: Module build synthesizes through the kernel

- **WHEN** `opm module build <dir>` runs against a module package directory
- **THEN** the instance SHALL be produced by kernel `SynthesizeInstance` from the loaded module and resolved values

#### Scenario: Single kernel per invocation

- **WHEN** any render-bearing command executes
- **THEN** exactly one `kernel.Kernel` SHALL be constructed for that invocation
- **AND** no code path SHALL construct a second `cuecontext.New()` for render work

#### Scenario: Instance file with extra values

- **WHEN** `opm instance build ./instance.cue -f overrides.cue` runs
- **THEN** the instance is acquired from its directory with `overrides.cue` layered as a values source, the render imports that layered package, and the rendered objects reflect the override

### Requirement: Render digests are kernel-derived and operator-parity

`status.lastAppliedRenderDigest` SHALL be computed over the kernel's rendered objects using the same canonical serialization the operator uses. A registry-gated integration check SHALL verify that the CLI's local-dir staging path and the operator's registry-acquisition call sequence, both ending in `Kernel.Render` against the same platform module, produce identical render digests for the same instance, with the runtime name held constant (0006 D30 gate; the runtime identity is stamped into rendered labels, so cross-actor digests differ by that label by construction). Evaluator-version skew reporting applies to the future cross-binary comparison (slice C3, where CLI and operator binaries embed separate CUE evaluators); the in-binary check renders both paths with one evaluator and cannot exhibit skew.

#### Scenario: Parity for the same inputs

- **WHEN** the parity check renders a fixture instance via the CLI kernel path and via the operator's call sequence against the same platform module
- **THEN** the two render digests SHALL be identical

#### Scenario: Skew reported explicitly (cross-binary check, slice C3)

- **WHEN** the future cross-binary parity comparison runs while the `cli` and `opm-operator` binaries embed different `cuelang.org/go` minor versions
- **THEN** the check SHALL fail with a message naming the evaluator-version skew as the suspected cause
