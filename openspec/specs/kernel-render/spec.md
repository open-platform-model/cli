# Capability: kernel-render

## Purpose

The CLI renders every instance through the `library` kernel — the same kernel the operator runs — replacing the CLI's own render/match pipeline (enhancement 0006 D9).

## Requirements

### Requirement: All renders go through the library kernel

The CLI SHALL render instances exclusively through `github.com/open-platform-model/library`'s kernel: one `Kernel.Render` call per render, taking a source-carrying instance and a source-carrying platform module, the CLI's runtime identity and the resolved skew policy; matching, execution and diagnostics happen inside that single build. The CLI MUST NOT carry its own component-matching, transformer-execution, or render-finalization implementation, MUST NOT hold any built platform value between renders, and MUST NOT import `opm-operator` Go packages (0006 D13). The rendered resource set is the render result's compiled objects.

The kernel reports advisory facts as structured rows and attaches no message strings to a render result. The CLI SHALL word them itself and surface them to the user as warnings, never dropping one: an unhandled optional trait is read from the render diagnostics' unhandled-trait table, and a module requiring a newer OPM-namespace build than the platform carries is read from the resolved-versions row marked newer. Both warnings SHALL name the same facts the kernel previously named — for skew, the path and both versions.

#### Scenario: Instance apply renders via the kernel

- **WHEN** `opm instance apply <file.cue>` runs
- **THEN** the rendered resource set SHALL be produced by `Kernel.Render`'s output, not by any `pkg/render` code

#### Scenario: No CLI-side match implementation

- **WHEN** the project is compiled
- **THEN** the packages `pkg/render` and `pkg/provider` SHALL NOT exist
- **AND** `pkg/loader` SHALL NOT contain component-to-transformer matching code

#### Scenario: Render warnings reach the user

- **WHEN** a render's module requires a newer catalog build than the platform pins and the skew policy is warn
- **THEN** the command prints a CLI-worded skew warning naming the path and both versions, derived from the resolved-versions row marked newer, and the render proceeds

#### Scenario: Unhandled optional trait warns

- **WHEN** a render succeeds with a trait no matched transformer handles whose effective posture is optional
- **THEN** the command prints a CLI-worded warning naming the component and the trait, derived from the diagnostics' unhandled-trait table

#### Scenario: Skew refusal is a validation failure

- **WHEN** the skew policy is refuse and a module requires a newer OPM-namespace build than the platform pins
- **THEN** the command fails before evaluation as a validation error naming the path, the module's required version and the platform's version

#### Scenario: Typed render causes keep their exit codes

- **WHEN** a render fails with unresolved demands or unmatched components
- **THEN** the command exits as a validation failure with the kernel's message; a transform error or an over-subscribed provider contract exits the same way, with the diagnostics printed beside the refusal

### Requirement: CLI entry points map onto kernel entry points

Each CLI loading path SHALL use the corresponding kernel acquire verb: instance files via `AcquireInstanceFromDir` with any `-f` values files passed as its trailing values sources, so the instance the render imports already carries them; local module packages via `AcquireModuleFromDir` + `SynthesizeInstance`; registry references via `AcquireModuleFromRegistry` + `SynthesizeInstance`; every platform source via `AcquirePlatformFromDir`. No acquire call SHALL pass per-call load options: the registry mapping is supplied once when the kernel is constructed, and it reaches module acquisition and the schema loader from there. The CLI MUST NOT reach for a raw-value loading tier or construct an artifact from a value it built itself; a caller needing the underlying CUE value SHALL read the acquired artifact's own package field. One `kernel.Kernel` SHALL be constructed per command invocation and threaded through the workflow; packages needing a `*cue.Context` SHALL receive the one the kernel's schema value was built in, never a second one of their own. The CLI MUST NOT fill values into an evaluated instance from Go.

#### Scenario: Module build synthesizes through the kernel

- **WHEN** `opm module build <dir>` runs against a module package directory
- **THEN** the module SHALL be acquired by `AcquireModuleFromDir` and the instance produced by kernel `SynthesizeInstance` from it and the resolved values

#### Scenario: Registry mapping is supplied once

- **WHEN** any render-bearing command acquires a module, an instance or a platform
- **THEN** the acquire call SHALL take no registry or load-options argument, and the mapping the build resolves against SHALL be the one the kernel was constructed with

#### Scenario: Single kernel per invocation

- **WHEN** any render-bearing command executes
- **THEN** exactly one `kernel.Kernel` SHALL be constructed for that invocation
- **AND** no code path SHALL construct a second `cuecontext.New()` for render work

#### Scenario: Instance file with extra values

- **WHEN** `opm instance build ./instance.cue -f overrides.cue` runs
- **THEN** the instance is acquired from its directory with `overrides.cue` passed as a trailing values source, the render imports that layered package, and the rendered objects reflect the override

#### Scenario: Publish gate acquires through the kernel

- **WHEN** the publish gate checks that a directory holds a valid module package
- **THEN** it SHALL acquire it through a kernel acquire verb, and SHALL classify a refusal by the shape-gate sentinels the library's errors package declares

### Requirement: Runtime identity in rendered output

The CLI SHALL inject its runtime identity (`opm-cli`) into the kernel render context, so rendered resources carry the CLI's runtime provenance the same way operator-rendered resources carry `opm-controller`.

#### Scenario: Runtime label distinguishes actor

- **WHEN** the CLI renders an instance
- **THEN** the render context's runtime identity SHALL be `opm-cli`

### Requirement: Render digests are kernel-derived and operator-parity

`status.lastAppliedRenderDigest` SHALL be computed over the kernel's rendered objects using the same canonical serialization the operator uses. A registry-gated integration check SHALL verify that the CLI's local-dir staging path and the operator's registry-acquisition call sequence, both ending in `Kernel.Render` against the same platform module, produce identical render digests for the same instance, with the runtime name held constant (0006 D30 gate; the runtime identity is stamped into rendered labels, so cross-actor digests differ by that label by construction). Evaluator-version skew reporting applies to the future cross-binary comparison (slice C3, where CLI and operator binaries embed separate CUE evaluators); the in-binary check renders both paths with one evaluator and cannot exhibit skew.

#### Scenario: Parity for the same inputs

- **WHEN** the parity check renders a fixture instance via the CLI kernel path and via the operator's call sequence against the same platform module
- **THEN** the two render digests SHALL be identical

#### Scenario: Skew reported explicitly (cross-binary check, slice C3)

- **WHEN** the future cross-binary parity comparison runs while the `cli` and `opm-operator` binaries embed different `cuelang.org/go` minor versions
- **THEN** the check SHALL fail with a message naming the evaluator-version skew as the suspected cause

### Requirement: Local-replacement renders warn (0010 D19)

When the effective module context of a render carries a local-path replacement — the main module (instance-file path) or the module directory (module path) has a `cue.mod/local-module.cue` with at least one local `replaceWith` — the render SHALL emit a warning that demanded keys may not correspond to published bytes. The check SHALL be a deterministic, offline file test (no registry query, no new fields); it SHALL NOT block the render or alter its output. Both render entry points SHALL perform it.

#### Scenario: Replaced dependency warns

- **WHEN** an instance render's main module carries `cue.mod/local-module.cue` replacing a dependency with a local checkout
- **THEN** a warning is emitted naming the local-replacement condition
- **AND** the render proceeds unchanged

#### Scenario: Clean context stays silent

- **WHEN** no `local-module.cue` with a local `replaceWith` is present
- **THEN** no such warning is emitted
