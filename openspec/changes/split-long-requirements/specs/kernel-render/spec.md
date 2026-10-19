## REMOVED Requirements

### Requirement: All renders go through the library kernel
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "The CLI renders instances through one Kernel.Render call", "The CLI carries no matching, execution or finalization of its own and imports no opm-operator package", "The CLI checks the compiled objects for shared apply identities", "A shared apply identity is refused as a validation failure", "The refusal of a shared identity prints the library header line and each shared identity", "The CLI words the advisory facts of the kernel as warnings", "The kernel-derived warnings name the same facts as before".

### Requirement: CLI entry points map onto kernel entry points
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "Each CLI loading path uses its kernel acquire verb", "No acquire call passes per-call load options", "The CLI uses no raw-value loading tier and builds no artifact from a value of its own", "One kernel is constructed per command invocation", "The CLI does not fill values into an evaluated instance from Go".

### Requirement: Render digests are kernel-derived and operator-parity
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "The last applied render digest is the library's shared render digest", "The digest leaves out the value of the managed-by label", "A registry-gated check verifies that the CLI and operator render digests are identical", "Evaluator-version skew applies to a future cross-binary comparison".

### Requirement: Local-replacement renders warn
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "Every render enables the kernel's local replacements", "A platform generated from the render's own deps carries the module's replacements", "The kernel replacement rows are worded as warnings, one per honoured replacement", "A module redirect of a path the platform names is reported inert", "The warnings do not block the render and a context without the file produces none", "Both render entry points enable the local replacements", "The provenance signal keeps its file-presence rule".

### Requirement: Unprovided provider-fulfilled demands are skipped on request
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "Render-bearing commands accept --skip-unprovided and pass it to the kernel", "Without --skip-unprovided a render behaves as before", "With --skip-unprovided a skipped trait is worded as a warning naming the alternatives", "With --skip-unprovided an omitted component is worded once as a warning", "The skipped-demand warnings go to the log stream", "A refusal for unprovided demands adds a hint naming the three ways out", "Instance apply and module apply refuse --skip-unprovided for an operator-managed instance".

### Requirement: A platform pinning an older core is refused as a re-pin
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "A platform pinning an older core fails as a validation failure with the library message", "A --platform directory with an older core gets a hint with the re-pin command", "The CLI does not fall back or retry after an older-core refusal", "A platform the CLI generates is never refused for its core".

### Requirement: A colliding platform is refused naming the entries to disable
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "A colliding platform fails as a validation failure whatever the instance", "A collision refusal lists one row per colliding contract before every other row", "An unresolved demand on a colliding key names the collision", "A platform that is not routable with no explaining row fails with the kernel message", "The CLI does not compute collisions of its own".

## ADDED Requirements

### Requirement: The CLI renders instances through one Kernel.Render call

The CLI SHALL render instances exclusively through `github.com/open-platform-model/library`'s kernel: one `Kernel.Render` call per render, taking a source-carrying instance and a source-carrying platform module, the CLI's runtime identity and the resolved skew policy; matching, execution and diagnostics happen inside that single build. The rendered resource set is the render result's compiled objects.

#### Scenario: Instance apply renders via the kernel

- **WHEN** `opm instance apply <file.cue>` runs
- **THEN** the rendered resource set SHALL be produced by `Kernel.Render`'s output, not by any `pkg/render` code

#### Scenario: Typed render causes keep their exit codes

- **WHEN** a render fails with unresolved demands or unmatched components
- **THEN** the command exits as a validation failure with the kernel's message; a transform error or an over-subscribed provider contract exits the same way, with the diagnostics printed beside the refusal

### Requirement: The CLI carries no matching, execution or finalization of its own and imports no opm-operator package

The CLI MUST NOT carry its own component-matching, transformer-execution, or render-finalization implementation, MUST NOT hold any built platform value between renders, and MUST NOT import `opm-operator` Go packages (enhancement 0006, "The CLI imports `library` only; it does not import `opm-operator` (no controller-runtime, no Flux); supersedes D4's dependency direction").

#### Scenario: No CLI-side match implementation

- **WHEN** the project is compiled
- **THEN** the packages `pkg/render` and `pkg/provider` SHALL NOT exist
- **AND** `pkg/loader` SHALL NOT contain component-to-transformer matching code

### Requirement: The CLI checks the compiled objects for shared apply identities

After a successful render and before any resource, digest or output is built, the CLI SHALL check the compiled objects for shared Kubernetes apply identities (apiVersion, kind, namespace and name) through the library's duplicate-identity helper.

