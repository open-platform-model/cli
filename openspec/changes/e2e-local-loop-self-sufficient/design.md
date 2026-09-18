# Design: e2e-local-loop-self-sufficient

## Context

See `proposal.md` § Why for motivation. Current state, read 2026-09-18 against this branch:

- `tests/e2e/mod_init_test.go` `TestMain` builds the suite's `opm` into a temp dir and writes a stub
  `$HOME/.opm/config.cue` carrying `registry: "localhost:5000"`. Every `runOPM` / `runOPMWithEnv`
  call runs under that `HOME`.
- The suite already has the pattern that makes a registry a non-issue. `publishRegistryEnv`
  (`publish_test.go`) and `templatesRegistryEnv` (`mod_init_test.go`) each start an in-process
  registry with `modregistrytest.NewServer` over `ocimem`, register `t.Cleanup(reg.Close)`, and
  return an `OPM_REGISTRY` that routes only the domains that test writes to at it while
  `opmodel.dev` stays on GHCR. `mod_init_test.go` calls this "the hermetic inversion". Those tests
  never read the stub's registry value.
- Two consumers never got that treatment. `TestE2E_Operator_InstallUninstallLifecycle` calls
  `runOPMWithEnv` with no `OPM_REGISTRY` and no `--config`, so it falls through to the stub. The
  operator-owned tests avoid the problem a third way: `runOperatorOwnedOPM` passes
  `--config hack/opm-config.cue`, which maps both domains at GHCR.
- `config.DefaultRegistry` (`internal/config/templates.go`) is exactly that GHCR mapping, and
  `internal/config/loader_test.go` already asserts it is what an unset key resolves to.
- `Taskfile.yml` `cluster:operator` invokes `./bin/opm`; no task in the file declares `deps:`.
  `build` declares `sources:` and `generates:`, so Task checksums it.
- `restoreDevOperator` (`tests/e2e/operator_test.go`) shells out to `task cluster:operator` from a
  `t.Cleanup` registered before the reset cleanup, so it runs last, and reports any failure with
  `t.Logf`.
- The PR `e2e` job (`.github/workflows/pr.yml`) creates no cluster, and `requireKindCluster` calls
  `t.Skipf`, so none of the cluster-backed tests have ever run in CI.

### Baseline: the cluster-backed suite before any edit (task 1.1)

Recorded 2026-09-19 with `go test ./tests/e2e/... -v -timeout 25m` from this worktree — a clean
tree with no `bin/opm` — against `kind-opm-dev` prepared and left as `task cluster:operator`
leaves it: CRDs present, `opm-operator-controller-manager` 1/1 available,
`Platform/cluster` at `Ready=False` / `Stalled=True`, reason `MaterializeFailed`,
`status.operatorVersion: v1.0.0-alpha.14`. Suite result `FAIL` in 616s. Every other test passed.

**Failures this change addresses**

- `TestE2E_Operator_InstallUninstallLifecycle/install` — `refused: opmodel.dev/catalogs/opm@v4 has
  no published release`, reported against `registry localhost:5000`. The stub config's registry key,
  fixed by section 2.
- `TestE2E_Operator_InstallUninstallLifecycle/idempotent re-install reports unchanged` — expected 3
  changed resources, got 0; the run installed CRDs only. Cascade of the failed `install`, so it
  clears with section 2.
- `TestE2E_Operator_InstallUninstallLifecycle/uninstall refuses while a finalizer is armed, then
  --remove-finalizers proceeds` — `kubectl get deployment opm-operator-controller-manager` exits 1
  because the Deployment was never created. Same cascade, same fix.
- The restore step, reported but not failed:
  `WARNING: could not restore the dev operator via ...: exit status 201` /
  `stat .../bin/opm: no such file or directory` / `task: Failed to run task "cluster:operator":
  exit status 127`. The clean-tree exit 127 is section 1; the fact that it was a warning rather than
  a failure is section 3. Note the run still exited non-zero here only because the three subtests
  above had already failed — with section 2 landed and section 3 not, a broken restore would leave
  the cluster stripped and the suite green.

  The two subtests after the restore warning, `install immediately after uninstall waits out the
  terminating Deployment` and `crds-only on a fresh cluster installs only the CRDs`, both passed;
  the restore runs last, so nothing in this run observed the stripped cluster.

