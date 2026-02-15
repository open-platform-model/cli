## Context

The CUE Go SDK (`cuelang.org/go`) does not expose a public API for module dependency operations like `cue mod tidy`. The actual implementation lives in `cuelang.org/go/internal/mod/modload.Tidy()`, which is behind Go's `internal` package boundary and cannot be imported by external projects.

Current state:

- OPM embeds CUE SDK v0.15.4 for evaluation and module loading
- `opm config init` creates `cue.mod/module.cue` with no `deps` field
- Users must manually run `cue mod tidy` to resolve imports (easy to forget)
- No version compatibility checking between SDK and binary
- `opm version` shows SDK version but not the user's installed `cue` binary

Investigated alternatives:

1. **Import `cuelang.org/go/cmd/cue/cmd` as library** — Package is public but only exposes `New(args) + Run(ctx)` (CLI invocation via strings). Pulls in LSP server, VCS code, ~15-20MB binary bloat. No structured API.
2. **Vendor `internal/mod/modload`** — Would require copying ~500 lines + transitive internal deps. Maintenance burden on every CUE upgrade.
3. **Wait for public API** — CUE team has been promoting packages to `mod/` but `Tidy` isn't there yet. No timeline.
4. **Shell out to `cue` binary** ✓ — Simple, no bloat, proven pattern (used by Timoni), enables future CUE operations.

## Goals / Non-Goals

**Goals:**

- Enable `opm config init` to automatically run `cue mod tidy` after file creation
- Provide version compatibility checking to warn users of SDK/binary version drift
- Make `opm version` show both SDK and binary versions for debugging
- Create reusable foundation for future CUE CLI operations (`cue mod get`, `cue export`, etc.)
- Graceful degradation when `cue` binary is not available
- Cross-platform support (Linux, macOS, Windows)

**Non-Goals:**