#### Scenario: Distinct identities render as before

- **WHEN** every compiled object has a distinct apiVersion, kind, namespace and name
- **THEN** the render proceeds unchanged and no duplicate-identity output appears

### Requirement: A shared apply identity is refused as a validation failure

After a successful render and before any resource, digest or output is built, the CLI SHALL refuse the render as a validation failure carrying the library's error when any identity is shared, so a build refuses exactly what an apply would have written twice and the CLI and the operator refuse the same module with the same wording (enhancement 0015, "One registration per module").

#### Scenario: Two registrations in one module are refused by every render

- **WHEN** a module's components render two `TransformerRegistration` objects with one name and `opm module build`, `opm module apply` or `opm instance diff` runs against it
- **THEN** the command exits as a validation failure, prints `render failed` with the library's header, and lists the identity once with both components and their transformers as details, and no object is printed, written or compared

### Requirement: The refusal of a shared identity prints the library header line and each shared identity

The refusal of requirement "A shared apply identity is refused as a validation failure" SHALL print the library's header line under the render-failed header and each shared identity, with every producing component and transformer, as details.

#### Scenario: The refusal prints each shared identity
- **WHEN** the render is refused for a shared identity
- **THEN** the refusal SHALL print the library's header line under the render-failed header and each shared identity, with every producing component and transformer, as details

### Requirement: The CLI words the advisory facts of the kernel as warnings

The kernel SHALL report advisory facts as structured rows and SHALL attach no message strings to a render result. The CLI SHALL word them itself and surface them to the user as warnings, never dropping one: an unhandled optional trait is read from the render diagnostics' unhandled-trait table, and a module requiring a newer OPM-namespace build than the platform carries is read from the resolved-versions row marked newer.

#### Scenario: Render warnings reach the user

- **WHEN** a render's module requires a newer catalog build than the platform pins and the skew policy is warn
- **THEN** the command prints a CLI-worded skew warning naming the path and both versions, derived from the resolved-versions row marked newer, and the render proceeds

#### Scenario: Unhandled optional trait warns

- **WHEN** a render succeeds with a trait no matched transformer handles whose effective posture is optional
- **THEN** the command prints a CLI-worded warning naming the component and the trait, derived from the diagnostics' unhandled-trait table

### Requirement: The kernel-derived warnings name the same facts as before

Both warnings of the requirement "The CLI words the advisory facts of the kernel as warnings" SHALL name the same facts the kernel previously named — for skew, the path and both versions.

#### Scenario: Skew refusal is a validation failure

- **WHEN** the skew policy is refuse and a module requires a newer OPM-namespace build than the platform pins
- **THEN** the command fails before evaluation as a validation error naming the path, the module's required version and the platform's version

### Requirement: Each CLI loading path uses its kernel acquire verb

Each CLI loading path SHALL use the corresponding kernel acquire verb: instance files via `AcquireInstanceFromDir` with any `-f` values files passed as its trailing values sources, so the instance the render imports already carries them; local module packages via `AcquireModuleFromDir` + `SynthesizeInstance`; registry references via `AcquireModuleFromRegistry` + `SynthesizeInstance`; every platform source via `AcquirePlatformFromDir`.

#### Scenario: Module build synthesizes through the kernel

- **WHEN** `opm module build <dir>` runs against a module package directory
- **THEN** the module SHALL be acquired by `AcquireModuleFromDir` and the instance produced by kernel `SynthesizeInstance` from it and the resolved values

#### Scenario: Instance file with extra values

- **WHEN** `opm instance build ./instance.cue -f overrides.cue` runs
- **THEN** the instance is acquired from its directory with `overrides.cue` passed as a trailing values source, the render imports that layered package, and the rendered objects reflect the override

#### Scenario: Publish gate acquires through the kernel

- **WHEN** the publish gate checks that a directory holds a valid module package
- **THEN** it SHALL acquire it through a kernel acquire verb, and SHALL classify a refusal by the shape-gate sentinels the library's errors package declares

### Requirement: No acquire call passes per-call load options

No acquire call SHALL pass per-call load options: the registry mapping is supplied once when the kernel is constructed, and it reaches module acquisition and the schema loader from there.

#### Scenario: Registry mapping is supplied once

- **WHEN** any render-bearing command acquires a module, an instance or a platform
- **THEN** the acquire call SHALL take no registry or load-options argument, and the mapping the build resolves against SHALL be the one the kernel was constructed with

### Requirement: The CLI uses no raw-value loading tier and builds no artifact from a value of its own

