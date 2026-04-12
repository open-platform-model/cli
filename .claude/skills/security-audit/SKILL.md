---
name: security-audit
description: Security audit skill — analyzes Go CLI code for vulnerabilities in config parsing, credential handling, Kubernetes client usage, OCI registry interaction, CUE evaluation, path traversal, terminal output injection, and supply chain risks. Targets a specific path, feature, or the full project. Produces a severity-ranked report (CRITICAL / WARNING / SUGGESTION).
user-invocable: true
argument-hint: "[path-or-feature]"
---

Perform a security audit of the OPM CLI codebase. Reports findings ranked by severity — never modifies code.

**Input**: Optionally specify a target after the command:

- A directory path (e.g., `internal/config/`) — scope to that subtree
- A feature name (e.g., `credentials`, `apply`, `registry`, `config-loading`) — scope to code related to that feature
- Omit entirely — audit the full project (architecture + all code layers)

## Scope Detection

- **Path provided** → Targeted audit of that directory / file
- **Feature keyword provided** → Discover relevant code via Explore subagent, then audit
- **Nothing provided** → Full-project audit (architecture + all code layers + build config)

---

## Audit Dimensions

The audit is organized into seven dimensions. Each dimension is checked against the in-scope code. Skip dimensions that are structurally irrelevant to the target (e.g., skip Registry & Supply Chain when auditing only output formatting).

### Dimension 1: Input Validation & Config Security

Config files, CLI flags, environment variables, CUE evaluation.

- **CUE evaluation safety**: User-supplied `.cue` files (release files, values files) are evaluated by the CUE SDK without sandbox or timeout
  - Check for resource exhaustion vectors: deeply nested structures, expensive unification, recursive definitions
  - Verify schema validation happens before full evaluation where possible (two-phase loading pattern)
  - Confirm CUE contexts are created fresh per operation, not shared/global
- **Config file validation**: `~/.opm/config.cue` validated against embedded `#CLIConfig` schema
  - Verify validation happens before config values are used
  - Check for config values that become file paths, URLs, or shell arguments without further validation
  - Ensure bootstrap regex extraction (phase 1) cannot be tricked into extracting wrong registry URL
- **Flag validation**: Cobra flags validated at parse time
  - Verify flags that accept paths, URLs, or names are validated before use
  - Check for flags that accept credentials directly (should use file or stdin instead)
- **Environment variable handling**: `OPM_REGISTRY`, `CUE_REGISTRY`, `OPM_CONFIG`, `KUBECONFIG`
  - Verify env var values are validated before use (especially URL-shaped values)
  - Check for env vars that could override security-sensitive defaults
  - Ensure env var fallback order is documented and predictable

### Dimension 2: Credential & Secret Handling

Kubeconfig, registry tokens, environment variable leakage, error message leakage.

- **Kubeconfig security**:
  - Verify kubeconfig path discovery follows standard client-go conventions
  - Check kubeconfig is never logged, printed, or included in error messages
  - Verify no manual kubeconfig parsing that bypasses client-go's credential handling
  - Check for `insecure-skip-tls-verify` handling — tool should not silently accept insecure clusters
- **Registry credentials**:
  - Check if registry URLs with embedded credentials are logged or printed
  - Verify CUE_REGISTRY/OPM_REGISTRY values with credentials don't appear in error output
  - Check for credential caching in memory or on disk
- **Secret data in error messages**:
  - Audit all `fmt.Errorf` and log calls that include URL, path, or config values
  - Verify registry URLs are sanitized (credentials stripped) before logging
  - Check that K8s API errors don't bubble up with token or cert details
- **Secret data in terminal output**:
  - Verify `--verbose` / debug mode doesn't expose credentials
  - Check that inventory Secrets (metadata, not credentials) don't leak sensitive fields
  - Audit structured log key-value pairs for credential-shaped values
