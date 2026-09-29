## Why

A component that attaches a provider-fulfilled contract, such as the `backup` trait, cannot render anywhere that has no provider for it: the module author's own `opm module build`, an instance rendered against its own deps, or a cluster without the OPM operator (where no provider is ever registered). Today the only way through is `--platform <dir>` with a platform that carries a provider. Clusters without the operator are supported for basic modules only (user decision 2026-09-29), so a user deploying there, or an author inspecting everything else a module renders, needs to say "render what this platform can, and tell me what you left out".

The kernel gains that switch in the library change `render-skips-unprovided-provider-demands`, under the rule core states in `skip-unprovided-provider-demands`. This change exposes it.

## What Changes

- New flag `--skip-unprovided` (bool, default false) on every render-bearing command: `opm module build`, `module vet`, `module apply`, `instance build`, `instance vet`, `instance diff`, `instance apply`. It sets the kernel's `RenderInput.SkipUnprovided`.
- Each skipped demand is printed as a warning: a skipped trait names the component and the contract; a skipped resource says the component was not rendered at all.
- The refusal hint for an unresolved demand the kernel marks `Unprovided` names the three ways out: install the provider, pass `--platform <dir>` with a platform that carries one, or pass `--skip-unprovided`. It applies to every platform source, not only the deps.
- `opm instance apply` and `opm module apply` record the skipped contracts on the ModuleInstance as the annotation `module-instance.opmodel.dev/skipped-contracts`, and clear it on an apply that skips nothing.
- An apply to an operator-managed instance refuses `--skip-unprovided`: the operator renders that instance and never skips, so the flag would have no effect.
- Library pin raised to the release carrying `SkipUnprovided`.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `kernel-render`: renders may skip unprovided provider-fulfilled demands on request, report each one, and name the flag in the refusal hint.
- `instance-inventory`: the ModuleInstance record carries the skipped contracts of the last apply.

## Impact

- Code: `internal/cmdutil/flags.go` (`RenderFlags`, `InstanceFileFlags`), `internal/workflow/render` (render input, skipped-row output, `refusalHint`), `internal/workflow/apply` (annotation, the thin-editor refusal), `internal/inventory/store.go` (`SpecInput`), the seven commands' help text, `go.mod` (library pin).
- Depends on: library `render-skips-unprovided-provider-demands` released; cli `retire-local-default-platform` merged (this change builds on its resolver and hint code).
- Release class: MINOR, a new flag with a default that changes nothing.
- Operator: unaffected. An instance applied with skips and later handed to the operator is refused by the operator's render until a provider is installed; the annotation shows why.
- Docs in other repos are follow-ups listed in `orchestration.md`.
