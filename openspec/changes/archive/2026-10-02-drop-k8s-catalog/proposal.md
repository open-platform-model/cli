## Why

The raw Kubernetes catalog `opmodel.dev/catalogs/k8s@v1` is retired (owner decision, 2026-10-02). catalog_opm deletes it in its change `retire-k8s-catalog` and, in the same change, moves the `opm` module root from `catalog_opm/opm/` to `catalog_opm/src/` (module path unchanged). The cli still names the k8s catalog as a first-party default and subscribes its dev platforms to it.

## What Changes

- `internal/config/templates.go`: `DefaultCatalogPaths` (opm and k8s) collapses to `DefaultCatalogPath`, the one first-party catalog. The only non-test reader, `opm operator install`, already used `DefaultCatalogPaths[0]`.
- `hack/platform/` and `hack/kind-platform.yaml` drop the k8s subscription; `hack/platform/cue.mod/module.cue` drops the dep.
- `internal/config/platform_test.go`: `TestBuildPlatformModule_KeyImportDriftNamesTheEntry` swapped the opm and k8s bindings; it is rewritten against one catalog.
- Tests that used the k8s path only as a sample second catalog switch to `example.com/catalogs/extra@v1`, which sorts before `opmodel.dev/catalogs/opm@v4` as the k8s path did, so every order-sensitive expectation holds.
- `AGENTS.md`, `CONSTITUTION.md` and `openspec/config.yaml` stop listing `opmodel.dev/catalogs/k8s@v1` among the beta lines.

Affected packages: `internal/config`, `internal/platform`, `internal/cmd/platform`, `internal/workflow/render`. No command, flag or output changes.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

None. No requirement changes; `.openspec.yaml` declares `skip_specs: true`.

## Impact

- **SemVer.** None: no user-visible behaviour changes, so after GA this would release nothing; every commit is `refactor`, `test` or `docs`. `opm config init` writes no platform and nothing reads `DefaultCatalogPaths` outside tests.
- **Not in this change.** The move of the catalog checkout path from `../catalog_opm/opm` to `../catalog_opm/src` (four tests including `TestRealTree_CatalogOpm`, which skips silently when the directory is missing, the platform-resolution spec's local-checkout scenario, and `docs/site/extending/publish-a-catalog.md`). It follows catalog_opm's `retire-k8s-catalog` in a later change; before that, `catalog_opm/src` does not exist.
- **Workspace root.** `.tasks/deps/platform-pins.sh` bumps the `opmodel.dev/catalogs/k8s@v1` key in `hack/kind-platform.yaml` and fails when the key is missing. The workspace change that drops it from the script lands before or with this one.
- **Existing platforms.** A user's platform that subscribes to `k8s@v1` keeps resolving its last published build (`1.0.0-beta.2`); the cli does not refuse or warn about it.
