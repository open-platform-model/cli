# Tasks: fixtures-catalogs

One PR, title `test(fixtures): let the shared fixture flow publish catalog fixtures`. The opm-operator
change `publish-test-catalog` carries the same three files; the two PRs merge in the same sitting.

`task test` includes `task test:integration` and `task test:e2e`, which need a kind cluster the
run owns (the e2e lifecycle test tears the operator down). The change touches no code either tier
exercises beyond the fixture flow that `task test:fixtures` runs, so run `task test:unit`,
`go test ./tests/fixtures/` and `task test:fixtures` against a throwaway registry, and report
integration and e2e as skipped instead of using a shared cluster.

Design.md "Research & Decisions" records the proof; no assumption is unverified, so there is no
spike section.

## 1. Catalog fixtures in the shared flow (hack, tests/fixtures)

- [x] 1.1 Copy `hack/fixtures.sh`, `test/fixtures/fixtures.go` and `test/fixtures/fixtures_test.go` from the opm-operator branch `test/publish-test-catalog` (design.md D1, D2) to `hack/fixtures.sh`, `tests/fixtures/fixtures.go` and `tests/fixtures/fixtures_test.go`. Verify: `cmp` against the operator copies reports no difference; `shellcheck hack/fixtures.sh` is clean.
- [x] 1.2 Verify no regression (design.md "No regression here"): `pins`, `check` against GHCR, `seed` into a throwaway registry and `consumers examples tests/e2e/testdata/operator-owned` under the mixed mapping behave as before; `go test ./tests/fixtures/` passes.
- [x] 1.3 `task fmt`, `task lint`, `task test:unit`, `go test ./tests/fixtures/`, `task test:fixtures` (see the note above) and `task openspec:check` green, then commit `test(fixtures): let the shared fixture flow publish catalog fixtures`.

## 2. Archive

- [x] 2.1 `openspec archive fixtures-catalogs --yes` on this branch, so the archive rides the implementing PR. Verify: `task openspec:check` green and `openspec/specs/test-fixture-lineage/spec.md` carries the ADDED requirement. Commit `chore(openspec): archive fixtures-catalogs`.
