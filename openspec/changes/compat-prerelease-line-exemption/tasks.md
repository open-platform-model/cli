## 1. Classifier and partition (`internal/publish`)

- [ ] 1.1 Add `isReleasePrerelease` next to `isDevTag` with a table test (`-alpha.1`, `-beta.2`, `-rc.1` true; `-0.dev.N.gsha`, stable false)
- [ ] 1.2 Add `CompatPrerelease` to `CatalogGateOutcomes`; thread `lineIsPrerelease` into `eligibleByPackage` and count beta/GA members into it instead of the walk set
- [ ] 1.3 Compute the flag once in `gateCompat` and in `RegistryCheck` and pass it through `compatScan`

## 2. Rendering

- [ ] 2.1 Extend the `compat gate` row in `plan.go` and the `compat` row in `check.go` with `%d prerelease-exempt`, always printed
- [ ] 2.2 Update any golden or `assert.Contains` strings in `gates_test.go` / `check_test.go` that match the old row shape

## 3. Hermetic tests

- [ ] 3.1 `TestGateCompat_PrereleaseLineExempt`: publish `1.0.0-alpha.1`, run `1.0.0-alpha.2` with a narrowed beta field, assert `Go()`, `0 compared`, `1 prerelease-exempt`
- [ ] 3.2 `TestGateCompat_FirstStableArmsGate`: same history, run `1.0.0`, assert refusal naming `1.0.0-alpha.2` as the predecessor
- [ ] 3.3 `realtree_test.go`: on a stable effective tag the real catalog tree compares clean against its published predecessor (depends on the library change being pinned in `go.mod`)
- [ ] 3.4 `check_test.go`: `--compat` on a prerelease build reports `prerelease-exempt`; on a stable build lists the violation and exits 2
- [ ] 3.5 Confirm every pre-existing `TestGateCompat_*` case passes unmodified

## 4. Validation gates

- [ ] 4.1 `task fmt`
- [ ] 4.2 `task lint`
- [ ] 4.3 `task test`
