## Why

The `internal/` codebase has accumulated dead code — unused packages, orphaned functions, unexported-but-exported symbols, and stale stubs. This increases cognitive load, misleads contributors about API surface, and inflates test/maintenance scope. Cleaning it up now (during active development) prevents further drift and aligns with Principle VII (Simplicity & YAGNI).

## What Changes

- **Delete** `internal/testutil/` — entirely unused package (zero imports)
- **Delete** `internal/identity/` — only imported by one test; UUID constant duplicated as a literal in `build/release_builder.go`
- **Delete** `internal/cmd/mod_stubs.go` — dead stub superseded by real `mod_build.go`
- **Remove** ~15 dead exported functions across `config/`, `output/`, `errors/`, `kubernetes/`, `cmd/` that are never called outside their own package or tests
- **Unexport** ~30+ symbols that are only used within their own package (colors, styles, health types, discovery internals, verbose types)
- **Fix** stale integration test in `tests/integration/deploy/main.go` (outdated `DiscoverResources` call signature)
- **Update** tests that referenced deleted code

## Capabilities

### New Capabilities

_None — this is a pure refactoring/cleanup change._

### Modified Capabilities

_None — no spec-level behavior changes. All removals are internal implementation details that have no effect on observable CLI behavior._

## Impact

- **Packages affected**: `internal/testutil`, `internal/identity`, `internal/cmd`, `internal/config`, `internal/errors`, `internal/output`, `internal/kubernetes`
- **No API/flag/command changes** — this is a PATCH-level change
- **Tests**: Some test files that reference deleted symbols will need updates (remove tests for deleted functions, or inline constants)
- **SemVer**: PATCH — no user-facing behavior changes