The CLI MUST NOT reach for a raw-value loading tier or construct an artifact from a value it built itself; a caller needing the underlying CUE value SHALL read the acquired artifact's own package field.

#### Scenario: A caller reads the acquired artifact
- **WHEN** a caller needs the underlying CUE value
- **THEN** it SHALL read the acquired artifact's own package field

### Requirement: One kernel is constructed per command invocation

One `kernel.Kernel` SHALL be constructed per command invocation and threaded through the workflow; packages needing a `*cue.Context` SHALL receive the one the kernel's schema value was built in, never a second one of their own.

#### Scenario: Single kernel per invocation

- **WHEN** any render-bearing command executes
- **THEN** exactly one `kernel.Kernel` SHALL be constructed for that invocation
- **AND** no code path SHALL construct a second `cuecontext.New()` for render work

### Requirement: The CLI does not fill values into an evaluated instance from Go

The CLI MUST NOT fill values into an evaluated instance from Go.

#### Scenario: No value is filled from Go
- **WHEN** the CLI holds an evaluated instance
- **THEN** the CLI MUST NOT fill values into it from Go

### Requirement: The last applied render digest is the library's shared render digest

`status.lastAppliedRenderDigest` SHALL be computed over the kernel's rendered objects with the library's shared render digest, `opm/k8s/inventory.RenderDigest`, which the operator also uses.

#### Scenario: The digest is computed by the shared function
- **WHEN** the CLI computes `status.lastAppliedRenderDigest`
- **THEN** it SHALL be computed over the kernel's rendered objects with the library's shared render digest, `opm/k8s/inventory.RenderDigest`, which the operator also uses

### Requirement: The digest leaves out the value of the managed-by label

`status.lastAppliedRenderDigest` SHALL leave the value of the `app.kubernetes.io/managed-by` label out, so the CLI (`opm-cli`) and the operator (`opm-controller`) digest the same render to the same value, while any other difference in an object, another label included, changes it (enhancement 0012, "Labels are stamped by the CUE render; the shared render digest ignores the runtime-name label value").

#### Scenario: The runtime name does not move the digest

- **WHEN** two renders produce the same objects except that one stamps `app.kubernetes.io/managed-by: opm-cli` and the other `opm-controller`
- **THEN** their render digests SHALL be identical

### Requirement: A registry-gated check verifies that the CLI and operator render digests are identical

A registry-gated integration check SHALL verify that the CLI's local-dir staging path and the operator's registry-acquisition call sequence, both ending in `Kernel.Render` against the same platform module, produce identical render digests for the same instance (enhancement 0006 ("The OQ1 render-digest parity experiment is deferred into the C2 kernel-adoption slice; closes the last `draft → accepted` gate") gate).

#### Scenario: Parity for the same inputs

- **WHEN** the parity check renders a fixture instance via the CLI kernel path and via the operator's call sequence against the same platform module
- **THEN** the two render digests SHALL be identical

### Requirement: Evaluator-version skew applies to a future cross-binary comparison

Evaluator-version skew reporting SHALL apply to a future cross-binary comparison, where the CLI and operator binaries embed separate CUE evaluators; the in-binary check renders both paths with one evaluator and cannot exhibit skew.

#### Scenario: Skew reported explicitly (cross-binary check, slice C3)

- **WHEN** the future cross-binary parity comparison runs while the `cli` and `opm-operator` binaries embed different `cuelang.org/go` minor versions
- **THEN** the check SHALL fail with a message naming the evaluator-version skew as the suspected cause

### Requirement: Every render enables the kernel's local replacements

Every render SHALL enable the kernel's local replacements, so a `cue.mod/local-module.cue` in the module context (the main module of an instance-file render, or the module directory of a module render) or in the platform module directory reaches the render build under the kernel's precedence: the platform's replacements whole, the module's only for paths the platform's dependency list does not name. Source: enhancement 0010 (a moved-ahead dev checkout is accepted and warned about).

#### Scenario: Platform replacement of its catalog is honoured and warns

- **WHEN** a render's platform module directory (`--platform <dir>`) carries `cue.mod/local-module.cue` replacing its catalog path with a checkout
- **THEN** the rendered objects reflect the checkout's transformer bytes
- **AND** a warning names the catalog path, the checkout directory and the platform as its source

#### Scenario: A malformed local file is a validation failure

- **WHEN** the module context's `cue.mod/local-module.cue` does not parse, or a replacement directory's `cue.mod/module.cue` declares a different module path than the replaced one
- **THEN** the command fails as a validation error with the kernel's message naming the file or the two paths, and no objects are rendered

