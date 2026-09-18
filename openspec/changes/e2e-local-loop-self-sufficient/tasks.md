# Tasks: e2e-local-loop-self-sufficient

Three sections, in the order `design.md` § Decision 4 fixes: the build dependency before the strict
restore that depends on it, the registry fix independent of both.

A note on the gates. `task test` cannot be fully green while `PinnedOperatorVersion` is
`v1.0.0-alpha.14`: that operator stalls on the cluster `Platform` and the operator-owned e2e tests
need one that reconciles, which is out of this change's scope (`proposal.md` § Not in this change).
Each section's commit task therefore runs `task lint` and `task test:unit` to green, and measures the
cluster-backed suite against the baseline recorded in task 1.1 — the bar is *no failure absent from
that baseline*, not an unconditionally green e2e run.

## 1. cluster:operator builds the CLI it runs (Taskfile.yml)

- [ ] 1.1 Against a prepared `kind-opm-dev` cluster, record the current cluster-backed e2e baseline
      in `design.md` § Context: every failing test, its subtests, and the cause of each. This is the
      reference sections 2 and 3 are measured against, and it must be taken before any edit. Verify:
      `design.md` names each failing test with its cause, and distinguishes failures this change
      addresses from those blocked on the operator pin.
- [ ] 1.2 Add `deps: [build]` to the `cluster:operator` task in `Taskfile.yml` per design.md
      § Decision 2. Verify: with `bin/opm` deleted, `task cluster:operator` against a running cluster
      builds the binary and completes the install; a second run rebuilds nothing and completes
      without duplicating operator arguments (the idempotence `kind-cluster-tasks` already requires).
- [ ] 1.3 `task lint` and `task test:unit` green, then commit
      `chore(taskfile): build the CLI cluster:operator installs with`.

## 2. The e2e suite resolves through the shipped default registry (tests/e2e)

- [ ] 2.1 Remove the `registry` key from the stub `~/.opm/config.cue` that `TestMain` writes in
      `tests/e2e/mod_init_test.go`, leaving the rest of the stub intact, so `config.DefaultRegistry`
      applies (design.md § Decision 1). Verify:
      `go test ./tests/e2e/ -run 'TestE2E_Operator_InstallUninstallLifecycle/install' -v` passes
      against a prepared cluster, where it previously failed with
      `opmodel.dev/catalogs/opm@v4 has no published release`.
- [ ] 2.2 Confirm no test is left depending on a registry it does not start: grep `tests/` for a
      hardcoded local registry address. Verify: no match under `tests/e2e/`, and the only remaining
      matches are the deliberate fixture-seeding paths (`hack/fixtures.sh`, the PR workflow's mixed
      mapping), which are outside this suite and unchanged.
- [ ] 2.3 `task lint` and `task test:unit` green, and `task test:e2e` shows no failure absent from
      the 1.1 baseline, then commit
      `test(e2e): resolve the suite through the shipped default registry`.

## 3. A failed operator restore fails its own run (tests/e2e)

- [ ] 3.1 Change `restoreDevOperator` in `tests/e2e/operator_test.go` to report through `t.Errorf`
      instead of `t.Logf`, keeping the message's underlying error and the by-hand repair command
      (design.md § Decision 3). Verify: with the restore forced to fail (invoke it with a `PATH` that
      omits `task`, or point it at a task name that does not exist), the suite exits non-zero and the
      message names the failed restore, the error and the remedy; with the restore working, the run
      reports nothing from it.
- [ ] 3.2 Cross-cutting: run `task check` in full against a prepared cluster and compare every
      failure against the 1.1 baseline. Verify: no failure is new, and each remaining one is
      attributable to the pinned operator; record that comparison in the PR description.
- [ ] 3.3 `task lint` and `task test:unit` green, then commit
      `test(e2e): fail the run when the dev operator cannot be restored`.
