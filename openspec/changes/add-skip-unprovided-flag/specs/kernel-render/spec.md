## ADDED Requirements

### Requirement: Unprovided provider-fulfilled demands are skipped on request

Every render-bearing command (`opm module build`, `module vet`, `module apply`, `instance build`, `instance vet`, `instance diff`, `instance apply`) SHALL accept `--skip-unprovided`, a boolean defaulting to false, and SHALL pass it to the kernel as the render's skip switch. The switch is the only thing the flag does: which demands are skippable, and what a skip does to the rendered output, is the kernel's rule (core `SPEC.md` §2.1 and §3.1).

- Without the flag, a render SHALL behave exactly as before this change.
- With the flag, the CLI SHALL print one warning per skipped demand after the render. A skipped trait SHALL be worded as `component "<c>": skipped provider-fulfilled trait "<fqn>" (no provider on this platform)`. A component the kernel omitted SHALL be worded once as `component "<c>" not rendered: provider-fulfilled resource "<fqn>" has no provider on this platform`, naming every skipped resource of that component. When the kernel reports alternatives, the warning SHALL name them.
- The skipped-demand warnings SHALL go to the log stream, never to the rendered manifest output of `build`.
- When a render is refused for unresolved demands and at least one of them is marked unprovided by the kernel, the output SHALL add a hint, for any platform source, naming the three ways out: install a provider for the contract, pass `--platform <dir>` with a platform that carries one, or pass `--skip-unprovided` to render the rest.
- `opm instance apply` and `opm module apply` SHALL refuse `--skip-unprovided` for an instance an operator manages, with a validation error saying the operator renders that instance and never skips, before any spec is written.

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

#### Scenario: An operator-managed instance refuses the flag

- **WHEN** `opm instance apply --skip-unprovided` targets an instance whose ModuleInstance an operator owns
- **THEN** the command SHALL fail with exit code 2 before writing the spec
- **AND** the message SHALL say the operator renders the instance and does not skip demands