- **File permissions**:
  - Verify files written by CLI use restrictive permissions (0600 for config, 0700 for directories)
  - Check that no credential-containing files are written with permissive modes
- **No hardcoded credentials**: No tokens, passwords, keys, or API secrets in source code

### Dimension 3: Kubernetes Client Security

client-go usage, RBAC assumptions, server-side apply, field manager, ownership enforcement.

- **Server-side apply behavior**:
  - Check `Force: true` usage in ApplyPatchOptions — this overrides other tools' field managers without warning
  - Verify field manager name is consistent and identifiable
  - Audit dry-run support — ensure dry-run flag is correctly propagated
- **Ownership enforcement**:
  - Verify controller/CLI ownership checks prevent CLI from mutating controller-managed resources
  - Check ownership detection handles edge cases (missing fields, legacy resources)
- **Deletion safety**:
  - Verify foreground deletion propagation is used correctly
  - Check for cascading deletion scope — ensure only owned resources are deleted
  - Audit `--create-namespace` behavior — does it check namespace existence first?
- **API warning handling**:
  - Verify API deprecation warnings are surfaced to user (not silently suppressed by default)
  - Check that warning suppression requires explicit user opt-in
- **Error handling on API calls**:
  - Verify Forbidden/Unauthorized errors produce actionable user messages
  - Check that conflict errors (optimistic locking) are handled correctly
  - Ensure NotFound handling (`client.IgnoreNotFound`) is used only where safe
- **No impersonation**: Verify tool does not use K8s impersonation unless explicitly requested by user

### Dimension 4: OCI Registry & Supply Chain

Registry interaction, module verification, dependency integrity, build reproducibility.

- **Registry TLS**:
  - Check if `+insecure` registry suffix is supported and what it means for TLS
  - Verify custom CA certificate support exists for private registries
  - Ensure HTTPS is default for registry communication
- **Module integrity**:
  - Check if CUE module content is verified against checksums after download
  - Verify module resolution doesn't allow fallback to untrusted registries
  - Audit registry URL precedence logic — can a malicious env var override trusted registries?
- **Go dependency supply chain**:
  - Verify `go.mod` and `go.sum` are committed and CI validates checksums
  - Check for typosquat-risk dependency names
  - Audit `GOPROXY` settings — should use official proxy, not untrusted mirrors
  - Check for abandoned/unmaintained transitive dependencies
- **Build reproducibility**:
  - Verify goreleaser config uses pinned Go versions
  - Check ldflags inject only version info, no secrets
  - Verify checksums are generated for release artifacts
  - Audit `.gitignore` / `.dockerignore` for credential file exclusions

### Dimension 5: File System & Path Security

Path traversal, symlink safety, file operations on user-supplied paths.

- **Path traversal prevention**:
  - Audit all `filepath.Join(baseDir, userInput)` patterns — can `../` escape the base?
  - Check tilde expansion (`ExpandTilde`) for injection vectors
  - Verify no user-supplied path is used with `os.Open`, `os.Create`, `os.ReadFile` without validation
  - For Go 1.24+: check if `os.Root` / `os.OpenInRoot` is used for untrusted paths
- **Template rendering**:
  - Verify template data comes from embedded files only, not user input
  - Check `text/template` usage — if user data enters templates, verify no code injection
- **Config file creation**:
  - Verify `config init` creates files with secure permissions
  - Check that `cue.mod/` directory structure is created safely
- **Symlink safety**:
  - If following symlinks for config or module resolution, verify they don't escape intended directory
- **No external command execution**:
  - Verify no `os/exec` calls exist in production code
  - If any subprocess execution exists, verify arguments are not constructed from user input

### Dimension 6: Terminal Output Security

ANSI escape injection, sensitive data in output, information disclosure in errors.

- **ANSI escape injection**:
  - Identify all output that includes user-controlled data (resource names, namespaces, config values)
  - Check if Kubernetes resource names/labels could contain ANSI escape sequences
  - Verify charmbracelet/lipgloss styling doesn't pass through raw escape codes from data
  - Check error messages from external sources (K8s API, CUE SDK) for escape sequences
