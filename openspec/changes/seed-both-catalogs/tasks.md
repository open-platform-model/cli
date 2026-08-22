# Tasks: seed-both-catalogs

Ordered so each group is green on its own: the constant first, then the template it feeds, then the
assertions, then the mirror peers that must not drift.

**Blocked until the first release of `opmodel.dev/catalogs/k8s@v1` is published to GHCR.** Group 1
cannot pick a pin before a version exists, and seeding an unpublished pin breaks the template's
never-dangles invariant.

## 1. Catalog path constants

- [ ] 1.1 In `internal/config`, turn the single `DefaultCatalogPath` into a pair that names both first-party catalogs, keeping the property that each path is spelled exactly once. Update the doc comment: it currently calls the abstraction catalog "the single first-party OPM catalog", which is no longer true.
- [ ] 1.2 Update the re-export in `internal/platform/catalog.go` to match, keeping it a re-export rather than a second source of truth.
- [ ] 1.3 `go build ./...` passes and every existing caller of the old constant compiles against the new shape.

## 2. Seeded platform template

- [ ] 2.1 Read the published version of `opmodel.dev/catalogs/k8s@v1` from GHCR and use it as the pin. Do not invent one.
- [ ] 2.2 Extend `DefaultPlatformTemplate` in `internal/config/templates.go` so the rendered `registry` block carries both subscriptions, each with an explicit concrete `version`, interpolated from the constants in 1.1 rather than written as literals.
- [ ] 2.3 Extend the template's own comment: it explains why the pin is hand-bumped and offline; it now explains it for two pins.
- [ ] 2.4 Rendered output is still valid data-only CUE with no imports, and still decodes through the embedded projection schema.

## 3. Assertions

- [ ] 3.1 Update `internal/cmd/config/init_test.go`: it asserts the seeded file has exactly one registry entry and no second catalog path. It now asserts two entries keyed by both catalog paths, each with a concrete version.
- [ ] 3.2 Keep the assertion that no `opmodel.dev/catalogs/kubernetes` entry appears — that path names a retired module, not the extracted catalog, and the distinction is easy to lose.
- [ ] 3.3 Add a case covering the `config-commands` scenario "Seeded platform offers the raw escape hatch": a demand on a raw contract is matchable from the seeded default with no user edit.
- [ ] 3.4 `task test` passes.

## 4. Mirror peers

- [ ] 4.1 Add the second subscription to `hack/platform.cue` with the same pin.
- [ ] 4.2 Add the second subscription to the operator's sample Platform with the same pin.
- [ ] 4.3 Update the mirror-contract note in `CLAUDE.md`: it names three files that must be bumped in one commit, and the set is now four pins across those files.
- [ ] 4.4 `task fmt`, `task lint`, `task test` all pass.