### Requirement: A platform generated from the render's own deps carries the module's replacements

When the platform is generated from the render's own deps (`opm module build` and `opm module vet` without `--platform`, and every other render that falls back to its deps), the module context's replacements of paths that platform pins SHALL be carried into it (see the `platform-resolution` capability), so they are honoured rather than overridden. Source: enhancement 0010 (a moved-ahead dev checkout is accepted and warned about).

#### Scenario: Module deps platform honours the module's replacement

- **WHEN** `opm module build` runs with no `--platform` from a module directory whose `cue.mod/local-module.cue` replaces `opmodel.dev/catalogs/opm@v4` with a checkout
- **THEN** the rendered objects reflect the checkout's transformer bytes
- **AND** one warning names the catalog path, the checkout directory and the module as its source, with the published-bytes caveat
- **AND** no warning says the redirect is ignored

#### Scenario: Instance deps platform honours the instance package's replacement

- **WHEN** `opm instance build --offline` runs on an instance package whose `cue.mod/local-module.cue` replaces `opmodel.dev/catalogs/opm@v4` with a checkout
- **THEN** the rendered objects reflect the checkout's transformer bytes
- **AND** one warning names the catalog path, the checkout directory and the module as its source

### Requirement: The kernel replacement rows are worded as warnings, one per honoured replacement

The CLI SHALL word the kernel's replacement rows as warnings, one per honoured replacement, naming the replaced path, the directory or module it was served from and whether the module or the platform supplied it, and stating that the rendered bytes may not correspond to any published build; a carried replacement SHALL be attributed to the module. Source: enhancement 0010 (a moved-ahead dev checkout is accepted and warned about).

#### Scenario: Replaced dependency warns

- **WHEN** an instance render's main module carries `cue.mod/local-module.cue` replacing a dependency the platform does not name with a local checkout
- **THEN** the rendered objects reflect the checkout's bytes
- **AND** a warning names the path, the checkout directory and the module as its source, with the published-bytes caveat
- **AND** the render proceeds

### Requirement: A module redirect of a path the platform names is reported inert

For each `replaceWith` in the module context's own `cue.mod/local-module.cue` that no row honours with the same target, the CLI SHALL warn that the platform names that path and that the redirect belongs in the platform module's `cue.mod/local-module.cue`. Source: enhancement 0010 (a moved-ahead dev checkout is accepted and warned about).

#### Scenario: Module-side replacement of a platform path is reported inert

- **WHEN** a render against a platform the render does not generate (`--platform <dir>` or the cluster Platform) runs from a module context whose `cue.mod/local-module.cue` replaces a path that platform's dependency list names
- **THEN** the rendered objects reflect the platform's pinned bytes
- **AND** a warning names the path and says the redirect belongs in the platform module's `cue.mod/local-module.cue`

### Requirement: The warnings do not block the render and a context without the file produces none

The local-replacement warnings SHALL NOT block the render or alter its output. A context without the `cue.mod/local-module.cue` file SHALL produce no such warning. Source: enhancement 0010 (a moved-ahead dev checkout is accepted and warned about).

#### Scenario: Clean context stays silent

- **WHEN** neither the module context nor the platform module carries a `local-module.cue` with a `replaceWith`
- **THEN** no such warning is emitted and the output is identical to a render without the opt-in

### Requirement: Both render entry points enable the local replacements

Both render entry points SHALL enable the kernel's local replacements. Source: enhancement 0010 (a moved-ahead dev checkout is accepted and warned about).

#### Scenario: Both entry points enable the replacements
- **WHEN** a render runs from either render entry point
- **THEN** both render entry points SHALL perform this

### Requirement: The provenance signal keeps its file-presence rule

The provenance signal (`SourceLocal`, the `module-instance.opmodel.dev/source: local` annotation) SHALL keep its file-presence rule. Source: enhancement 0010 (a moved-ahead dev checkout is accepted and warned about).

#### Scenario: The provenance signal follows the file
- **WHEN** a module context carries a `cue.mod/local-module.cue`
- **THEN** the provenance signal (`SourceLocal`, the `module-instance.opmodel.dev/source: local` annotation) keeps its file-presence rule

### Requirement: Render-bearing commands accept --skip-unprovided and pass it to the kernel