**Failures blocked on the pinned operator (`proposal.md` § Not in this change)**

Each is `operator did not reconcile generation 2 of default/e2e-operator-owned within 3m`
(`instance_operator_owned_test.go`), which is the `MaterializeFailed` Platform above, not anything
this change touches:

- `TestE2E_ThinEditor_ValuesRoundTrip` (193s)
- `TestE2E_Delete_OperatorOwnedDelegates/without spec.prune the operator orphans the workloads, and
  the CLI says so` (183s)
- `TestE2E_Delete_OperatorOwnedDelegates/with spec.prune the operator removes the workloads` (186s)

These are the failures cli issue 214 tracks. Sections 2 and 3 are measured against this list: the
bar is no failure absent from it.

## Goals / Non-Goals

**Goals**

- A clean checkout plus a running kind cluster is the whole precondition for the cluster-backed
  suite; nothing else has to be started or built by hand.
- Every failure in the loop names its own cause in the run where it happened.
- No new machinery: the change should be net-subtractive.

**Non-Goals**

- Making the suite hermetic in the strong sense (every registry read declared by its test). See
  Decision 1 and Open Questions.
- Any assertion that the restored operator can actually reconcile. See Decision 3 and
  `proposal.md` § Not in this change.
- Introducing a suite-wide registry fixture. The per-test in-process registry already covers every
  writing test, and a shared one would add lifetime and isolation problems these tests do not have.

## Decisions

### 1. Point the stub at `config.DefaultRegistry` by reference, not by copy

The stub config's `registry` key is set from the Go constant `config.DefaultRegistry` via
`fmt.Sprintf`, the pattern `internal/cmd/config/vet_test.go:42` already uses.

**Alternatives considered**

1. *Write the GHCR mapping into the stub as a literal.* Creates a third copy of a string that
   already lives in `internal/config/templates.go` and `hack/opm-config.cue`. A copy is a thing that
   drifts, and the drift would be invisible until a cluster test failed on a registry error again.
2. *Point the stub at a deliberately unroutable address and require every test to declare its
   registry.* Maximum hermeticity: a test that forgets would fail loudly instead of reaching the
   network. But it rewrites every currently-passing cluster test for a property nothing has asked
   for, and it would fail the lifecycle test in a new way rather than fixing it.
3. *Delete the key.* **Does not work, measured 2026-09-19.** `ResolveRegistry`
   (`internal/config/resolver.go:52-75`) is flag > env > config with no default arm: unset in all
   three leaves `cfg.Registry` empty, and `opmodel.dev` never routes to GHCR. `DefaultRegistry` is
   only the literal `opm config init` interpolates into `DefaultConfigTemplate`, and
   `TestLoadConfigFile_DefaultTemplateIsValid` asserts that the *template* carries it — not that an
   absent key resolves to it. With the key deleted, `install` still fails; only the message changes,
   from `registry localhost:5000` to `registry ` (empty).
4. *Give the lifecycle test `--config hack/opm-config.cue`, as `runOperatorOwnedOPM` does.* Fixes
   the one test, but leaves the stub naming no registry at all, so the next test written without an
   explicit one hits the same empty-registry failure this change exists to remove.
5. *Add a default arm to `ResolveRegistry`.* Would make option 3 work and fix it for end users too,
   but it changes shipped behaviour for every `opm` invocation. That is a releasing `fix`, and it
   belongs in its own change.

**Decision**: 1-by-reference. The stub exists to make the suite independent of the developer's real
`~/.opm`, not to override the shipped default; pinning it to a dead address was never the point and
is the whole defect. Interpolating the constant gives the suite exactly the resolution a user gets
from `opm config init` while adding no copy that can drift — the objection that ruled out the
literal form does not reach a Go constant reference.

**Consequence worth stating**: an e2e invocation that names no registry of its own now reaches GHCR
where it previously failed fast. That is already true of every migrated test in the suite, and
`skipWithoutCoreSchema` is the established handling for an unreachable registry.

### 2. `deps: [build]` rather than a precondition or `go run`

**Alternatives considered**

1. *A `preconditions:` entry checking the binary exists, with "run `task build`" as the message.*
   Turns exit 127 into a clear error, which is an improvement, but it does not make the task usable
   by a caller that cannot run commands by hand — and the caller that matters is a test cleanup.
