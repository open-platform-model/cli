## Why

The raw Kubernetes catalog `opmodel.dev/catalogs/k8s@v1` is retired. opm-operator's registry-backed
platform specs used it as a second resolvable catalog and now need a test catalog of their own
(opm-operator change `publish-test-catalog`). That catalog publishes through `hack/fixtures.sh`, which
is a byte-identical copy in both repos (workspace `task fixtures:lint`), so the script learns
catalog fixtures here too. The cli has no catalog fixture and gains none.

## What Changes

- **`hack/fixtures.sh`** (byte-identical with opm-operator): a catalog fixture root, `CATALOGS_DIR`,
  defaulting to the `catalogs/` sibling of `FIXTURES_DIR` when that directory exists. Fixtures under
  it run `opm catalog publish` and `opm catalog version set`, the rest `opm module ...`; `pins`,
  `check`, `seed` and `publish` iterate catalogs first, then modules; `consumers` does not count a
  cue.mod under the catalog root as an unlisted consumer. Progress lines name the kind.
- **`tests/fixtures/fixtures.go`** (byte-identical): `CatalogDir`, `LoadCatalog` and `MustCatalog`,
  the Go spelling of the same root. **`tests/fixtures/fixtures_test.go`** (byte-identical): every
  catalog fixture loads and sits under `testing.opmodel.dev/catalogs/` (skipped when the root is
  absent, as it is here), and the identity-literal check covers catalogs.

In this repo nothing observable changes: `tests/fixtures/catalogs` does not exist, so `CATALOGS_DIR`
stays empty and every subcommand prints and does what it did, apart from `(module)` in its progress
lines.

SemVer class: none. Every commit is `test` or `chore`; nothing in the `opm` binary changes.

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `test-fixture-lineage`: the shared fixture flow publishes catalog fixtures from a sibling root
  through the catalog pipeline.

## Impact

- Files: `hack/fixtures.sh`, `tests/fixtures/fixtures.go`, `tests/fixtures/fixtures_test.go`,
  `openspec/specs/test-fixture-lineage/spec.md` (on archive).
- Cross-repo: merge together with the opm-operator `publish-test-catalog` PR, which carries the same
  three files; `task fixtures:lint` is a local task, so the window between the two merges breaks no CI.
- `publish-fixtures.yml` runs on the merge (it triggers on `hack/fixtures.sh`) and republishes
  nothing: the podinfo fixture is unchanged.