Every render-bearing command (`opm module build`, `module vet`, `module apply`, `instance build`, `instance vet`, `instance diff`, `instance apply`) SHALL accept `--skip-unprovided`, a boolean defaulting to false, and SHALL pass it to the kernel as the render's skip switch. The switch SHALL be the only thing the flag does: which demands are skippable, and what a skip does to the rendered output, is the kernel's rule (core `SPEC.md` §2.1 and §3.1).

#### Scenario: A catalog-fulfilled gap still refuses

- **WHEN** `--skip-unprovided` is given and a component demands a catalog-fulfilled contract nothing on the platform implements
- **THEN** the render SHALL be refused as without the flag
- **AND** no `--skip-unprovided` hint SHALL be added for that demand

### Requirement: Without --skip-unprovided a render behaves as before

Without the flag, a render SHALL behave exactly as before this change.

#### Scenario: Default render still refuses

- **WHEN** `opm module build` runs without `--skip-unprovided` on a module whose component attaches a required provider-fulfilled trait nothing provides
- **THEN** the render SHALL be refused with exit code 2
- **AND** the output SHALL add a hint naming a provider, `--platform <dir>` and `--skip-unprovided`

### Requirement: With --skip-unprovided a skipped trait is worded as a warning naming the alternatives

With the flag, the CLI SHALL print one warning per skipped demand after the render. A skipped trait SHALL be worded as `component "<c>": skipped provider-fulfilled trait "<fqn>" (no provider on this platform)`. When the kernel reports alternatives, the warning SHALL name them.

#### Scenario: A skipped trait renders the rest

- **WHEN** the same command runs with `--skip-unprovided`
- **THEN** the render SHALL succeed and print the component's other objects
- **AND** one warning SHALL name the component and the skipped trait's contract

### Requirement: With --skip-unprovided an omitted component is worded once as a warning

With the flag, the CLI SHALL print one warning per skipped demand after the render. A component the kernel omitted SHALL be worded once as `component "<c>" not rendered: provider-fulfilled resource "<fqn>" has no provider on this platform`, naming every skipped resource of that component.

#### Scenario: A skipped resource drops its component

- **WHEN** a component declares a provider-fulfilled resource nothing provides and `opm instance build --skip-unprovided` runs
- **THEN** no object of that component SHALL be printed
- **AND** one warning SHALL say the component was not rendered and name the resource
- **AND** the other components SHALL render

### Requirement: The skipped-demand warnings go to the log stream

The skipped-demand warnings SHALL go to the log stream, never to the rendered manifest output of `build`.

#### Scenario: Warnings stay out of the manifest
- **WHEN** a render skips an unprovided demand
- **THEN** the skipped-demand warnings SHALL go to the log stream, never to the rendered manifest output of `build`

### Requirement: A refusal for unprovided demands adds a hint naming the three ways out

When a render is refused for unresolved demands and at least one of them is marked unprovided by the kernel, the output SHALL add a hint, for any platform source, naming the three ways out: install a provider for the contract, pass `--platform <dir>` with a platform that carries one, or pass `--skip-unprovided` to render the rest.

#### Scenario: A cluster without the provider names the flag

- **WHEN** `opm instance apply` resolves the cluster `Platform` and is refused for a required provider-fulfilled trait no registration provides
- **THEN** the hint SHALL name installing a provider, `--platform <dir>` and `--skip-unprovided`

### Requirement: Instance apply and module apply refuse --skip-unprovided for an operator-managed instance

`opm instance apply` and `opm module apply` SHALL refuse `--skip-unprovided` for an instance an operator manages, with a validation error saying the operator renders that instance and never skips, before any spec is written.

#### Scenario: An operator-managed instance refuses the flag

- **WHEN** `opm instance apply --skip-unprovided` targets an instance whose ModuleInstance an operator owns
- **THEN** the command SHALL fail with exit code 2 before writing the spec
- **AND** the message SHALL say the operator renders the instance and does not skip demands

### Requirement: A platform pinning an older core fails as a validation failure with the library message