- **Information disclosure in errors**:
  - Verify internal file paths don't leak system username or directory structure
  - Check that CUE validation errors don't expose full config content
  - Audit K8s API error messages for server-side information leakage
  - Verify stack traces are never shown in non-debug mode
- **Output formatting safety**:
  - Check `%v` / `%+v` formatting on objects that may contain sensitive data
  - Verify manifest output (`writeYAML`) doesn't include Secret data fields
  - Audit diff output for sensitive field exposure
- **Log level discipline**:
  - Verify DEBUG logging doesn't expose credentials even when enabled
  - Check that production log levels filter sensitive information
  - Ensure structured logging (charmbracelet/log) escapes values correctly

### Dimension 7: Architecture & Trust Boundaries

Apply when scope is project-wide or covers a significant subsystem.

**Trust boundary identification**:
- Map where privilege levels change: user input → CLI parsing → CUE evaluation → K8s API calls → cluster state changes
- CUE evaluation boundary: untrusted `.cue` files → evaluated by CUE SDK → produces K8s manifests
- Registry boundary: CUE module resolution → OCI registry fetch → evaluated as trusted code
- K8s API boundary: CLI holds credentials → makes authenticated API calls → modifies cluster state

**Confused deputy assessment**:
- Can a malicious `.cue` release file cause the CLI to:
  - Apply resources to unintended namespaces?
  - Delete resources it shouldn't?
  - Fetch modules from attacker-controlled registries?
  - Write files to unexpected locations?
- Can CUE module imports introduce untrusted code into the evaluation?
- Can K8s API warnings or errors be crafted to inject content into CLI output?

**STRIDE assessment** — for each major component or data flow crossing a trust boundary:

| Threat | Question |
|--------|----------|
| **Spoofing** | Can a malicious registry serve spoofed CUE modules? Can CUE files reference attacker-controlled registries? |
| **Tampering** | Can CUE modules be modified between download and evaluation? Can K8s resources be modified between render and apply? |
| **Repudiation** | Are apply/delete operations logged? Can inventory track who applied what? |
| **Information Disclosure** | Can credentials leak through errors, logs, or output? Can CUE evaluation expose config secrets? |
| **Denial of Service** | Can malicious CUE cause CPU exhaustion? Can oversized configs cause memory exhaustion? Can finalizer-heavy resources block cleanup? |
| **Elevation of Privilege** | Can a release file escalate cluster permissions via the CLI's credentials? Can CUE module imports execute arbitrary code? |

**Defense in depth**:
- Validate at each layer: CLI flags → config schema → CUE evaluation → K8s API admission
- No single validation layer (CLI-side or server-side) should be the sole defense

---

## Technology-Specific Checks

Apply the relevant subset based on in-scope code.

### Go Code Patterns

- `crypto/rand` used instead of `math/rand` for any security-relevant values (nonces, tokens, identifiers)
- No `text/template` with user-controlled input — verify template data sources
- No `fmt.Sprintf` constructing URLs, label selectors, or field selectors from user input without validation
- All error returns checked in security-sensitive paths — no silent `_` for errors from `Get`, `Create`, `Update`, `Delete`, `Patch`
- No `%v` or `%+v` formatting of objects that may contain credential data
- Race conditions: no shared mutable state (maps, slices) without mutex or channels if concurrent access is possible
- Error wrapping preserves sentinels (`%w`) but doesn't include raw credential values in context strings
- `filepath.Clean` / `filepath.Join` used correctly — aware of platform-specific traversal (CVE-2022-41722 on Windows)

### Cobra & CLI Framework

- No credential-accepting flags that would be visible in `ps aux` — prefer `--token-file` or stdin over `--token`
- Help text and examples don't include example credentials or real URLs
- Unknown subcommands produce errors (not silent no-ops)
- `--version` output doesn't expose internal system paths or build environment details
- `RunE` used consistently — errors returned, not printed and exited inline

