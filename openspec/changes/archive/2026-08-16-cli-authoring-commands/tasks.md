# Tasks — cli-authoring-commands

> Slim change: writer gaps, then commands, then record. Init work lives in `cli-template-modules`. Bare-`@` ban on every commit.

## 1. cueedit

- [x] 1.1 `SetIdentityVersion` no-op detection (spliced == existing → no write) returning `Changed`; the one `internal/publish` caller updated; test: unchanged value → identical bytes and mtime.
- [x] 1.2 `ReadIdentityVersion(dir) (value, defaulted, err)`; tests over literal/defaulted/open/absent shapes.
- [x] 1.3 `SetCueModModule(dir, path)` splice-writer for the `module:` line; tests incl. comment preservation (consumer lands in `cli-template-modules`).

## 2. version set

- [x] 2.1 Shared `runVersionSet` in `cmdutil`: read → no-op fast path ("already X") → set → report old → new; structural refusal via `ErrIdentityShape` pointing at `opm module vet`; exit 0/0/2.
- [x] 2.2 `opm module version set` + `opm catalog version set` cobra wiring (`version` subgroups); constructor tests.
- [x] 2.3 Real-repo fixtures per the graduation gate: copies of the shipped catalog and module identity files; idempotency, assertion preservation, and release-please-marker preservation asserted on both.

## 3. Specs, gates, record

- [x] 3.1 New spec `authoring-commands` (version-set contract).
- [x] 3.2 `CLAUDE.md` command map; `task fmt vet lint test` green.
- [x] 3.3 Record in `enhancements/0011/`: slice `cli-authoring-commands` → `in-progress` note that it lands as two changes (this + `cli-template-modules`); flip to `done` with both `openspec_ref`s when the second lands.
