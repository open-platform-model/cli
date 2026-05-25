## Context

The CLI already supports two ways to render a module to manifests:

1. **`opm release apply <release.cue>`** — loads a user-authored release file and applies it.
2. **`opm module build [path]`** — synthesizes a `#ModuleRelease` from a module package directory (using `debugValues` or `-f` overrides) and renders manifests.

The asymmetric hole is `opm module apply` — there is no way to *deploy* a module directory without first authoring a release.cue.

Crucially, the synthesis pipeline already produces a fully-typed `*ModuleRelease` with a **deterministic UUID** computed in CUE:

```
module.metadata.uuid    = SHA1(OPMNamespace, fqn)
release.metadata.uuid   = SHA1(OPMNamespace, "<moduleUUID>:<name>:<namespace>")
```

So re-running `opm module apply ./foo` with the same `--name`/`--namespace` yields the same release UUID → existing inventory secret found → upgrade / prune / dry-run semantics all work identically to `opm release apply`. The synthetic release is not "second-class"; it just happens to be authored by the CLI instead of the user.

After `render.FromModule` returns a `*render.Result`, the type is identical to what `render.FromReleaseFile` returns, and `workflowapply.Execute` consumes either without distinction.

## Goals / Non-Goals

**Goals:**

- Add `opm module apply [path]` (alias `opm mod apply`) as the deploy counterpart to `opm module build`.
- Preserve full semantic equivalence with `opm release apply` after the render stage: inventory secret, stale-resource pruning, server-side apply, dry-run, ownership checks.
- Match the `opm release apply` flag surface exactly (plus `--name` from `opm module build`) so users who know one know both.
- Keep the implementation thin: the cobra command should be a near-clone of `internal/cmd/release/apply.go` with `FromReleaseFile` swapped for `FromModule`.

**Non-Goals:**

- A safety gate on cluster context. The module author is responsible for which cluster they deploy to — same posture as `opm release apply` today.
- A `--wait`/`--timeout` flag. `opm release apply` does not have one; adding it only to `module apply` creates asymmetry. Track separately if needed.
- Output-formatting flags (`--output`, `--split`, `--out-dir`). These belong on `opm module build`, not on an apply command that mutates the cluster.
- Auto-migration tooling for converting a synthetic release into a real `release.cue` file. Out of scope; covered by user documentation.
- Modifying the synthesis flow itself (`loader.SynthesizeModuleReleaseFromPackage`, `render.FromModule`). They already produce the right shape.

## Decisions

### Decision 1: Reuse `workflowapply.Execute` verbatim — no new apply path

`render.FromModule(...) (*Result, error)` and `render.FromReleaseFile(...) (*Result, error)` return the same `*Result` shape. `workflowapply.Execute` takes that result and a `ChangeDescriptor` and does not distinguish between synthesized and file-loaded releases. Therefore `opm module apply` calls `workflowapply.Execute` with no new code paths in the apply package.

**Alternatives considered:**

- *Separate apply path for synthetic releases* — Rejected. Doubles the surface area for no gain. The whole point of "shortcut equivalent" is that the apply pipeline doesn't care how the release was authored.
- *Introduce a "debug-deploy" mode that skips inventory* — Rejected. Loses upgrade and prune semantics, contradicts the user requirement that synthetic apply must behave like real apply.

### Decision 2: Synthetic release identity is deterministic in CUE, not the CLI

Release UUID derivation happens in `catalog/core/v1alpha1/modulerelease/module_release.cue`. The CLI does nothing special. The UUID is stable as long as `(module-fqn, name, namespace)` is stable.

**Consequence:** `--name` and `--namespace` participate in release identity. `opm module apply ./foo` and `opm module apply ./foo --name bar` are *two different releases*, each with its own inventory secret. This is correct (it mirrors how naming a release in a `release.cue` works) but is documented in the help text so users do not expect `--name` to act as a "rename" operation.

### Decision 3: `ChangeDescriptor` fields for inventory

