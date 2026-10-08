## Why

`task -x deps:cascade:test` with `CASCADE_TEST_SET=all` fails on `main` since 2026-10-05: S2, S4 and S9 print "the setup did not apply". The CI job `Cascade task (network)` is red on `main` and on every pull request that touches the cascade files, so it guards nothing.

The cause is the first step of `setup_older` in `.tasks/cascade/test.sh`: `go get library@<older row> && go mod tidy`. The row is `v1.0.0-beta.2`. On 2026-10-05 cli#322 (library `v1.0.0-beta.6`) started to import `opm/k8s/object`, and cli#328, cli#329 and cli#330 added `labels`, `inventory` and `health`. No published library below `v1.0.0-beta.6` holds all four, so `go mod tidy` fails, and so would the task's own build of `opm` from the merge base. No other published version can replace the row: the tree's library is the newest release.

The setup technique is the defect. It lowers the pin under unchanged source, which holds only while the cli uses no library API newer than the row. Every pull request that bumps the library and adopts its new API breaks it again, and the workflow's path filter does not run for such a pull request.

## What Changes

- `setup_older` lowers the library to a version the test makes from the tree's own library: the same source under a lower version name, served to `go get` from a file proxy in the test's temporary directory. The source always compiles, so the setup no longer depends on which library API the cli uses. The cascade run itself is untouched: it still gets the real library version through the normal Go proxy settings.
- The `older` library row leaves `testdata/older.tsv`; the catalog, core and podinfo rows stay. This departs from section 8 of the shared cascade contract for the library key; design.md records it, and the contract itself is not changed here.
- `setup_older` names the step that failed and shows the last lines of its output in the FAIL line, instead of "the setup did not apply" alone.
- The `older.tsv` check moves to the pre-checks, so the offline set (the required `Lint` job) also runs it. It needs no network.
- No scenario is removed, and no scenario assertion changes. `cascade.sh` does not change.

SemVer: none after GA (test code only, no shipped file). Commits are typed `test`, which release-please hides.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `deps-cascade`: the test requirement gains two statements: the older-pins setup takes its older library from the tree's own library, and the offline set checks the `older.tsv` rows.

## Impact

- `.tasks/cascade/test.sh`, `.tasks/cascade/testdata/older.tsv`.
- The full set now needs `zip` on `PATH` (present on `ubuntu-latest`).
- The setup writes one made-up library version (`v0.0.0-0.cascade.<tree version>`) into the Go module cache. Its name holds the tree version, so its content never changes under one name.
- No workflow, Taskfile, Go code or pin changes.
