# Render path: platform source and render flags

Covers how a render-bearing command picks its platform, which commands touch the cluster, the
`--skip-unprovided` flag, and the skew policy (enhancements 0019 and 0021). It does not cover the
kernel's matching rules; those are the library's.

## Contents

- Schema line
- Platform source precedence
- Per command behavior
- The deps platform and the cache
- Skew and the core floor
- Module path arguments
- `--skip-unprovided`
- Pitfalls
- Check before you finish

## Schema line

The cli embeds the library on the core v2 line. The cluster Platform CR surface is scalar
subscriptions (`{enable?, version!}`, registry keys carry the catalog's major suffix), and module
identity is read verbatim from core-v2 metadata (`metadata.modulePath` is the complete registry
address). CUE fixtures lag the shipped pins until the next `task -x deps:cascade` run moves them, so
read a fixture's own `cue.mod` for its versions.

## Platform source precedence

Every render-bearing command resolves a platform module directory in this order
(`internal/platform.Resolve`):

1. `--platform <dir>`.
2. The cluster `Platform` CR named `cluster`.
3. A platform generated from the render's own committed dependency pins: the instance package's
   `cue.mod/module.cue` for `instance` commands, the module's for `module` commands.

The command acquires the directory once with the kernel's `AcquirePlatformFromDir` and renders with
the single `Kernel.Render` call (`internal/workflow/render`). The cli holds no built platform value
and carries no matching or transformer execution.

The one exception is `opm controller install`. It renders the controller module against its own
dependency pins only (`render.ModuleOpts.DepsOnly`): no `--platform` and no cluster Platform read, so
repairing the controller never depends on the Platform it serves (enhancement 0021). Only
`opm controller install` seeds a Platform. No other render-bearing command creates one.

## Per command behavior

| Command | Reads the cluster | Cluster unreachable or Platform absent |
| --- | --- | --- |
| `module build`, `module vet` | Never | Not applicable; renders for the module's author |
| `instance build`, `instance vet` | When a kubeconfig context resolves (`--kubeconfig`, `--context`, bounded to 10 s) | Warns and falls back to the deps; `--offline` skips the cluster |
| `instance diff`, `instance apply`, `module apply` | Yes | Fall back to the deps only when the cluster has no readable Platform; an unreachable cluster fails them |

`opm instance` commands take only instance packages and decide by package kind. A module package is
refused, naming the matching `opm module` command (a shared helper in `internal/cmdutil/`).

## The deps platform and the cache

The cluster CR and the deps are turned into a module first, through the library's
`opm/helper/platformmodule` generator (byte-identical to the controller's). The result is cached under
`~/.opm/cache/platforms/<content-hash>/`. The cache is idempotent derived state, safe to delete, and
moves with `--config`.

The deps platform carries one registry entry per `opmodel.dev/catalogs/*` pin, closed over with
`platformmodule.Closure`, with core floored at the kernel's verified release. It adds the context's
own `local-module.cue` replacements of paths it pins. Skew cannot arise against it and is not
checked. The provenance line names the source (`module deps` or `instance deps` for the fallback).

## Skew and the core floor

- Catalog version skew follows the config file's `skewPolicy` (`warn` by default, or `refuse`) for
  `--platform` directories. The cluster CR's `spec.skewPolicy` wins when the CR is the source.
- A `--platform` module that pins a core older than a field the kernel reads is refused before the
  render, with the library's message plus a hint naming the directory and the
  `cue mod get` re-pin command.
- There is no local default platform. `opm config init` writes `~/.opm/config.cue` only, and no
  command reads a platform from the OPM home. `opm config vet` warns about a platform directory an
  older release left there. The repo's maintained platform module is `hack/platform/`, passed with
  `--platform` by the offline tests. the kind Platform manifest in `hack/` mirrors its pins in the same commit,
  and `task -x deps:cascade` moves both.

## Module path arguments

`opm module build` and `opm module apply` (not `vet`) also take a published module path
(`internal/modref`). The module is acquired with `AcquireModuleFromRegistry` and synthesized exactly
as a directory module, with no local module context: no replacements and no local render provenance.
A helper in `internal/cmdutil/` classifies the positional argument.

## `--skip-unprovided`

Every render-bearing command takes `--skip-unprovided` (default false). It passes straight to the
kernel as `RenderInput.SkipUnprovided`. A demand for a provider-fulfilled contract that no enabled
catalog provides is then skipped instead of refused. A skipped trait leaves its component rendering;
a skipped resource drops the whole component. Which demands are skippable is the kernel's rule.

The cli only:

- words the `Diagnostics.Skipped` rows (as warnings on the log stream);
- on a refusal with a row marked `Unprovided`, adds a hint naming the three ways out
  for every platform source;
- records skips on the ModuleInstance in `instance apply` and `module apply`, as the annotation
  `module-instance.opmodel.dev/skipped-contracts` (sorted `<component>=<fqn>` pairs; omitted, and so
  cleared by server-side apply, when nothing was skipped);
- refuses the flag for a controller-managed instance, because the controller renders it and never
  skips.

A platform whose enabled registry entries share a contract key is refused by the kernel, whatever the
instance and whatever `--skip-unprovided` says. The cli prints the kernel's collision rows first
(in `internal/workflow/render/`) and adds no hint.

## Pitfalls

- Adding matching or transformer code to the cli to work around a render gap: it is a library defect.
  Fix it upstream.
- Making `controller install` read the cluster Platform: it must stay deps-only (enhancement 0021).
- Treating the platform cache as state: it is derived and safe to delete.
- Letting `instance build` fail on an unreachable cluster: it warns and falls back.

## Check before you finish

- [ ] The change keeps one `Kernel.Render` call and no cli-side platform value.
- [ ] A new render-bearing command follows the precedence and the per command table above.
- [ ] Help text for a changed flag passes `task docs:bundle:check`.
- [ ] Offline tests pass `--platform hack/platform` or a fixture platform.
