## RENAMED Requirements

- FROM: `### Requirement: Render digests are kernel-derived and operator-parity`
- TO: `### Requirement: Render digests are kernel-derived and controller-parity`

## REMOVED Requirements

### Requirement: Unprovided provider-fulfilled demands are skipped on request
**Reason**: Renamed with the product: the operator is now the controller (`opm controller`, `opm-system`, `spec.owner: controller`). Restated under the new title.
**Migration**: None.

## MODIFIED Requirements

### Requirement: All renders go through the library kernel

The CLI SHALL render instances exclusively through `github.com/open-platform-model/library`'s kernel: one `Kernel.Render` call per render, taking a source-carrying instance and a source-carrying platform module, the CLI's runtime identity and the resolved skew policy; matching, execution and diagnostics happen inside that single build. The CLI MUST NOT carry its own component-matching, transformer-execution, or render-finalization implementation, MUST NOT hold any built platform value between renders, and MUST NOT import `opm-controller` Go packages (enhancement 0006). The rendered resource set is the render result's compiled objects.

After a successful render and before any resource, digest or output is built, the CLI SHALL check the compiled objects for shared Kubernetes apply identities (apiVersion, kind, namespace and name) through the library's duplicate-identity helper, and SHALL refuse the render as a validation failure carrying the library's error when any identity is shared, so a build refuses exactly what an apply would have written twice and the CLI and the controller refuse the same module with the same wording (enhancement 0015). The refusal SHALL print the library's header line under the render-failed header and each shared identity, with every producing component and transformer, as details.

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

#### Scenario: Two registrations in one module are refused by every render

- **WHEN** a module's components render two `TransformerRegistration` objects with one name and `opm module build`, `opm module apply` or `opm instance diff` runs against it
- **THEN** the command exits as a validation failure, prints `render failed` with the library's header, and lists the identity once with both components and their transformers as details, and no object is printed, written or compared

#### Scenario: Distinct identities render as before

- **WHEN** every compiled object has a distinct apiVersion, kind, namespace and name
- **THEN** the render proceeds unchanged and no duplicate-identity output appears

### Requirement: Runtime identity in rendered output

The CLI SHALL inject its runtime identity (`opm-cli`) into the kernel render context, so rendered resources carry the CLI's runtime provenance the same way controller-rendered resources carry `opm-controller`.

#### Scenario: Runtime label distinguishes actor

- **WHEN** the CLI renders an instance
- **THEN** the render context's runtime identity SHALL be `opm-cli`

### Requirement: Render digests are kernel-derived and controller-parity

`status.lastAppliedRenderDigest` SHALL be computed over the kernel's rendered objects with the library's shared render digest, `opm/k8s/inventory.RenderDigest`, which the controller also uses. The digest leaves the value of the `app.kubernetes.io/managed-by` label out, so the CLI (`opm-cli`) and the controller (`opm-controller`) digest the same render to the same value, while any other difference in an object, another label included, changes it (enhancement 0012). A registry-gated integration check SHALL verify that the CLI's local-dir staging path and the controller's registry-acquisition call sequence, both ending in `Kernel.Render` against the same platform module, produce identical render digests for the same instance (enhancement 0006). Evaluator-version skew reporting applies to a future cross-binary comparison, where the CLI and controller binaries embed separate CUE evaluators; the in-binary check renders both paths with one evaluator and cannot exhibit skew.

#### Scenario: Parity for the same inputs

- **WHEN** the parity check renders a fixture instance via the CLI kernel path and via the controller's call sequence against the same platform module
- **THEN** the two render digests SHALL be identical

#### Scenario: Skew reported explicitly (cross-binary check, slice C3)

- **WHEN** the future cross-binary parity comparison runs while the `cli` and `opm-controller` binaries embed different `cuelang.org/go` minor versions
- **THEN** the check SHALL fail with a message naming the evaluator-version skew as the suspected cause

#### Scenario: The runtime name does not move the digest

- **WHEN** two renders produce the same objects except that one stamps `app.kubernetes.io/managed-by: opm-cli` and the other `opm-controller`
- **THEN** their render digests SHALL be identical

## ADDED Requirements

### Requirement: Unprovided provider-fulfilled demands are skipped when asked

Every render-bearing command (`opm module build`, `module vet`, `module apply`, `instance build`, `instance vet`, `instance diff`, `instance apply`) SHALL accept `--skip-unprovided`, a boolean defaulting to false, and SHALL pass it to the kernel as the render's skip switch. The switch is the only thing the flag does: which demands are skippable, and what a skip does to the rendered output, is the kernel's rule (core `SPEC.md` §2.1 and §3.1).

- Without the flag, a render SHALL behave exactly as before this change.
- With the flag, the CLI SHALL print one warning per skipped demand after the render. A skipped trait SHALL be worded as `component "<c>": skipped provider-fulfilled trait "<fqn>" (no provider on this platform)`. A component the kernel omitted SHALL be worded once as `component "<c>" not rendered: provider-fulfilled resource "<fqn>" has no provider on this platform`, naming every skipped resource of that component. When the kernel reports alternatives, the warning SHALL name them.
- The skipped-demand warnings SHALL go to the log stream, never to the rendered manifest output of `build`.
- When a render is refused for unresolved demands and at least one of them is marked unprovided by the kernel, the output SHALL add a hint, for any platform source, naming the three ways out: install a provider for the contract, pass `--platform <dir>` with a platform that carries one, or pass `--skip-unprovided` to render the rest.
- `opm instance apply` and `opm module apply` SHALL refuse `--skip-unprovided` for an instance a controller manages, with a validation error saying the controller renders that instance and never skips, before any spec is written.

#### Scenario: Default render still refuses

- **WHEN** `opm module build` runs without `--skip-unprovided` on a module whose component attaches a required provider-fulfilled trait nothing provides
- **THEN** the render SHALL be refused with exit code 2
- **AND** the output SHALL add a hint naming a provider, `--platform <dir>` and `--skip-unprovided`

#### Scenario: A skipped trait renders the rest

- **WHEN** the same command runs with `--skip-unprovided`
- **THEN** the render SHALL succeed and print the component's other objects
- **AND** one warning SHALL name the component and the skipped trait's contract

#### Scenario: A skipped resource drops its component

- **WHEN** a component declares a provider-fulfilled resource nothing provides and `opm instance build --skip-unprovided` runs
- **THEN** no object of that component SHALL be printed
- **AND** one warning SHALL say the component was not rendered and name the resource
- **AND** the other components SHALL render

#### Scenario: A catalog-fulfilled gap still refuses

- **WHEN** `--skip-unprovided` is given and a component demands a catalog-fulfilled contract nothing on the platform implements
- **THEN** the render SHALL be refused as without the flag
- **AND** no `--skip-unprovided` hint SHALL be added for that demand

#### Scenario: A cluster without the provider names the flag

- **WHEN** `opm instance apply` resolves the cluster `Platform` and is refused for a required provider-fulfilled trait no registration provides
- **THEN** the hint SHALL name installing a provider, `--platform <dir>` and `--skip-unprovided`

#### Scenario: A controller-managed instance refuses the flag

- **WHEN** `opm instance apply --skip-unprovided` targets an instance whose ModuleInstance a controller owns
- **THEN** the command SHALL fail with exit code 2 before writing the spec
- **AND** the message SHALL say the controller renders the instance and does not skip demands
