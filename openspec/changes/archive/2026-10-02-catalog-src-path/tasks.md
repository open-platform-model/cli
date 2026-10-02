# Tasks: catalog-src-path

One PR, title `test: read the opm catalog checkout from catalog_opm/src`. No design.md: the change
is a path rename with no decision to record.

## 1. Test and docs paths

- [x] 1.1 Point `internal/publish/realtree_test.go`, the three render tests and `docs/site/extending/publish-a-catalog.md` at `catalog_opm/src`; the skip message names the path. Verify: `go test -v -run TestRealTree_CatalogOpm ./internal/publish/` runs and passes with the workspace checkout present. Commit `test: read the opm catalog checkout from catalog_opm/src` and `docs(extending): point the catalog check paths at catalog_opm/src`.

## 2. Spec path

- [x] 2.1 The delta restates "Module commands render against the module's deps" with every scenario and only the path changed. Verify: `task openspec:check` green. Commit `docs(openspec): name catalog_opm/src in the local checkout scenario`.

## 3. Archive

- [x] 3.1 `openspec archive catalog-src-path --yes` on this branch, so the archive rides the implementing PR. Verify: `task openspec:check` green. Commit `chore(openspec): archive catalog-src-path`.
