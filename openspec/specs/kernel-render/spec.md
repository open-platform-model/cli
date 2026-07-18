# Capability: kernel-render

## Purpose

The CLI renders every instance through the `library` kernel — the same kernel the operator runs — replacing the CLI's own render/match pipeline (enhancement 0006 D9).

## Requirements

### Requirement: All renders go through the library kernel

The CLI SHALL render instances exclusively through `github.com/open-platform-model/library`'s kernel: `Validate` → `Match` → `Plan` → `Compile` → `Finalize` against a materialized platform. The CLI MUST NOT carry its own component-matching, transformer-execution, or render-finalization implementation. The CLI MUST NOT import `opm-operator` Go packages (0006 D13).

#### Scenario: Instance apply renders via the kernel

- **WHEN** `opm instance apply <file.cue>` runs
- **THEN** the rendered resource set SHALL be produced by kernel `Compile`/`Finalize` output, not by any `pkg/render` code

#### Scenario: No CLI-side match implementation

- **WHEN** the project is compiled
- **THEN** the packages `pkg/render` and `pkg/provider` SHALL NOT exist
- **AND** `pkg/loader` SHALL NOT contain component-to-transformer matching code

### Requirement: CLI entry points map onto kernel entry points

Each CLI loading path SHALL use the corresponding kernel entry point: instance files via `LoadInstancePackage`/`LoadSourceFromFile` + `ProcessModuleInstance`; local module packages via `LoadModulePackage` + `SynthesizeInstance`; registry references via `AcquireModuleFromRegistry` + `SynthesizeInstance`. One `kernel.Kernel` SHALL be constructed per command invocation and threaded through the workflow; packages needing a `*cue.Context` SHALL receive the kernel's.

#### Scenario: Module build synthesizes through the kernel

- **WHEN** `opm module build <dir>` runs against a module package directory
- **THEN** the instance SHALL be produced by kernel `SynthesizeInstance` from the loaded module and resolved values

#### Scenario: Single kernel per invocation

- **WHEN** any render-bearing command executes
- **THEN** exactly one `kernel.Kernel` SHALL be constructed for that invocation
- **AND** no code path SHALL construct a second `cuecontext.New()` for render work

### Requirement: Runtime identity in rendered output

The CLI SHALL inject its runtime identity (`opm-cli`) into the kernel render context, so rendered resources carry the CLI's runtime provenance the same way operator-rendered resources carry `opm-controller`.

#### Scenario: Runtime label distinguishes actor

- **WHEN** the CLI renders an instance
- **THEN** the render context's runtime identity SHALL be `opm-cli`

### Requirement: Render digests are kernel-derived and operator-parity

`status.lastAppliedRenderDigest` SHALL be computed over the kernel-finalized manifests using the same canonical serialization the operator uses. A registry-gated integration check SHALL verify that the CLI's local-dir staging path and the operator's registry-acquisition call sequence produce identical render digests for the same instance against the same Platform spec, with the runtime name held constant (0006 D30 gate; the runtime identity is stamped into rendered labels, so cross-actor digests differ by that label by construction). Evaluator-version skew reporting applies to the future cross-binary comparison (slice C3, where CLI and operator binaries embed separate CUE evaluators); the in-binary check compiles both paths with one evaluator and cannot exhibit skew.

#### Scenario: Parity for the same inputs

- **WHEN** the parity check renders a fixture instance via the CLI kernel path and via the operator's renderer against the same Platform spec
- **THEN** the two render digests SHALL be identical

#### Scenario: Skew reported explicitly (cross-binary check, slice C3)

- **WHEN** the future cross-binary parity comparison runs while the `cli` and `opm-operator` binaries embed different `cuelang.org/go` minor versions
- **THEN** the check SHALL fail with a message naming the evaluator-version skew as the suspected cause
