## MODIFIED Requirements

### Requirement: All renders go through the library kernel

The CLI SHALL render instances exclusively through `github.com/open-platform-model/library`'s kernel: one `Kernel.Render` call per render, taking a source-carrying instance and a source-carrying platform module, the CLI's runtime identity and the resolved skew policy; matching, execution and diagnostics happen inside that single build. The CLI MUST NOT carry its own component-matching, transformer-execution, or render-finalization implementation, MUST NOT hold any built platform value between renders, and MUST NOT import `opm-operator` Go packages (0006 D13). The rendered resource set is the render result's compiled objects.

After a successful render and before any resource, digest or output is built, the CLI SHALL check the compiled objects for shared Kubernetes apply identities (apiVersion, kind, namespace and name) through the library's duplicate-identity helper, and SHALL refuse the render as a validation failure carrying the library's error when any identity is shared, so a build refuses exactly what an apply would have written twice and the CLI and the operator refuse the same module with the same wording (enhancement 0015 D15). The refusal SHALL print the library's header line under the render-failed header and each shared identity, with every producing component and transformer, as details.

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