The apply workflow writes a `ChangeDescriptor` into the inventory record so cluster operators can later see how the release was produced. For module-apply:

| Field | Value | Rationale |
| --- | --- | --- |
| `Path` | absolute module directory path | Points back to the source even though no release file exists. |
| `ValuesStr` | empty | No release-file values file; `-f` values are not surfaced here today (matches `release apply` behavior). |
| `Version` | `result.Module.Version` (whatever `metadata.version` decodes to in the module CUE) | Honest representation; may be empty during dev. |
| `Local` | `true` | The module is loaded from disk, not pulled from a registry. |

**Alternatives considered:**

- *Synthesize a fake file path like `<modulePath>/[synthetic]`* — Rejected. Cluster operators reading inventory should see real paths, not synthetic decorations.
- *Include the resolved `<module>-debug` synth name in `Path`* — Rejected. The synth name is already in `Result.Release.Name`; duplicating it adds noise.

### Decision 4: Help text is the safety mechanism

There is no `--debug-only` flag, no cluster annotation check, no kubectl context name heuristic. The user has the same authority over `opm module apply` as over `opm release apply`. The mitigation is in the help text:

- Synth release naming convention is called out.
- The graduation path (`opm release delete <module>-debug` before authoring a real `release.cue`) is mentioned.
- The apply summary line in the log includes the synthetic release name so accidental runs are visible in scrollback.

### Decision 5: Flag-naming alignment with `opm module build`

`opm module build` uses `--name` for the synthetic-release name override. `RenderFlags` has a `--release-name` flag that `module build` does not use today. `opm module apply` will use `--name` to match `module build` (consistency within the module subcommand group). Resolving the `--release-name` vs `--name` inconsistency at the `RenderFlags` level is out of scope.

### Decision 6: Subcommand placement

`opm module apply` lives at `internal/cmd/module/apply.go` and is registered in `NewModuleCmd` (alongside `init`, `vet`, `build`). It does **not** live under `release/`. The `module` group is the home for "starting from a module package directory"; the `release` group is the home for "starting from a release file." `opm release apply` remains strict (release-file-only) — symmetry with `opm release build`'s polymorphism is a separate decision the user explicitly deferred.

## Risks / Trade-offs

- **Risk:** A module author runs `opm module apply ./foo` against the wrong cluster context (e.g., prod).
  → **Mitigation:** Apply summary log includes the synthetic release name (`foo-debug` is visually obvious as a debug-flavored name); user must have explicitly configured the kubeconfig/context. Same risk as `opm release apply` today; same mitigation: trust the operator.

- **Risk:** A module author hacks via `opm module apply ./foo`, then later authors `releases/prod/foo/release.cue` with a different name (`foo-prod`). The synthetic `foo-debug` release's inventory secret + cluster resources persist as orphaned.
  → **Mitigation:** Documented in help text — "before switching to `opm release apply`, run `opm release delete <module>-debug`." No tool automation; this is a known small footgun consistent with the "shortcut" framing.

- **Risk:** `--name` looks like a "rename" operation but actually creates a new release because it participates in the UUID hash.
  → **Mitigation:** Help text explicitly notes that `--name` and `--namespace` participate in release identity; the apply log line shows the resolved synth name so the effect is visible.

- **Trade-off:** Two surfaces (`module apply` and `release apply`) can deploy the same logical module to the same cluster if the user picks the same release name in both. Inventory and ownership checks ensure the *second* tool will detect and refuse to stomp the first's resources (per existing `EnsureCLIMutable` logic). No new corruption risk; just a UX edge case.

- **Trade-off:** Help-text-as-safety is weaker than a flag gate. Acceptable per user decision; revisit if support tickets accumulate.

## Migration Plan

Pure addition. No existing flows change. No data migration. No rollback steps beyond reverting the commit. Existing `opm release apply` users are unaffected.

## Open Questions

None. All flag-set and semantic decisions are locked. Implementation can proceed.