When the kernel refuses a render because the platform module pins a core release older than the first one deriving a field the kernel reads (the library's core-floor refusal, raised before the render is staged), every render-bearing command SHALL fail as a validation failure printing the library's message, which names the platform, the missing field and the core release required.

#### Scenario: The older-core refusal prints the library message
- **WHEN** the kernel refuses a render because the platform module pins a core release older than the first one deriving a field the kernel reads
- **THEN** every render-bearing command SHALL fail as a validation failure printing the library's message, which names the platform, the missing field and the core release required

### Requirement: A --platform directory with an older core gets a hint with the re-pin command

In the refusal of requirement "A platform pinning an older core fails as a validation failure with the library message", when the platform is a directory the user named with `--platform`, the command SHALL add a hint naming that directory and the exact `cue mod get opmodel.dev/core@<version>` command to run in it, where `<version>` is the core release the kernel was verified against (which satisfies every such floor, not only the one that fired).

#### Scenario: An older-core --platform directory is refused with the re-pin command

- **WHEN** `opm instance build` runs with `--platform <dir>` whose `cue.mod/module.cue` pins `opmodel.dev/core@v2` at `v2.0.0-alpha.11`
- **THEN** the command exits as a validation failure, prints `render failed` with the library's message naming `providedBy` and `2.0.0-alpha.12`, prints a hint naming `<dir>` and `cue mod get opmodel.dev/core@v2.0.0-alpha.12`, and prints no manifest

### Requirement: The CLI does not fall back or retry after an older-core refusal

In the refusal of requirement "A platform pinning an older core fails as a validation failure with the library message", the CLI SHALL NOT fall back to another platform source, retry against a generated platform, or render with a count of its own.

#### Scenario: No fallback after the refusal
- **WHEN** the kernel refuses a render for an older core
- **THEN** the CLI SHALL NOT fall back to another platform source, retry against a generated platform, or render with a count of its own

### Requirement: A platform the CLI generates is never refused for its core

A platform the CLI generates (from the cluster Platform or from the render's own dependency pins) SHALL pin the kernel's verified core, so it is never refused the way of requirement "A platform pinning an older core fails as a validation failure with the library message".

#### Scenario: A generated platform is never refused for its core

- **WHEN** a render-bearing command runs with no `--platform` and the platform is generated from the cluster Platform or from the render's dependency pins
- **THEN** the generated platform pins the kernel's verified core release and the render is not refused for an older core

### Requirement: A colliding platform fails as a validation failure whatever the instance

When the kernel refuses a render because more than one enabled registry entry of the platform defines a contract key, every render-bearing command SHALL fail as a validation failure printing the kernel's message under the render-failed header. The refusal applies whatever the instance and whether or not `--skip-unprovided` is passed. Source: enhancement 0026 (interim safety net until side-by-side majors).

#### Scenario: Skipping unprovided demands does not skip a collision

- **WHEN** the same render runs with `--skip-unprovided`
- **THEN** the render is refused with the same collision rows

### Requirement: A collision refusal lists one row per colliding contract before every other row

When the kernel refuses a render because more than one enabled registry entry of the platform defines a contract key, every render-bearing command SHALL fail as a validation failure, followed by one row per colliding contract, printed before every other refusing row, naming the contract key and the registry keys (catalog path plus major) of the entries defining it, as the kernel's diagnostics carry them. Source: enhancement 0026 (interim safety net until side-by-side majors).

#### Scenario: A colliding --platform directory is refused with its rows

- **WHEN** `opm instance build` runs with `--platform <dir>` whose platform enables two majors of one catalog listing the same contract keys
- **THEN** the command exits as a validation failure, prints `render failed` with the kernel's message, prints one `defined by more than one enabled registry entry` row per colliding key naming both registry keys before any other row, and prints no manifest

### Requirement: An unresolved demand on a colliding key names the collision

An unresolved demand on a colliding key SHALL be worded as defined by more than one enabled registry entry, never as implemented by nothing on the platform, and the render SHALL add no platform-source hint beside a collision. Source: enhancement 0026 (interim safety net until side-by-side majors).

#### Scenario: An unresolved demand on a colliding key names the collision

- **WHEN** a refused render carries an unresolved demand whose contract key collides
- **THEN** the demand's row says the key is defined by more than one enabled registry entry and names them, and no hint about the platform source is printed

### Requirement: A platform that is not routable with no explaining row fails with the kernel message

When the kernel refuses a platform as not routable with no row explaining it, the command SHALL fail as a validation failure printing the kernel's message. Source: enhancement 0026 (interim safety net until side-by-side majors).

#### Scenario: A platform not routable with no row
- **WHEN** the kernel refuses a platform as not routable with no row explaining it
- **THEN** the command SHALL fail as a validation failure printing the kernel's message

### Requirement: The CLI does not compute collisions of its own

The CLI SHALL NOT compute collisions of its own. Source: enhancement 0026 (interim safety net until side-by-side majors).

#### Scenario: The CLI trusts the kernel for collisions
- **WHEN** a platform enables entries that define one contract key
- **THEN** the CLI SHALL NOT compute collisions of its own
