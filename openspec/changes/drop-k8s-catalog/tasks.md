## 1. internal/config and hack: one first-party catalog

- [ ] 1.1 `internal/config/templates.go`: replace `DefaultCatalogPaths` and `DefaultCatalogPath` with the `DefaultCatalogPath` const (design D1); `internal/platform/catalog.go` keeps re-exporting it
- [ ] 1.2 `hack/platform/platform.cue` and `hack/platform/cue.mod/module.cue`: drop the k8s import, `#registry` entry and dep; `cue mod tidy` in `hack/platform`
- [ ] 1.3 `hack/kind-platform.yaml`: drop the `opmodel.dev/catalogs/k8s@v1` entry
- [ ] 1.4 `internal/config/platform_test.go`: lines 59 and 143 check `DefaultCatalogPath`; rewrite `TestBuildPlatformModule_KeyImportDriftNamesTheEntry` per design D2 and record the measured re-keying (or the deletion and core's covering test) in design.md
- [ ] 1.5 `task test:e2e` against the kind cluster with the edited platform
- [ ] 1.6 `task lint` and `task test` green, then commit `refactor(config): drop the k8s catalog from the first-party defaults`

## 2. Test samples: a neutral second catalog

- [ ] 2.1 Swap `opmodel.dev/catalogs/k8s@v1` for `example.com/catalogs/extra@v1` (design D3) in `internal/cmd/platform/pull_test.go`, `internal/platform/generate_test.go`, `internal/platform/resolve_test.go`, `internal/platform/moduledeps_test.go`, `internal/platform/spec_test.go`, `internal/workflow/render/validation_test.go`
- [ ] 2.2 `internal/workflow/render/replacements_test.go`: the k8s entry becomes `example.com/catalogs/extra@v1` with `replaceWith: "../catalog_extra"`
- [ ] 2.3 `grep -rn 'catalogs/k8s' --include=*.go .` returns nothing
- [ ] 2.4 `task lint` and `task test` green, then commit `test: use a neutral second catalog in samples`

## 3. Rule files

- [ ] 3.1 `AGENTS.md` (Beta promise bullet), `CONSTITUTION.md` and `openspec/config.yaml`: the beta-line list drops `opmodel.dev/catalogs/k8s@v1`
- [ ] 3.2 `grep -rn -E 'catalogs/k8s|k8s catalog|raw (kubernetes )?catalog' --exclude-dir=archive --exclude-dir=.git --exclude=CHANGELOG.md .` returns only this change's own files
- [ ] 3.3 `task lint`, `task test` and `task openspec:check` green, then commit `docs: drop the k8s catalog from the beta lines`
