## Why

The cli tells a registry failure apart by reading the error's message text, in three places:

- `internal/cuemod/connectivity.go:14-27` `IsConnectivityError` (a `net.Error` in the chain, else the text `cannot do HTTP request`), used by `opm instance init` after `Kernel.AcquireModuleFromRegistry` (`internal/cmd/instance/init.go:283-287`) and after `cuemod.Tidy` (`internal/instinit/write.go:64-71`);
- `internal/publish/check.go:223-231` (`fetchPublishedTree`): the text `not found` after `reg.Fetch` is `ErrNotPublished`, anything else a `*ConnectivityError`;
- `internal/publish/compat.go:225-237` (`loadPublishedPackage`): the text `cannot find module providing package` after `load.Instances` is "absent at this version", anything else a `*ConnectivityError`.

A CUE bump that rewords any of these silently moves an exit code. Library v1.0.0-beta.6 ships the typed classification (`opm/errors`: `FetchKind`, `*FetchError`, `ErrTransient`, `Classify`; library spec `fetch-error-classification`, source 0021:D8:R12), and it is now the only place that reads CUE's registry error text. The owner decided in the beta.1 kernel checklist walkthrough, decision d1: "export opmerrors.Classify(err) for raw cue/load, modconfig and cuemod errors so the cli drops its 3 text probes". This change is the cli half of d1.

The cli still pins library v1.0.0-beta.4, and its cascade receiver runs in dry-run, so no bump PR will arrive on its own. This is the first cli change on beta.6, so it carries the bump. beta.6 deprecates `opm/helper/objectset` in favour of `opm/k8s/object` (same behaviour), and the cli's two imports of it would fail `task lint` (SA1019) after the bump, so the import swap rides with it.

## What Changes

- **Library bump.** `github.com/open-platform-model/library` moves from v1.0.0-beta.4 to v1.0.0-beta.6 (no other module moves). `internal/workflow/render/render.go` and `validation.go` import `opm/k8s/object` instead of the deprecated `opm/helper/objectset`. The duplicate-identity refusal's text is unchanged.
- **Exit codes pinned first.** Before any probe changes, table tests drive each failure form through a real in-memory registry and pin what each site answers today: the connectivity answer, `ErrNotPublished`, "absent" (`found=false`) and the platform build hint. These tests pass unchanged before and after the swap.
- **The three probes read types.** `IsConnectivityError` answers from `Classify`'s kind (`FetchUnreachable`), not text. `fetchPublishedTree` maps `FetchNotFound` from the version lookup to `ErrNotPublished`. `loadPublishedPackage` treats `FetchNotFound` as "absent" only when the probed version or package is the thing that is missing, which it checks by type and by the fetched tree. A dependency of the probed build that the registry does not hold stays a `*ConnectivityError` (exit 3), as today. The `platformBuildHint` "pin a published build" branch reads `FetchNotFound` instead of the text `module not found`; its matches for author defects that no registry interaction caused (`cannot find package`, `cannot expand module graph`, `#registry`) stay, because the library leaves those unclassified on purpose.
- The CUE-wording pin test in `internal/cuemod/connectivity_test.go` is deleted: the library pins those forms against the embedded CUE now. The behaviour tests stay: an unreachable registry is connectivity, an unpublished version is `ErrNotPublished`, an absent package is `found=false`.

**No exit code changes.** Every command keeps the exit code it has today for every failure form, including the two that look odd under the new types and are kept on purpose: a 5xx answer during `opm instance init`'s acquire exits 1 (it is transient in the library's terms, but the registry answered), and a dependency the registry does not hold during the compat walk exits 3.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `instance-init`: states which registry failures count as unreachable (exit 3) while the module is acquired and its dependency closure resolved, and that the cli takes that decision from the library's classification rather than the message text.
- `catalog-registry-check`: states the exit code for an unpublished coordinate (5, which the command already returns) beside the existing 0/2/3, and which registry answers read as unpublished. The command's help text gains the same `5 not published` line; it is the only help text change.
- `artifact-publishing`: the compat walk's connectivity requirement says which not-found answers are the scan's negative signal and which abort as connectivity.

## Impact

- **Release class: `refactor`, PATCH (after GA as well), shipped as the next beta.N.** No flag, command or exit code changes; the only text change is the `opm catalog registry check` help line naming exit 5, which the command already returns. The PR title is the changelog line: `refactor: classify registry failures by the library's typed errors`.
- Commands whose failure paths are touched: `opm instance init`, `opm catalog registry check` (with and without `--compat`), `opm catalog publish` (the compat gate), and every render-bearing command's platform build hint.
- Packages: `internal/cuemod` (`connectivity.go`, a status-registry helper in `cuemodtest`), `internal/publish` (`check.go`, `compat.go`), `internal/config` (`platform.go`), `internal/instinit` (`write.go`, doc comment only), `internal/workflow/render` (`render.go`, `validation.go`, import swap), `go.mod`/`go.sum`, and the comment of `TestE2E_InstanceInit_TidyRegistryFailureShape` in `tests/e2e/instance_init_test.go`.
- Library pin: beta.6 also brings the round-1 to round-3 library work (core pin `opmodel.dev/core@v2.0.0-beta.4` as the kernel default, typed fetch errors, context checks between stages, failure-path values attribution). Section 1 takes the bump on its own and makes the suite green before any probe changes.
- Coordination: no open cli PR bumps the library (checked 2026-10-05: #321 cascade re-pin, #319 release, Dependabot PRs on k8s.io and x/mod). Later cli changes in the same plan follow this one: the module-metadata accessor adoption, then the `opm/k8s/object` tier adoption; both overlap `internal/cmd/instance/init.go` and `internal/workflow/render`.
- Out of scope: making a kernel verb's connectivity failure exit 3 everywhere (for example `opm module build <published path>`), which would change exit codes; any retry; the record-read exit codes (cli#310).
