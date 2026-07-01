## 1. Go module and source rewrite

- [x] 1.1 Update `go.mod`'s `module` line: `github.com/opmodel/cli` → `github.com/open-platform-model/cli`
- [x] 1.2 Rewrite every internal `github.com/opmodel/cli/...` import path across the codebase (~114 `.go` files) to `github.com/open-platform-model/cli/...` in a single pass — this cannot be split across commits, a half-renamed Go-source tree does not compile
- [x] 1.3 Run `task fmt` (`goimports -w .`) to re-sort import blocks disturbed by the path substitution
- [x] 1.4 Run `task build` and `task test:unit` to confirm the rewrite compiles and unit tests pass

## 2. Build tooling (ldflags)

- [x] 2.1 Update `Taskfile.yml`'s three `-X` ldflags (`Version`, `GitCommit`, `BuildDate`) to the new import path
- [x] 2.2 Update `.goreleaser.yml`'s three `-X` ldflags to the new import path
- [x] 2.3 Build locally (`task build`) and run `./bin/opm version` — confirm the version string is populated, not empty, proving the ldflags path still resolves

## 3. Documentation

- [x] 3.1 Update `CLAUDE.md`'s `oerrors "github.com/opmodel/cli/pkg/errors"` alias reference to the new path
- [x] 3.2 Fix `CLAUDE.md`'s Environment Notes go-version note (`1.25.0`) to match `go.mod`'s actual `go 1.26.0` directive

## 4. OpenSpec spec sync

- [x] 4.1 Sync the 9 spec deltas in this change (`errors-domain`, `pkg-types`, `core`, `render-pipeline`, `refactoring-requirements`, `core-component`, `core-module`, `core-provider`, `core-transformer`) into their `openspec/specs/` main spec files — text-only, import-path references, no requirement-behavior change

## 5. Verification

- [x] 5.1 Repo-wide grep confirms zero `opmodel/cli` references remain outside `openspec/changes/archive/`
- [x] 5.2 `task lint` passes
- [x] 5.3 `task test` passes (full suite: unit, integration, e2e)