- Embedding the `cue` binary in the `opm` binary (would add ~50MB per platform)
- Auto-installing the `cue` binary if missing (security/complexity)
- Supporting alternative CUE implementations or forks
- Blocking execution on version mismatches (warn only — user may know it's safe)
- Fine-grained version comparison beyond major.minor (patch drift is typically safe)

## Decisions

### Decision 1: Create dedicated `internal/cue/` package

**Choice:** New package `internal/cue/` for CUE binary operations.

**Rationale:**

- **Separation of concerns** — CUE binary interaction is distinct from CUE SDK evaluation (which lives in `internal/build/`, `internal/config/`)
- **Composability** — Commands in `internal/cmd/` orchestrate; they don't implement binary discovery logic
- **Reusability** — Future commands (`opm mod tidy`, `opm mod get`) can reuse the same package
- **Testability** — Easier to mock binary execution when isolated in a single package

**Alternatives considered:**

- Put in `internal/version/` — Too narrow, doesn't belong with version metadata
- Put directly in commands — Violates separation of concerns, harder to test and reuse

**Package API:**

```go
package cue

// FindBinary locates the cue binary on PATH.
// Returns absolute path or error if not found.
func FindBinary() (string, error)

// GetVersion runs "cue version" and parses the output.
// Returns semantic version string (e.g., "v0.15.4") or error.
func GetVersion(ctx context.Context, binPath string) (string, error)

// CompatibilityStatus represents SDK/binary version comparison result.
type CompatibilityStatus struct {
    SDKVersion    string
    BinaryVersion string
    Compatible    bool   // true if major.minor match
    Warning       string // non-empty if versions differ
}

// CheckCompatibility compares SDK and binary versions (major.minor only).
// Patch differences are ignored.
func CheckCompatibility(sdkVersion, binaryVersion string) CompatibilityStatus

// RunResult captures command execution output.
type RunResult struct {
    Stdout   string
    Stderr   string
    ExitCode int
}

// Run executes "cue <args>" with inherited environment and specified working directory.
// Returns RunResult with captured output or error.
func Run(ctx context.Context, binPath string, workDir string, args ...string) (*RunResult, error)
```

---

### Decision 2: Use `exec.LookPath` for binary discovery

**Choice:** Use Go's `os/exec.LookPath("cue")` to find the binary.

**Rationale:**

- **Cross-platform** — Works on Linux, macOS, Windows (respects `%PATH%` / `$PATH`)
- **Standard library** — No external dependencies
- **Respects user environment** — Finds the `cue` binary the user would invoke from shell
- **Simple** — Single function call

**Alternatives considered:**

- Hardcoded paths (`/usr/local/bin/cue`, etc.) — Not portable, breaks if user installs elsewhere
- Custom PATH search logic — Reinvents the wheel, error-prone on Windows

**Error handling:**

- If `LookPath` fails, return `ErrBinaryNotFound` with hint: "Install CUE from <https://cuelang.org/docs/install/>"
- Callers decide whether missing binary is fatal (e.g., `opm mod tidy` fails, `opm version` shows "not found")

---

### Decision 3: Parse `cue version` output with regex

**Choice:** Invoke `cue version` and extract version with regex pattern `v\d+\.\d+\.\d+`.

**Rationale:**

- **Stable format** — `cue version` has printed `cue version vX.Y.Z` for years, unlikely to change
- **Cross-platform** — Same output format on all platforms
- **Robust** — Regex handles variations (extra text after version, release candidates like `v0.16.0-alpha.1`)

**Implementation:**

```go
func GetVersion(ctx context.Context, binPath string) (string, error) {
    cmd := exec.CommandContext(ctx, binPath, "version")
    out, err := cmd.Output()
    if err != nil {
        return "", fmt.Errorf("running cue version: %w", err)
    }
    
    // Extract version with regex: "cue version v0.15.4" -> "v0.15.4"
    re := regexp.MustCompile(`v\d+\.\d+\.\d+(-[\w.]+)?`)
    match := re.FindString(string(out))
    if match == "" {
        return "", fmt.Errorf("could not parse version from: %s", out)
    }
    return match, nil
}
```

**Alternatives considered:**

- Hardcode expected format and split on spaces — Fragile, breaks on minor format changes
- Use `cuelang.org/go/mod/module.ParseVersion` — Can't import (internal), overkill for simple parsing

**Risk mitigation:**

- If parsing fails, treat as "unknown version" rather than fatal error
- Log warning: "Could not detect CUE binary version, compatibility check skipped"

---

### Decision 4: Compare major.minor only, ignore patch

**Choice:** Version compatibility check compares major and minor versions only. Patch differences are ignored.

**Rationale:**

- **SemVer semantics** — Patch versions should be backwards-compatible (bug fixes only)
- **Practical tolerance** — CUE v0.15.3 vs v0.15.4 drift is expected and safe
- **Reduce noise** — Avoid warning users about trivial version differences

**Implementation:**

```go
func CheckCompatibility(sdkVer, binVer string) CompatibilityStatus {
    sdkMajorMinor := extractMajorMinor(sdkVer)  // "v0.15.4" -> "v0.15"
    binMajorMinor := extractMajorMinor(binVer)  // "v0.15.3" -> "v0.15"
    
    compatible := (sdkMajorMinor == binMajorMinor)
    
    var warning string
    if !compatible {
        warning = fmt.Sprintf("CUE SDK (%s) and binary (%s) versions differ. Unexpected behavior may occur.", sdkVer, binVer)
    }
    
    return CompatibilityStatus{
        SDKVersion:    sdkVer,
        BinaryVersion: binVer,
        Compatible:    compatible,
        Warning:       warning,
    }
}
```

**Alternatives considered:**

- Exact version match — Too strict, would warn on v0.15.3 vs v0.15.4
- Ignore version entirely — Misses major incompatibilities (v0.15 vs v0.16)

---

### Decision 5: Warn on mismatch, don't block execution

**Choice:** Version compatibility warnings are printed to stderr but don't prevent command execution.

**Rationale:**

- **User control** — User may know version drift is safe for their use case (e.g., only running `cue mod tidy`)
- **Avoid false positives** — Some operations work fine across minor versions
- **Fail-safe** — Better to warn and succeed than block and frustrate users

**Output format:**

- Use `output.Warn()` for structured logging (yellow, stderr)
- Example: `WARN CUE SDK (v0.15.4) and binary (v0.16.2) versions differ. Unexpected behavior may occur.`

**Future enhancement:** Could add `--strict-version` flag to make mismatches fatal if users request it.

---

### Decision 6: Integrate into `opm config init` with graceful fallback

**Choice:** `opm config init` attempts to run `cue mod tidy` after writing files. If `cue` binary is not found, prints warning and continues (preserves existing behavior).

**Rationale:**

- **Best UX** — Most users have `cue` installed; auto-running saves them a step
- **Backward compatible** — Users without `cue` binary still get config files and a helpful notice
- **Fail-safe** — Don't break the command if external binary is missing

**Implementation in `config_init.go`:**

```go
// After writing config.cue and module.cue:

// Try to run cue mod tidy
binPath, err := cue.FindBinary()
if err != nil {
    // Binary not found - warn and show manual instructions
    output.Println("")
    output.Println(output.FormatNotice("CUE binary not found. Run 'cue mod tidy' manually to resolve dependencies"))
    output.Println("Install CUE from: https://cuelang.org/docs/install/")
    return nil
}

// Check version compatibility
compat := cue.CheckCompatibility(version.CUESDKVersion, binVersion)
if !compat.Compatible {
    output.Warn(compat.Warning)
}

// Run cue mod tidy
result, err := cue.Run(ctx, binPath, paths.HomeDir, "mod", "tidy")
if err != nil {
    output.Warn("cue mod tidy failed: " + err.Error())
    output.Println("")
    output.Println(output.FormatNotice("Run 'cue mod tidy' manually in " + paths.HomeDir))
    return nil
}

// Success
output.Println("")
output.Println(output.FormatCheckmark("Dependencies resolved with cue mod tidy"))
```

**Alternatives considered:**

- Make `cue` binary required — Breaks users' existing workflows
- Skip `cue mod tidy` entirely — Doesn't solve the original problem
- Add `--skip-tidy` flag — Adds complexity, violates YAGNI (users can edit files manually if needed)

---

### Decision 7: Extend `opm version` to show binary version

**Choice:** Add CUE binary version (or "not found") to `opm version` output.

**Rationale:**

- **Debugging** — Users and maintainers can see SDK/binary versions at a glance
- **Transparency** — Makes version compatibility check visible
- **Minimal change** — Just adds 2 lines to existing output

**Output format:**

```
opm version 0.2.0
  Commit:      abc1234
  Built:       2026-02-15T10:30:00Z
  Go:          go1.23.5
  CUE SDK:     v0.15.4
  CUE binary:  v0.15.4 (compatible)
```

Or if not found:

```
  CUE binary:  not found on PATH
```

Or if incompatible:

```
  CUE binary:  v0.16.2 (version mismatch - unexpected behavior may occur)
```

**Implementation in `version.go`:**

```go
// After printing SDK version:
binPath, _ := cue.FindBinary()
if binPath == "" {
    output.Println("  CUE binary: not found on PATH")
} else {
    binVer, err := cue.GetVersion(ctx, binPath)
    if err != nil {
        output.Println("  CUE binary: " + binPath + " (version unknown)")
    } else {
        compat := cue.CheckCompatibility(info.CUESDKVersion, binVer)
        status := "compatible"
        if !compat.Compatible {
            status = "version mismatch - unexpected behavior may occur"
        }
        output.Println(fmt.Sprintf("  CUE binary: %s (%s)", binVer, status))
    }
}
```

---

### Decision 8: Inherit environment and allow working directory override

**Choice:** `cue.Run()` inherits the current process environment and allows callers to specify working directory.

**Rationale:**

- **Respect user env** — `$CUE_REGISTRY`, `$HOME`, and other env vars should propagate to `cue` subprocess
- **Flexibility** — Callers may need to run `cue` commands in different directories (e.g., module dir, config dir)
- **Standard practice** — Matches how `exec.Command` works

**Implementation:**

```go
func Run(ctx context.Context, binPath string, workDir string, args ...string) (*RunResult, error) {
    cmd := exec.CommandContext(ctx, binPath, args...)
    cmd.Dir = workDir
    cmd.Env = os.Environ()  // Inherit environment
    
    var stdout, stderr bytes.Buffer
    cmd.Stdout = &stdout
    cmd.Stderr = &stderr
    
    err := cmd.Run()
    exitCode := 0
    if err != nil {
        if exitErr, ok := err.(*exec.ExitError); ok {
            exitCode = exitErr.ExitCode()
        } else {
            return nil, fmt.Errorf("executing cue: %w", err)
        }
    }
    
    return &RunResult{
        Stdout:   stdout.String(),
        Stderr:   stderr.String(),
        ExitCode: exitCode,
    }, nil
}
```

**Alternatives considered:**

- Custom environment filtering — Overkill, breaks user expectations
- No working directory override — Limits flexibility (caller would need to chdir, not thread-safe)

---

## Risks / Trade-offs

### [Risk] External dependency on `cue` binary

**Description:** Users must install the `cue` binary separately. If it's missing or wrong version, some commands may not work as expected.

**Mitigation:**

- Graceful degradation — `config init` succeeds without binary, just skips `cue mod tidy`
- Clear error messages with installation link when binary is required but missing
- Document `cue` binary as optional dependency (required for full functionality)
- CI/CD pipelines can ensure binary is available (e.g., install via package manager or download from GitHub releases)

---

### [Risk] Version parsing brittleness

**Description:** If `cue version` output format changes, regex parsing could fail.

**Mitigation:**

- Use lenient regex that captures `vX.Y.Z` anywhere in output
- Treat parse failures as "unknown version" with warning, not fatal error
- Unit tests with various `cue version` output formats (stable releases, alpha/beta, dev builds)
- Monitor upstream CUE releases for output changes

---

### [Risk] PATH search failures on restricted environments

**Description:** In containerized or sandboxed environments, `cue` may not be on PATH even if installed.

**Mitigation:**

- Support `OPM_CUE_PATH` environment variable to override binary location
- Document how to configure PATH in Docker/CI environments
- Consider future `--cue-path` global flag if users request it

---

### [Risk] Subprocess execution failures (permissions, missing libs)

**Description:** `cue` binary may fail to execute due to permission issues, missing shared libraries, or platform incompatibilities.

**Mitigation:**

- Capture stderr and include in error messages for debugging
- Unit tests with intentionally broken binaries (non-executable, wrong arch)
- Document platform requirements in README (e.g., glibc version on Linux)

---

### [Risk] Version compatibility warnings may be false positives

**Description:** User may see warning about version mismatch when operation would actually work fine.

**Mitigation:**

- Make warnings informational, not blocking
- Include context in warning: "Unexpected behavior may occur" (not "will fail")
- Document how to suppress warnings if needed (future enhancement)

---

### [Risk] Performance overhead from subprocess invocation

**Description:** Shelling out adds ~50-100ms latency per invocation compared to in-process API.

**Mitigation:**

- Acceptable for infrequent operations (`config init`, `mod tidy`)
- Subprocess reuse not needed (operations are one-shot)
- If performance becomes issue, can explore in-memory caching of version checks

---

## Open Questions

None — all technical decisions finalized. Implementation can proceed to tasks phase.
