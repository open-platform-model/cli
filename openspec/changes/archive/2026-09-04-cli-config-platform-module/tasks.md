# Tasks: cli-config-platform-module

Gate: implementation of tasks 2 onward needs the D5 core prerelease and both catalogs resolvable beside it (for the pinned template and the vet build tests).

## internal/config

- [x] 1. Paths: replace `PlatformFile` with the platform module directory (`PlatformDir`, sibling `platform/` of the resolved config path); update `PlatformFilePath` callers and keep a helper naming the legacy file path for migration checks.
- [x] 2. Templates: add `DefaultPlatformModuleFile` (`cue.mod/module.cue`: `opmodel.dev/platforms/local@v0`, pins for core + both `DefaultCatalogPaths`) and `DefaultPlatformCUE` (`platform.cue`: embeds `core.#Platform`, imports both catalogs with package names verified against the published artifacts, two `#registry` entries with `#catalog:` imports, maintenance-loop comments); delete `DefaultPlatformTemplate`.
- [x] 3. Platform validation: replace `LoadPlatformFile`/`ValidatePlatformFile`'s file path with a module build through the kernel's shape-gated platform loader; wrap failures in `DetailError` naming the module dir and, where the cause carries it, the dependency or `#registry` entry; trim `schema/platform.cue` to what the CR decode still uses. Unit tests: clean build, nonexistent pin, key/import drift.

## internal/cmd/config

- [x] 4. Init: write the platform module (0700 dir, 0600 files), remove a legacy `~/.opm/platform.cue` with a printed note, keep offline behavior and existing refusal/force semantics; update help text (files list, maintenance sentence). Update `init_test.go`.
- [x] 5. Vet: swap the platform check to the module build; add the missing-module note and the legacy-file failure with the `--force` migration hint; keep `FormatVetCheck` streaming and check ordering. Update `vet_test.go` with the new pass/fail/legacy scenarios.

## Mirrors and docs

- [x] 6. Rewrite `hack/platform.cue` as the `hack/platform/` module (same pins as the template) and update any hack tooling referencing the old path.
- [x] 7. Update `CLAUDE.md`'s pin-mirror note (pins now live in `internal/config/templates.go`'s embedded `cue.mod`, `hack/platform/cue.mod`, `hack/kind-platform.yaml`) and the README/config docs sentences describing `~/.opm/platform.cue`.
- [x] 8. Workspace-root deps tooling: rewrite the cli stanza of `.tasks/deps/platform-pins.sh` to bump the embedded `cue.mod` deps block in `internal/config/templates.go` (the `"<catalog>@vN": v: "vX.Y.Z"` form replaces the deleted scalar `version:` lines; keep the `spec_test.go` literal bump the script's header names); verify `task deps:update:modules`' find actually picks up `cli/hack/platform/cue.mod` so the mirror bumps in the same pass as the seed; update the `deps:update` / `deps:pins:platform-pins` task descriptions and the root `CLAUDE.md` line naming the "cli DefaultPlatformTemplate seed" pin location.

## Verification

- [x] 9. Grep the repo for remaining `platform.cue` data-file assumptions (`PlatformFile`, `#PlatformFile`, "data-only" prose) outside the CR-decode path and the archive; fix each hit or hand it to `cli-render-switch` with a code comment naming that change.
- [x] 10. `task check` green; `task test:unit` covers init/vet paths without a cluster. Run `task deps:update` (dry-eye check: inspect the diff, revert) from the workspace root to prove the retooled pin bump touches the embedded seed, `hack/platform/cue.mod`, and nothing stale.
