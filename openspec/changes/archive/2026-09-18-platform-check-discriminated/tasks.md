# Tasks: platform-check-discriminated

Three sections per design.md. Section 1 is the spike (design.md § Where the pins move): it lands the pins and proves the suite is green on library alpha.32 and core alpha.10 before the report changes.

## 1. Pins: library alpha.32, core alpha.10 (go.mod, hack/platform, internal/config, templates)

- [x] 1.1 From the workspace root run `task deps:update`; keep only this repo's rewritten files (`hack/platform/cue.mod/module.cue`, `internal/config/templates.go` `DefaultCorePin`/`DefaultCatalogPins`, `templates/*/cue.mod/module.cue`) and leave every sibling repo's rewritten tree for its own `fix(deps)` PR (or reset it with `git checkout` there). Then `go get github.com/open-platform-model/library@v1.0.0-alpha.32 && go mod tidy`. Verify: `DefaultCorePin` reads `v2.0.0-alpha.10`, `hack/platform` pins the same, `go.mod` names alpha.32, and `go build ./...` passes.
- [x] 1.2 Spike: `go test ./internal/cmd/platform/... ./internal/platform/... ./internal/config/...` against the registry. Record in design.md § Context which fixtures now decode `comparable` (expected: all of them, with `catalogFulfilledPluralityPlatform` carrying one row and every test still green because nothing reads the row yet). Verify: the finding is written; any test that fails for a reason other than registry reachability is fixed or explained before the commit.
- [x] 1.3 `task fmt lint test:unit` green, then commit `fix(deps): bump core to v2.0.0-alpha.10, the catalogs, and library to v1.0.0-alpha.32`.

## 2. The report (internal/platform)

- [x] 2.1 In `internal/platform/check.go`, add `Comparable []libplatform.ComparablePredicates` and the unexported `discriminated` to `Report`, fill both in `NewReport`, add `Discriminated()`, and correct `Routable()`'s doc to "one of the two values that decide the exit status". Verify: `go build ./internal/...` passes and nothing outside the package constructs a `Report` field-by-field.
- [x] 2.2 Extend `Render()` per design.md § Rendering: the comparable section after the over-subscribed one (rows sorted by broader then narrower, shared contracts sorted, the D5 prose), `verdictLine` taking the noun so the third line reads `discriminated: no — N pair(s) is/are comparable`, the two existing verdict lines byte-identical, and the empty-inventory branch gaining the vacuous `discriminated` line. Verify: table tests in `check_test.go` cover comparable-only, over-subscribed plus comparable, and empty; the existing clean, unfulfilled-only and over-subscribed cases pass unchanged; `TestReportRenderIsDeterministic` covers two row orderings.
- [x] 2.3 Add the verdict test beside `TestRoutableIsTheOnlyVerdict`: an undiscriminated report is routable and not discriminated, an over-subscribed one is discriminated and not routable, and `fulfilled` decides neither; name enhancement 0015 D5 and D18 in the comments. Verify: `go test ./internal/platform/...` passes.
- [x] 2.4 `task fmt lint test:unit` green, then commit `feat(platform): report comparable transformer pairs in the check report`.

## 3. The command (internal/cmd/platform)

- [x] 3.1 In `internal/cmd/platform/check.go`, refuse when `!report.Routable() || !report.Discriminated()` with the validation `ExitError` (`Printed: true`) and the message from design.md § Exit code naming both counts; extend the help text's exit-code list with `comparable`. Verify: `opm platform --help` and `opm platform check --help` render the new line.
- [x] 3.2 Split the fixture per design.md § The plurality fixture is split: `catalogFulfilledPluralityPlatform`'s mirror gains a required label, and the label-less mirror becomes `undiscriminatedPlatform` on top of `routablePlatform`. Verify: `TestPlatformCheck_CatalogFulfilledPluralityIsNotOverSubscription` passes with `discriminated: yes` asserted in the report.
- [x] 3.3 New command tests: `undiscriminatedPlatform` exits with the validation code, the report names mirror as broader, schedule as narrower and the container contract as shared, and the error names both counts; a platform that is both over-subscribed and undiscriminated exits once with both headings printed; a platform pinning core `v2.0.0-alpha.9` fails naming `comparable` and `2.0.0-alpha.10` with no report printed (beside the existing alpha.6 case). Verify: `go test ./internal/cmd/platform/...` passes against the registry and skips without it, as the file's neighbours do.
- [x] 3.4 Cross-cutting: `task check` in full (including `openspec:check`), and confirm no other test asserted the two-verdict report shape. Verify: green.
- [x] 3.5 `task check` green, then commit `feat(cmd): fail opm platform check on comparable transformer predicates`.
