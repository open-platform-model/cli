## 1. Pass --skip-unprovided to the kernel

- [x] 1.1 Raise the library to the head of `feat/render-skips-unprovided-provider-demands` as a pseudo-version (`go get github.com/open-platform-model/library@<sha>`, sha from B's report), then `go mod tidy`; verify `go build ./...` and that `kernel.RenderInput.SkipUnprovided`, `kernel.SkippedDemand` and `UnresolvedDemand.Unprovided` resolve
  - Note: the library was already released when this section ran, so 1.1 pinned `v1.0.0-alpha.34` directly (no pseudo-version); section 5 has nothing left to change.
- [x] 1.2 `internal/cmdutil/flags.go`: add `SkipUnprovided` and `--skip-unprovided` (default false, help text from design.md) to `RenderFlags` and `InstanceFileFlags`; verify with a flags test and `opm instance build --help` / `opm module build --help` listing the flag
- [x] 1.3 `internal/workflow/render`: carry the flag through the instance and module render opts into `newRenderInput`, and add `Skipped []kernel.SkippedDemand` to `Result`; verify with a unit test that the kernel input carries the switch for both entry points and that a render without the flag is byte-identical to before
- [x] 1.4 `task lint` and `task test` green, then commit `feat(render): pass --skip-unprovided to the kernel`

## 2. Report skips and name the flag in refusals

- [ ] 2.1 `internal/workflow/render`: add `formatSkipped` and print its lines with `output.Warn` after a successful render; unit tests cover a skipped trait, an omitted component with two skipped resources (one line naming both), and a row with alternatives
- [ ] 2.2 `internal/workflow/render/validation.go`: add the unprovided branch to `refusalHint`, ahead of the deps-source hints and independent of the source; unit tests cover the deps, cluster and `--platform` sources, and a refusal with no unprovided row keeping today's hint
- [ ] 2.3 Add a test module under `internal/workflow/render/testdata` whose component attaches the `backup` trait from `opmodel.dev/catalogs/opm@v4` (first published in `v4.2.0`; pin the version `hack/platform/` uses) when a `backup` value is true, its default; e2e: `opm module build` refuses with exit 2 and the unprovided hint, `opm module build --skip-unprovided` exits 0 with the component's objects on stdout and the skip warning on stderr only; verify `task test:e2e`
- [ ] 2.4 `task lint` and `task test` green, then commit `feat(render): report skipped provider demands and name --skip-unprovided in refusals`

## 3. Record skips on apply

- [ ] 3.1 `internal/inventory/store.go`: add `AnnotationSkippedContracts` and `SpecInput.SkippedContracts`; `ApplySpec` sets one annotations map holding the source and skipped annotations; unit tests cover both annotations together, sorting and deduplication, and omission when empty
- [ ] 3.2 `internal/workflow/apply`: fill `SkippedContracts` from `Result.Skipped` as `<component>=<fqn>`; refuse `--skip-unprovided` with exit 2 before `executeThinEditor` writes anything for an operator-managed instance; unit tests for both
- [ ] 3.3 Integration on `kind-opm-dev`: `opm instance apply --skip-unprovided` of an instance of the section-2 test module stamps the annotation; a re-apply of the same instance with the `backup` value false clears it; verify `task test:integration`
- [ ] 3.4 `task lint` and `task test` green, then commit `feat(apply): record skipped provider contracts on the ModuleInstance`

## 4. Document the flag

- [ ] 4.1 `AGENTS.md` (render-path note), `QUICKSTART.md` and `docs/site/`: describe `--skip-unprovided`, what a skip does to a trait and to a resource, and the annotation; verify every page that describes an unresolved-demand refusal mentions the flag
- [ ] 4.2 `task lint`, `task test` and `task openspec:check` green, then commit `docs: describe --skip-unprovided`

## 5. Pin the released library

- [ ] 5.1 Once the supervisor reports the library release carrying the interface, replace the pseudo-version with it (`go get github.com/open-platform-model/library@<version>`, `go mod tidy`); verify `go.mod` carries no pseudo-version for the library and the section 1-3 tests pass unchanged
- [ ] 5.2 `task lint` and `task test` green, then commit `fix(deps): raise the library to <version> for SkipUnprovided`
