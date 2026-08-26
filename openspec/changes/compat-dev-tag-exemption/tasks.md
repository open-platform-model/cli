## 1. Classifier and window (`internal/publish`)

- [ ] 1.1 Add `isDevTag` to `compat.go` with a table test covering `-0.dev.N.gsha`, `-dev.1`, `-alpha.1`, `-beta.2`, `-rc.1`, and a stable tag
- [ ] 1.2 Filter dev tags in `predecessorVersions`; extend `TestPredecessorVersions_WindowAndOrder` with dev tags interleaved with alpha/beta tags and assert the measured ordering (`-0.dev.*` below `-alpha.N`)

## 2. Gate outcome and rendering

- [ ] 2.1 Add `CompatDevExempt` to `CatalogGateOutcomes`; set it in `gateCompat` before the scan when `isDevTag(p.Tag)`, mark `CompatChecked`, return
- [ ] 2.2 Render `dev-exempt` on the `compat gate` row in `plan.go` and on the `compat` row in `check.go`; keep every other build's wording unchanged
- [ ] 2.3 Apply the same skip in `RegistryCheck` (`check.go`) for a dev-tagged fetched build

## 3. Hermetic tests

- [ ] 3.1 `TestGateCompat_DevBuildNotJudged`: publish release `1.0.0`, run a dev-tagged tree that removes a beta field, assert `Go()` and `dev-exempt` in the plan
- [ ] 3.2 `TestGateCompat_DevBuildNeverBaseline`: publish `1.0.0` then a dev tag carrying an extra field, run `1.1.0` without that field, assert pass and no dev coordinate in the output; add a member present only in the dev tag and assert it counts as new
- [ ] 3.3 `check_test.go`: `--compat` on a dev build reports `dev-exempt`; on a release build with newer dev history compares against the release
- [ ] 3.4 Confirm every pre-existing `TestGateCompat_*` case passes unmodified

## 4. Validation gates

- [ ] 4.1 `task fmt`
- [ ] 4.2 `task lint`
- [ ] 4.3 `task test`
