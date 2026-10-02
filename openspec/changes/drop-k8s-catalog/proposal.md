## Why

The raw Kubernetes catalog `opmodel.dev/catalogs/k8s@v1` is retired (owner decision, 2026-10-02). catalog_opm deletes it in its change `retire-k8s-catalog` and, in the same change, moves the `opm` module root from `catalog_opm/opm/` to `catalog_opm/src/` (module path unchanged). The cli still names the k8s catalog as a first-party default, subscribes the dev platforms to it, and reads the catalog checkout at `../catalog_opm/opm` in four tests, one of which (`TestRealTree_CatalogOpm`) skips silently when that directory is missing, so after the move it would stop testing without anyone noticing.

## What Changes

- `internal/config/templates.go`: `DefaultCatalogPaths` (opm and k8s) collapses to `DefaultCatalogPath`, the one first-party catalog. The only non-test reader, `opm operator install`, already used `DefaultCatalogPaths[0]`.
- `hack/platform/` and `hack/kind-platform.yaml` drop the k8s subscription; `hack/platform/cue.mod/module.cue` drops the dep.
- `internal/config/platform_test.go`: `TestBuildPlatformModule_KeyImportDriftNamesTheEntry` swapped the opm and k8s bindings; it is rewritten against one catalog.
- Tests that used the k8s path only as a sample second catalog switch to `example.com/catalogs/extra@v1`, which sorts before `opmodel.dev/catalogs/opm@v4` as the k8s path did, so every order-sensitive expectation holds.
- Tests and docs that read `../catalog_opm/opm` read `../catalog_opm/src`; the platform-resolution spec's local-checkout scenario names `../catalog_opm/src`.
- `AGENTS.md`, `CONSTITUTION.md` and `openspec/config.yaml` stop listing `opmodel.dev/catalogs/k8s@v1` among the beta lines; `docs/site/extending/publish-a-catalog.md` names `catalog_opm/src/`.

Affected packages: `internal/config`, `internal/platform`, `internal/cmd/platform`, `internal/workflow/render`, `internal/publish`. No command, flag or output changes.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `platform-resolution`: the local-checkout scenario's example path moves from `../catalog_opm/opm` to `../catalog_opm/src`. No behaviour changes.

## Impact

- **SemVer.** None: no user-visible behaviour changes, so after GA this would release nothing; every commit is `refactor`, `test` or `docs`. `opm config init` writes no platform and nothing reads `DefaultCatalogPaths` outside tests.
- **Order.** Sections 1, 2 and 4 can land any time. Section 3 (the `src` path) lands after catalog_opm's `retire-k8s-catalog` merges: before it, `../catalog_opm/src` does not exist and the local-checkout tests skip or fail.
- **Workspace root.** `.tasks/deps/platform-pins.sh` bumps the `opmodel.dev/catalogs/k8s@v1` key in `hack/kind-platform.yaml` and fails when the key is missing. The workspace change that drops it from the script lands before or with this one.
- **Existing platforms.** A user's platform that subscribes to `k8s@v1` keeps resolving its last published build (`1.0.0-beta.2`); the cli does not refuse or warn about it.
