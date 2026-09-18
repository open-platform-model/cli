## Why

The cluster-backed e2e suite is exercised only on a developer's laptop: the PR `e2e` job runs on a
bare runner with no kind cluster, and `requireKindCluster` skips every cluster test there. So the
local loop is the sole gate on `TestE2E_Operator_InstallUninstallLifecycle` and the operator-owned
tests — and that loop currently violates the principle `e2e-cluster-preconditions` exists to
enforce: a missing preparation step surfaces as an unrelated product error, in a later run, or not
at all.

Three concrete faults, all observed in one sitting on 2026-09-18:

- The suite's stub `~/.opm/config.cue` pins `registry: "localhost:5000"`, a registry nothing starts.
  The lifecycle test's `operator install` therefore fails with `opmodel.dev/catalogs/opm@v4 has no
  published release` — a product-shaped error for a fixture-shaped cause. It is the only
  `localhost:5000` under `tests/`, and the tests written to the current standard already ignore it
  by building an in-process registry and passing `OPM_REGISTRY` explicitly.
- `task cluster:operator` runs `./bin/opm` but never builds it, and no task in the Taskfile declares
  `deps:`. In a fresh worktree the lifecycle test's restore step dies with exit 127.
- When that restore fails, `restoreDevOperator` reports it with `t.Logf`, so the run still exits 0
  and leaves the cluster stripped of its CRDs and operator. The next `task test:integration` then
  fails with `the server could not find the requested resource`, pointing nowhere near the cause.
  The function's own doc comment predicts this outcome and does not act on it.

Together they cost a full diagnosis cycle and two destroyed cluster states before any of it pointed
at its own cause.

## What Changes

- **The e2e suite stops naming a registry that does not exist.** The stub config's `registry` key is
  removed so `config.DefaultRegistry` — the GHCR mapping the CLI actually ships — applies, which is
  already the asserted fallback (`internal/config/loader_test.go`). Tests that need a writable
  registry keep building their own in-process one; nothing gains a dependency on a running daemon.
- **`cluster:operator` builds the CLI it installs with.** The task gains `deps: [build]`. `build` is
  checksum-cached through its `sources`/`generates`, so this is a no-op whenever the binary is
  current, and it makes the task work unattended — which is what a test cleanup calling it requires.
- **A failed operator restore fails the run that broke the cluster.** `restoreDevOperator` reports
  through `t.Errorf` instead of `t.Logf`, so a run that cannot put the dev operator back is red
  where the damage happened rather than in someone else's next run.

- **Not in this change.**
  - *The pinned operator version.* `PinnedOperatorVersion` is `v1.0.0-alpha.14` (2026-08-29), five
    releases behind, and it cannot materialize a scalar-version subscription: `Platform/cluster`
    sits at `Ready=False`/`Stalled=True`, reason `MaterializeFailed`, on a spec that plainly carries
    the version. That is a `fix(deps)` bumping an embedded manifest, and it releases; it belongs in
    its own change.
  - *Tightening `cluster:operator`'s reconciling check.* `kind-cluster-tasks` already requires the
    task to "confirm the operator is genuinely reconciling by waiting for `status.operatorVersion`",
    but the operator stamps that field on every reconcile regardless of outcome, so a stalled
    operator satisfies it and the task reports success against a platform that cannot render. The
    honest check is the `Ready` condition — but with the current pin that check never passes, so
    landing it here would leave `main` red. It is a follow-up gated on the operator bump above.
  - *Giving PR CI a kind cluster.* The coverage gap that let all of this rot is real, but adding a
    cluster to the `e2e` job is a CI-cost decision of its own.

## Capabilities

### New Capabilities

None. Both behaviours already have a capability that owns them.

### Modified Capabilities

- `e2e-cluster-preconditions`: two added requirements — the suite resolves modules through the
  shipped default registry rather than a local one nothing starts, and a destructive test that
  cannot restore the dev operator fails its own run instead of reporting and exiting zero.
- `kind-cluster-tasks`: one added requirement — `cluster:operator` builds the CLI binary it invokes,
  so it succeeds from a clean tree and can be called unattended by a test cleanup.

## Impact

- **Affected files:** `tests/e2e/mod_init_test.go` (stub config), `tests/e2e/operator_test.go`
  (`restoreDevOperator`), `Taskfile.yml` (`cluster:operator`).
- **Affected commands:** none. No product surface changes; `opm` behaves identically.
- **Behaviour change:** `task cluster:operator` builds first when the binary is stale or absent. The
  e2e suite resolves `opmodel.dev` from GHCR instead of an unreachable local address, which is what
  every already-migrated test in the suite does. A destructive e2e run that cannot restore the
  cluster now exits non-zero.
- **SemVer:** none — tests and dev tooling only, no shipped surface. Section commits are
  `chore(taskfile)` and `test(e2e)`, neither of which releases.
- **Complexity (Principle VII):** net negative. One config key removed, one duplicated registry
  constant removed, one `deps:` entry added, one log call changed to an error call. No new
  machinery, no new flags, and no new dependency on a registry daemon.