### CUE SDK Usage

- Fresh `cue.Context` created per operation — no shared mutable CUE state
- CUE loading uses explicit overlays and config, not ambient filesystem scanning
- Registry URL set via env var before CUE load — verify env var is sanitized
- CUE validation errors include helpful context but not full config values that may be sensitive
- Module resolution follows explicit registry config, not implicit fallback chain

### Kubernetes client-go Usage

- Client creation follows standard kubeconfig resolution (not custom parsing)
- REST config not modified to disable TLS or add insecure transport
- Context propagation: all API calls accept and honor `context.Context` for cancellation
- Server-side apply: field manager name is descriptive and unique to this tool
- Error classification: distinguish auth errors, RBAC errors, conflict errors, not-found errors with appropriate user messages

### Dependencies & Build

- `go.mod` pins specific versions (no `v0.0.0-` pseudo-versions for critical deps unless necessary)
- `go.sum` committed and verified in CI
- `golangci-lint` includes `gosec` linter for static security analysis
- No vendored dependencies with local modifications that diverge from upstream
- goreleaser config: no secrets in build environment, checksums generated for all artifacts

---

## Execution Steps

### Full-Project Audit

1. **Map the attack surface**

   Launch an Explore subagent to identify:
   - All external input sources: flags, config files, env vars, CUE files, stdin
   - All credential touchpoints: kubeconfig resolution, registry auth, env vars
   - All K8s API interactions: apply, delete, get, list, patch, namespace creation
   - All CUE evaluation points: config loading, release files, values files, module packages
   - All file system operations: reads, writes, path construction, permission handling
   - All output paths: stdout, stderr, structured logs, error messages
   - Registry interaction patterns and trust assumptions
   - Build and release configuration

2. **Audit each dimension**

   Launch Explore subagents (parallelize where independent) to check each dimension against the relevant code. Each subagent must return findings with: **file path**, **line number(s)**, **what the issue is**, **why it matters**, and **severity** (CRITICAL / WARNING / SUGGESTION).

3. **Apply technology-specific checks**

   Check Go code patterns, Cobra framework usage, CUE SDK patterns, client-go usage, and build configuration against the technology-specific checklists.

4. **Deduplicate, rank, and generate report**

### Targeted Audit (Path or Feature)

1. **Identify scope**

   If a path: use that directory directly.
   If a feature keyword: launch an Explore subagent to find all code related to that feature.

2. **Apply relevant dimensions and technology checks**

   Skip dimensions that don't apply. Apply Dimension 7 (Architecture) only if the target spans a trust boundary.

3. **Generate report**

---

## Severity Classification

| Severity | Definition | Examples |
|----------|-----------|----------|
| **CRITICAL** | Exploitable vulnerability, credential exposure, privilege escalation, or auth bypass. Must be addressed before release. | Credentials logged in plaintext, path traversal allowing arbitrary file read, command injection via user input, CUE evaluation executing code, registry MITM with no TLS verification, hardcoded secrets in source |
| **WARNING** | Security weakness with material impact, or best-practice violation that increases attack surface. Should be addressed in the current cycle. | Missing input validation on user-supplied paths, credentials potentially in error messages, Force=true without user awareness, debug logging exposes sensitive config values, insecure file permissions, missing HTTPS enforcement for registries |
| **SUGGESTION** | Defense-in-depth improvement, hardening recommendation, or theoretical risk with low current exploitability. Address when convenient. | CUE evaluation timeout/limits, ANSI escape sanitization in output, enhanced credential redaction in structured logs, additional schema validation for edge cases, supply chain hardening (SBOM, signatures) |

### Classification Heuristics