2. *`go run ./cmd/opm` instead of `./bin/opm`.* Needs no binary at all, but drops the version
   ldflags, and `opm operator install` reports the version it installed with.
3. *`deps: [build]`.* Task's dependency runs first; `build`'s `sources`/`generates` checksum makes it
   a no-op when the binary is current.

**Decision**: 3. It is the only option that satisfies the requirement's real content — the task must
work unattended — and it costs nothing on the common path.

### 3. Fail through `t.Errorf`, on the restore command's own outcome

`restoreDevOperator`'s `t.Logf` becomes `t.Errorf`. It runs inside `t.Cleanup`, where `t.Errorf`
still fails the test, so no restructuring is needed.

**Alternatives considered**

1. *`require`/`t.Fatalf`.* `FailNow` from a cleanup goroutine is not the documented contract, and
   there is nothing left to abort — the restore is the last thing the test does.
2. *Also assert the restored operator reaches `Ready=True` on `Platform/cluster`.* This is the check
   that would have caught the real problem: `task cluster:operator` currently prints
   `operator v1.0.0-alpha.14 is reconciling` while the Platform sits at `Ready=False`/`Stalled=True`
   with `MaterializeFailed`, because `status.operatorVersion` — the field `kind-cluster-tasks`
   requires the task to wait for — is stamped on every reconcile regardless of outcome. Adding the
   assertion here would make every e2e run red until the operator pin is bumped, which would leave
   `main` un-releasable and violate Principle VIII.
3. *`t.Errorf` on the command's exit status only.*

**Decision**: 3, with 2 recorded as the follow-up gated on the operator bump. This change makes the
loop *honest about what it checks*; making the check itself stronger is a separate change with a
separate prerequisite, and conflating them would make neither landable.

### 4. Section order is load-bearing

Each section must end green on its own (Principle VIII), which fixes the order:

```
  1. deps: [build]          cluster:operator works from a clean tree.
         |                  Nothing depends on this yet; green on its own.
         v
  2. drop the stub's        the lifecycle test's `operator install` stops
     registry key           failing on registry resolution. Green on its own.
         |
         v
  3. t.Logf -> t.Errorf     the restore now fails the run when it fails.
                            Safe only because 1 made it succeed from a clean
                            tree; landing this first would turn a fresh
                            worktree's exit 127 into a red suite.
```

Sections 1 and 2 are independent of each other. Section 3 depends on 1.

## Risks / Trade-offs

- [A flaky `task cluster:operator` becomes a flaky suite, where before it was a silent warning] →
  The restore runs only after the test's own assertions have passed, so a failure here is a real
  environment failure, not a test-logic flake. The alternative — a silently poisoned cluster — costs
  more, and cost it already: it is what produced the misleading
  `the server could not find the requested resource` diagnosis this change came from.
- [Dropping the stub's registry key makes previously-offline-failing invocations reach the network] →
  Already the case for every migrated test; `skipWithoutCoreSchema` is the established handling and
  needs no change.
- ["Restore succeeded" still does not mean "the operator can reconcile", because of the pinned
  version] → Stated in `proposal.md` § Not in this change and in Decision 3. This change does not
  claim that property; the follow-up gated on the operator bump adds it.
- [None of this is covered by CI, so it can rot again] → Real, and out of scope by decision. The
  change reduces the surface that can rot (one fewer duplicated constant, one fewer manual step) but
  does not close the gap. Giving the PR `e2e` job a cluster is the change that would.

## Migration Plan

Three sections in one PR, each a commit that leaves `main` releasable: `chore(taskfile)` for section
1, `test(e2e)` for sections 2 and 3. No type in that set releases, so no version is cut. Rollback is
a revert of any individual section; they are independent apart from the 1-before-3 ordering. No
stored state, no user-visible surface, no cluster state changes beyond what the tasks already do.

## Open Questions

- Should the suite eventually move to strong hermeticity — every test declaring the registry it
  resolves through, with no default fallback at all (Decision 1, alternative 2)? It is a coherent
  direction and would make a forgotten registry a loud failure rather than a network read. It does
  not change these specs, this approach or this task breakdown: it would be a later change to the
  same requirement, taken deliberately across the whole suite rather than as a side effect of
  repairing one test.
