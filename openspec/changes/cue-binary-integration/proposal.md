## Why

The CLI needs to invoke CUE module operations (`cue mod tidy`, potentially `cue mod get` in the future) that have no public Go API — `modload.Tidy()` lives behind Go's `internal` package boundary. Shelling out to the `cue` binary is the pragmatic path, but we need reliable binary discovery, execution, and version compatibility checks to avoid silent mismatches between the CUE SDK compiled into `opm` (v0.15.4) and the user's installed `cue` binary.

## What Changes

- Add a new internal package for CUE binary discovery and execution (`internal/cue/`)
- Add CUE binary version compatibility checking — warn when SDK and binary major.minor diverge
- Extend `opm version` to display the CUE binary version (or "not found") alongside the existing CUE SDK version
- Use the new execution capability in `opm config init` to run `cue mod tidy` after writing config files, resolving the deps reset issue

### Behavior when `cue` binary is not found

Commands gracefully degrade:

- `opm config init`: Succeeds but warns that `cue mod tidy` was skipped. Prints the existing yellow notice directing users to run it manually.
- `opm version`: Shows "CUE binary: not found on PATH" instead of version number.
- Future commands requiring CUE: Fail with actionable error: "cue binary not found on PATH. Install from <https://cuelang.org/docs/install/>"

### Version compatibility checking

When the `cue` binary is found and invoked:

- Compare SDK major.minor against binary major.minor
- **Warn** (yellow, stderr) if major or minor version differs: "CUE SDK (v0.15.4) and binary (v0.16.2) versions differ. Unexpected behavior may occur."
- **Do not block** execution — version drift may be safe for simple operations
- Display check result in `opm version` output

## Capabilities

### New Capabilities

- `cue-binary-exec`: Finding the `cue` binary on PATH, executing CUE commands (starting with `cue mod tidy`), capturing output and errors. Includes version detection and compatibility checking. Foundation for future CUE CLI operations (`cue mod get`, `cue export`, etc.).
- `cue-version-check`: Comparing CUE SDK version (embedded at build time) against the installed `cue` binary version. Warns on major.minor mismatch. Surfaces binary version in `opm version` output.

### Modified Capabilities

- `config-commands`: `opm config init` attempts to run `cue mod tidy` after writing `cue.mod/module.cue`. Succeeds with warning if `cue` binary is not found (preserves existing behavior with improved visibility via version check).
- `core`: `opm version` command displays CUE binary version and compatibility status alongside existing SDK version.

## Impact

- **SemVer**: **MINOR** — new capabilities, extended command output, no breaking changes to existing behavior
- **Affected commands**:
  - `opm version` (extended output)
  - `opm config init` (optional `cue mod tidy` invocation)
- **New package**: `internal/cue/`
  - `FindBinary() (string, error)` — locate `cue` on PATH via `exec.LookPath`
  - `GetVersion(binPath string) (semver, error)` — invoke `cue version` and parse output
  - `CheckCompatibility(sdkVersion, binVersion string) CompatibilityStatus` — compare major.minor
  - `Run(ctx, args ...string) (stdout, stderr, error)` — execute `cue <args>` with current dir and env
- **Modified packages**:
  - `internal/version/` — add `CUEBinaryVersion` field to `Info` struct, add `CUEBinaryPath` field
  - `internal/cmd/` — `version.go` (display binary version), `config_init.go` (invoke `cue mod tidy`)
- **Dependencies**: No new Go module dependencies — uses `os/exec` from stdlib
- **Platform portability**: Must work cross-platform (Linux, macOS, Windows)
  - Use `exec.LookPath("cue")` for PATH search (cross-platform)
  - Use `exec.Command` for invocation (cross-platform)
  - Parse `cue version` output (format is stable across platforms)
- **Testing**:
  - Unit tests with mocked binary (fake executable in temp PATH)
  - Integration tests with real `cue` binary (skipped if not available)
  - Cross-platform CI validation (Linux, macOS, Windows)
