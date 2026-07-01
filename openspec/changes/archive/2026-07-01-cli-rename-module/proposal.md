## Why

Enhancement [0006](../../../../enhancements/0006/) (D15, `enhancements/0006/03-decisions.md`) requires the CLI's Go module path to align with the org namespace (`github.com/open-platform-model/*`) before any slice adds the `library` edge (D9/D13). `library` and `opm-operator` already live at `github.com/open-platform-model/*`; the CLI alone sits at `github.com/opmodel/cli`. Renaming now — before B2 (`cli-operator-install-command`), C1 (`cli-cr-inventory-backend`), and C2 (`cli-kernel-adoption`) start writing new cross-repo imports — means those imports are written once, under the final name. Renaming after would mean a second break, on top of whatever those slices already touched. The CLI has a single user and owes no backwards-compatibility (D14), so the import-path break itself costs nothing; the only cost worth managing is doing the rename completely.

## What Changes

- Rename the Go module: `go.mod`'s `module` line, `github.com/opmodel/cli` → `github.com/open-platform-model/cli`. **BREAKING** (import path change) — accepted under D14, no external consumers.
- Rewrite every internal `github.com/opmodel/cli/...` import statement across the codebase (~114 `.go` files) to the new path. Compiler-checked; low risk on its own.
- Update the two build-tooling files that embed the import path as **ldflags string literals**, not import statements: `Taskfile.yml` and `.goreleaser.yml` (`-X github.com/opmodel/cli/internal/version.Version=...` and the matching `GitCommit`/`BuildDate` flags). These are invisible to `go build`/`go vet` — a `.go`-only sweep would compile and test clean while silently breaking `opm version`'s output. Verified as a separate, explicit step from the Go-file rewrite.
- Update `CLAUDE.md`'s `oerrors "github.com/opmodel/cli/pkg/errors"` alias reference.
- Update the 9 active OpenSpec main specs whose `WHEN`/`THEN` requirement text names the old import path: `errors-domain`, `pkg-types`, `core`, `render-pipeline`, `refactoring-requirements`, `core-component`, `core-module`, `core-provider`, `core-transformer`. Text-only — the underlying requirement (e.g. "alias the errors package as `oerrors`") is unchanged, only the import-path string it names. Archived changes under `openspec/changes/archive/` are left untouched — historical record, not living spec.
- Fix the adjacent Go-version drift while `go.mod` is already being touched: `go.mod` declares `go 1.26.0`, `CLAUDE.md`'s Environment Notes says `1.25.0`. Bring the doc in line with the actual `go.mod` value.
- Add a verification step: a repo-wide grep confirming zero remaining `opmodel/cli` references outside `openspec/changes/archive/`, run before the change is considered complete.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `errors-domain`: requirement text updates the `oerrors` import-path example to the new module path. No behavioral change.
- `pkg-types`: requirement text updates three import-path references (`pkg/core`, `pkg/module`, `pkg/bundle`) to the new module path. No behavioral change.
- `core`: requirement text updates the `pkg/core` import-path reference to the new module path. No behavioral change.
- `render-pipeline`: requirement text updates its import-path reference to the new module path. No behavioral change.
- `refactoring-requirements`: requirement text updates its import-path reference to the new module path. No behavioral change.
- `core-component`: requirement text updates its import-path reference to the new module path. No behavioral change.
- `core-module`: requirement text updates its import-path reference to the new module path. No behavioral change.
- `core-provider`: requirement text updates its import-path reference to the new module path. No behavioral change.
- `core-transformer`: requirement text updates three import-path references (`internal/core/transformer` ×2, `internal/transformer`) to the new module path. No behavioral change.

## Impact

- **Code**: every package under `cmd/`, `internal/`, `pkg/` — import statements only, no logic changes.
- **Build tooling**: `Taskfile.yml`, `.goreleaser.yml` — ldflags string literals for version injection.
- **Docs**: `CLAUDE.md` (one reference), 9 `openspec/specs/*/spec.md` files (text-only).
- **Downstream**: unblocks enhancement 0006 slices B2 → C1 → C2, none of which may add the `library`/CRD cross-repo imports until this lands (D15's stated ordering rationale).
- **External consumers**: none. Verified `opm-operator/go.sum` and `library/go.sum` do not import `opmodel/cli`; no `.github/` CI workflows reference the path.
- **Existing in-progress change**: `openspec/changes/cue-binary-integration/` (0/72 tasks, stale since 2026-02-15) is out of scope for this change and is left untouched.
