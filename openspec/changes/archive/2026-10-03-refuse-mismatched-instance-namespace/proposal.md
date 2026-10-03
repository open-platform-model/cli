## Why

`opm instance apply ./jellyfin -n staging`, with an instance file that declares `metadata.namespace: "media"`, succeeds and leaves the instance split across two namespaces. `renderInstance` copies the kernel's instance metadata and then overwrites `result.Instance.Namespace` with the resolved namespace whenever it came from `--namespace` or `OPM_NAMESPACE` (`internal/workflow/render/render.go:259-266`), with no comparison. The rendered resources keep the kernel's `metadata.namespace` (`media`), and so do the instance fqn and uuid: core makes the field required (`namespace!`, `core/src/module_instance.cue:18`) and derives both from it (`:50-56`). The `ModuleInstance` record, however, is written to `staging`. Later `instance status`, `delete` and prune by file look in the file's namespace (`internal/cmdutil/instance_arg.go:48-53`), miss the record, and the resources are orphaned.

`OPM_NAMESPACE` is worse than the flag, because the by-file query commands never honour it: `OPM_NAMESPACE=other opm instance apply f` writes the record to `other`, while `OPM_NAMESPACE=other opm instance status f` looks in the file's namespace.

The instance file owns its namespace. `#ModuleInstance` (named `#ModuleRelease` in the ADR) carries the authoritative deployed namespace (library `adr/001-module-default-namespace-as-annotation.md`, ADR-001), and the namespace is part of the instance's identity. Owner decision 2026-10-03 (i5): refuse when `-n`/`OPM_NAMESPACE` disagrees with the instance file's `metadata.namespace`; the module path is unaffected.

## What Changes

- The instance-file render path (`render.FromInstanceFile`, shared by `opm instance apply`, `build`, `diff` and `vet`) refuses, with exit code 2, when the resolved namespace came from `--namespace` or `OPM_NAMESPACE` and differs from the acquired instance's `metadata.namespace`. The refusal names both namespaces and where the override came from, and tells the user to edit `metadata.namespace` in the instance file. It runs after the instance is acquired and before platform resolution, so nothing is rendered, applied or diffed.
- A namespace from `~/.opm/config.cue` or the built-in default is not an override and is never compared: those commands keep using the file's namespace.
- An override equal to the file's namespace is accepted as today.
- The module path (`opm module build`, `apply`, `vet` with a directory or a published module) is unchanged: it synthesizes the instance in the override namespace, so the record, resources and fqn already agree.
- The `-n` help text of `opm instance apply`, `diff` and `build` says the value must equal the instance file's `metadata.namespace`. `opm instance vet` shares the render path and therefore the refusal, so its help text and its `-n production` example change with it. The decision names apply, diff and build only; including vet is the supervisor's reading of i5 (vet shares the guard, so leaving its help as "Target namespace" beside an example that now exits 2 would ship a self-contradictory CLI). If the owner wants vet exempt, an opt-out field on `InstanceFileOpts` would drop it from the requirement and the help-text scenario. The generated command reference is regenerated.
- The skip-unprovided integration program applies its fixture with `-n opm-skip-unprovided-itest` while the fixture declares `namespace: "default"`, which is exactly the split this change refuses. The fixture's `metadata.namespace` moves to `opm-skip-unprovided-itest`.

Release class: breaking during beta, and the PR title carries `feat(render)!` with a `BREAKING CHANGE:` footer that the CHANGELOG shows. During beta it ships as the next `1.0.0-beta.N` and never moves the module path. For `opm instance apply` and `diff` the refused invocation produced a broken split instance, but `opm instance build -n X` and `opm instance vet -n X` used to produce correct output and now exit 2, and an exported `OPM_NAMESPACE` now makes `build --offline` and `vet` refuse every instance file in another namespace. That is a behaviour change of a flag and an environment variable, so it ships as a break, not a fix. Migration: drop the override, or edit `metadata.namespace` in the instance file.

Delivery: one PR, default.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `inst-commands`: adds the requirement that the instance render commands refuse a `--namespace`/`OPM_NAMESPACE` override that disagrees with the instance file's `metadata.namespace`, while the module commands keep honouring the override.

## Impact

- Commands: `opm instance apply`, `build`, `diff`, `vet` (refusal and `-n` help text). `opm module build`, `apply`, `vet` unchanged.
- Packages: `internal/workflow/render` (the guard and its tests), `internal/cmd/instance` (help text), `docs/site/reference/cli/opm-instance.md` (regenerated), `internal/workflow/render/testdata/skip-unprovided/instance/instance.cue` (fixture namespace).
- No library, core or operator change. No new flag.