- **Exploitability**: Can this be triggered by a user running the CLI with crafted input? Does it require control of the environment (env vars, config files)?
- **Impact**: What is the worst-case outcome? (credential theft, cluster compromise, data exfiltration, local file access, denial of service)
- **Scope**: How many users/environments are affected? (all users, specific configurations, CI/CD only)
- **False positives**: When uncertain, prefer SUGGESTION over WARNING, WARNING over CRITICAL
- **Confidence**: Only report findings with >= 80% confidence. If uncertain, state the uncertainty and suggest investigation rather than assert a vulnerability

---

## Report Format

```markdown
## Security Audit Report

### Scope
- **Target**: Full project | `<path>` | Feature: `<name>`
- **Date**: YYYY-MM-DD

### Summary
| Dimension                        | Status              |
|----------------------------------|---------------------|
| Input Validation & Config        | N issues / Clean    |
| Credential & Secret Handling     | N issues / Clean    |
| Kubernetes Client Security       | N issues / Clean    |
| OCI Registry & Supply Chain      | N issues / Clean    |
| File System & Path Security      | N issues / Clean    |
| Terminal Output Security         | N issues / Clean    |
| Architecture & Trust Boundaries  | N issues / Skipped  |

**Totals**: X CRITICAL · Y WARNING · Z SUGGESTION

### CRITICAL (Must fix)

1. **[Title]** — `file/path:line`
   **Dimension**: (e.g., Credential & Secret Handling)
   **Description**: What the issue is and how it could be exploited
   **Evidence**: Code snippet or pattern observed
   **Recommendation**: Specific fix with file/line target

### WARNING (Should fix)

1. **[Title]** — `file/path:line`
   **Dimension**: ...
   **Description**: ...
   **Evidence**: ...
   **Recommendation**: ...

### SUGGESTION (Nice to fix)

1. **[Title]** — `file/path:line`
   **Dimension**: ...
   **Description**: ...
   **Recommendation**: ...

### Positive Observations
- (Security practices done well — always include at least one)

### Skipped / Out of Scope
- (Dimensions or checks skipped and why)

### Final Assessment
- If CRITICAL issues: "X critical issue(s) found. Address before release."
- If only warnings: "No critical issues. Y warning(s) to consider."
- If all clear: "All checks passed. No security issues identified in scope."
```

---

## Guardrails

- **NEVER make code changes** — this skill is analysis and reporting only
- **Delegate deep analysis to Explore subagents** — protect the main context window from the volume of file reads and grep operations
- **>= 80% confidence threshold** — if uncertain, state it explicitly and suggest investigation rather than assert a vulnerability
- **Always include Positive Observations** — an audit that only reports negatives erodes trust and misses the value of confirming what works
- **Always include Skipped / Out of Scope** — the requestor needs to know what was NOT checked
- **Include code evidence** — every CRITICAL and WARNING must cite a file:line reference and show the relevant code pattern
- **Be specific in recommendations** — "fix the credential handling" is not actionable; "strip credentials from registry URL before logging at `internal/config/loader.go:102` using `url.Redacted()`" is
- **Do not overstate severity** — a theoretical risk with no current exploitability path is a SUGGESTION, not a CRITICAL. Crying wolf undermines the report
- **Respect the target scope** — a targeted audit stays in scope. Note adjacent concerns in "Skipped / Out of Scope" rather than expanding unbounded
- **Actionability** — every issue must have a specific recommendation with file/line references where applicable. No vague "consider reviewing" suggestions

## Graceful Degradation

- If target is a single Go file: skip Dimensions 4, 5, 7 (Registry, Path, Architecture), note in Skipped
- If target is config loading only: focus Dimensions 1, 2, 5; skip K8s and Registry, note in Skipped
- If target is K8s interaction only: focus Dimensions 3, 6, 7; skip Config and Registry, note in Skipped
- If target is output formatting only: focus Dimension 6; skip all others, note in Skipped
- If no K8s interaction in scope: skip Dimension 3 checks, note in Skipped
- If no registry interaction in scope: skip Dimension 4 checks, note in Skipped
- Always note which checks were skipped and why
