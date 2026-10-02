## Why

catalog_opm PR 120 moved the opm catalog module root from `catalog_opm/opm/` to `catalog_opm/src/`.
The CUE module path `opmodel.dev/catalogs/opm@v4` and every import path are unchanged; only the
checkout directory moved. The `platform-resolution` scenario "A local catalog checkout is rendered"
still names the old directory in its `replaceWith` example, and the real-tree publish test read the
old path and skipped silently.

## What Changes

- `platform-resolution`: the scenario's example `replaceWith` becomes `../catalog_opm/src`. No
  behavior changes; the requirement is restated in full with only that path edited.
- `internal/publish/realtree_test.go`, three render tests and `docs/site/extending/publish-a-catalog.md`
  read the new directory. The real-tree test's skip message now names the path it looked for.

SemVer class: none. Nothing in the `opm` binary changes.

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `platform-resolution`: the local-checkout scenario names `../catalog_opm/src`.

## Impact

- Files: the five listed above and `openspec/specs/platform-resolution/spec.md`.
