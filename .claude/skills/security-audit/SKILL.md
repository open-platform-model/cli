---
name: security-audit
description: Security audit skill for the OPM CLI — the Go/Cobra command-line tool that loads untrusted local CUE, resolves OCI registries (often `+insecure`), pulls modules, renders scaffolding templates, and applies manifests to live Kubernetes clusters. Audits input/path handling, registry trust, artifact integrity, template injection, K8s privilege, and credential handling. Targets a path, feature, or the full project. Produces a severity-ranked report (CRITICAL / WARNING / SUGGESTION).
user-invocable: true
argument-hint: "[path-or-feature]"
---

Perform a security audit of the OPM CLI codebase. Reports findings ranked by severity — never modifies code.

The CLI is the operator-facing bridge that wires three trust zones together: **untrusted local files/CUE → an OCI registry (commonly insecure/no-TLS in this workspace) → a live Kubernetes cluster** acted on with the operator's own kubeconfig privileges. The defining risks are supply-chain (artifact substitution over `+insecure` registries, no digest verification), path handling on user-supplied file args, template injection in scaffolding, and confused-deputy application of attacker-authored manifests to the cluster. Unlike the `library` kernel, the CLI **does log** and **does perform process/cluster side effects**, so secret hygiene and K8s safety apply in full.

**Input**: Optionally specify a target after the command:

- A directory path (e.g., `internal/config/`, `internal/kubernetes/`) — scope to that subtree
- A feature name (e.g., `registry`, `apply`, `templates`, `config-loading`) — scope to code related to that feature
- Omit entirely — audit the full project (commands → config/registry → CUE load → K8s apply)

## Scope Detection

- **Path provided** → Targeted audit of that directory / file
- **Feature keyword provided** → Discover relevant code via Explore subagent, then audit
- **Nothing provided** → Full-project audit

> **Read first**: `CONSTITUTION.md` and `openspec/config.yaml` for committed direction (e.g., typed/validated provider I/O is committed but may not be implemented yet — treat current untyped handling against that target).

---

## Audit Dimensions

Nine dimensions tailored to a registry- and cluster-facing Go CLI. Each is checked against the in-scope code. Skip dimensions structurally irrelevant to the target.

### Dimension 1: Input Validation & Path Handling

- User-supplied file paths (release files, config paths, module dirs) from flags/args are validated; `filepath.Abs`/`filepath.Dir` normalize but do **not** strip `..` or resolve symlinks — assess arbitrary-read (`../../../etc/...`) and symlink-follow risk
- File paths are confined to an intended root where the command semantics expect it (no reading outside the working module/release dir without intent)
- Flag/arg values (names, namespaces, versions) validated (type, length, charset) before use in paths, registry refs, or K8s object names
- No unvalidated input concatenated into registry references, label selectors, or field selectors
- Key files: `pkg/loader/release_file.go`, `internal/config/paths.go`, `internal/cmdutil/`

### Dimension 2: Registry Authority & Insecure Transport

The supply-chain front door.

- Two-phase bootstrap (`BootstrapRegistry`) regex-extracts `registry:` from config then `os.Setenv("CUE_REGISTRY", ...)` **before** full validation — assess whether a malicious config can inject an attacker registry or extra mappings via the regex (newline/field injection into the env value)
- `defer os.Unsetenv("CUE_REGISTRY")` cleanup is correct and the global env mutation is not racy across concurrent loads
- Workspace norm uses `+insecure` (plain HTTP, no TLS) registries — flag that artifact pulls over insecure transport are MITM-substitutable; confirm production guidance/guardrails distinguish insecure-local from real registries
- Registry value from config/env is treated as security-relevant (it determines where code-like artifacts come from)
- Key files: `internal/config/loader.go` (bootstrap + CUE load)

### Dimension 3: Artifact Integrity

- Modules/catalogs are pulled via CUE's OCI loader — assess that there is **no digest/checksum/signature verification** beyond registry + (often absent) TLS trust; mutable tags are trusted
- Version resolution doesn't silently accept a downgraded or substituted module
- Pulled artifact content is validated (kind/schema via the library) before being applied to a cluster
- Key files: `internal/config/loader.go`, `pkg/loader/`

### Dimension 4: Template Injection (Scaffolding)

- Go `text/template` renders module scaffolding from user-controlled `TemplateData` (`ModuleName`, `PackageName`, etc.) — assess injection into generated file **contents** and, more importantly, into generated **file paths / names** (path traversal via a crafted module name)
- User-controlled names that become filesystem paths during scaffolding are sanitized (no `../`, no absolute paths, allowlisted charset)
- If any rendered output is HTML or shell, the correct safe package/escaping is used (`html/template`, no shell interpolation)
- Key files: `internal/templates/embed.go`

### Dimension 5: Kubernetes Privilege & Manifest Safety

The CLI acts on a live cluster with the operator's credentials.

- kubeconfig is read from a user-controlled config path — confirm the path is validated and the chosen context is explicit (no accidental wrong-cluster apply)
- Manifests applied via `client-go` are the validated output of the OPM pipeline, not arbitrary user YAML passed through unchecked (confused deputy: CLI applies attacker-authored manifests with operator privileges)
- Namespace handling is explicit — no cross-namespace apply driven by attacker-controlled fields; no privilege escalation via applied `ServiceAccount`/RBAC objects without operator intent
- Apply/delete operations are scoped and reversible; inventory tracking can't be poisoned to delete unrelated resources
- Key files: `internal/kubernetes/` (apply.go, delete.go)

### Dimension 6: Secrets & Credential Handling

The CLI logs and handles cluster + registry credentials — secret hygiene applies in full.

- Registry credentials and kubeconfig contents are never written to logs/stdout (including verbose/debug modes), error messages, or `%v`/`%+v` dumps of config structs
- K8s Secrets used for deployment inventory (e.g., `opm.demo.uuid-*`) — their data is not logged or echoed; inventory Secrets don't leak workload secrets
- No hardcoded credentials, tokens, or registry passwords in source or fixtures
- Credentials held transiently, not persisted in long-lived global state
- Key files: `internal/config/`, `internal/kubernetes/`

### Dimension 7: CUE / Provider Input Injection

- User CUE is loaded **with import resolution** — confirm imports resolve only to declared, trusted deps and the configured registry, not attacker-injected paths
- Provider config is passed as **untyped CUE values** (typed/validated provider I/O is committed direction per CONSTITUTION but may be unimplemented) — flag the unvalidated-provider-content gap against that committed target
- Provider values that influence file paths, registry targets, or K8s objects are validated before use
- Key files: `pkg/loader/provider.go`, `internal/config/loader.go`

### Dimension 8: Supply Chain & Build

- Dependencies pinned and integrity-checked: `cuelang.org/go v0.16.1`, `k8s.io/client-go v0.36.0`, `github.com/spf13/cobra` — `go.sum` present; scan for known CVEs (especially client-go and CUE)
- No `math/rand` for security-relevant values; `crypto/rand` where randomness matters
- CI workflows (`.github/workflows/`) use least-privilege `permissions:`, pin third-party actions to commit SHAs, leak no registry/cluster credentials into logs or artifacts
- Build (`Taskfile.yml`) inputs pinned; release binaries reproducible; no secrets in build args
- Key files: `go.mod`, `go.sum`, `Taskfile.yml`, `.github/workflows/`

### Dimension 9: Architecture & Trust Boundaries

Apply when scope is project-wide or covers a significant subsystem.

**Trust boundary identification**:
- Three zones: user/local files (untrusted) ↔ OCI registry (semi-trusted, often insecure) ↔ Kubernetes cluster (operator-privileged)
- Where untrusted CUE/flags cross into a registry pull, a rendered file, or a cluster mutation

**Confused deputy assessment**:
- Can an attacker-authored module/release/config cause the CLI to apply privileged resources to the cluster with the operator's credentials?
- Can config content redirect the registry to an attacker's, then deliver substituted artifacts?
- Can a crafted module name traverse the filesystem during scaffolding?

**STRIDE assessment** — for each flow crossing a boundary:

| Threat | Question |
|--------|----------|
| **Spoofing** | Can a pulled artifact be substituted (no digest, insecure transport)? Can config spoof the registry? |
| **Tampering** | Can artifacts be tampered between pull and apply? Can scaffolding output be injected? |
| **Repudiation** | Are cluster mutations attributable/auditable (who applied what)? |
| **Information Disclosure** | Can kubeconfig, registry creds, or Secret data leak via logs/errors? |
| **Denial of Service** | Can a crafted artifact wedge the CLI or flood the cluster with resources? |
| **Elevation of Privilege** | Can applied manifests grant ServiceAccounts/RBAC beyond operator intent? |

**Defense in depth**:
- Path validation + registry trust + schema validation (via library) + explicit cluster scoping layer up
- Least privilege on the kubeconfig context the CLI uses

---

## Technology-Specific Checks

Apply the relevant subset based on in-scope code.

### Go CLI Code (Cobra)

- All errors checked on security-sensitive calls (file open, registry load, K8s apply) — no silent `_`
- `crypto/rand` not `math/rand` for any token/nonce; safe path handling (no raw `filepath.Join` of unvalidated user segments into a read/write)
- No `os/exec` with user-interpolated args; no `text/template` for HTML/shell output
- `os.Setenv` of `CUE_REGISTRY` is paired with cleanup and not racy; prefer per-load env config where possible
- Cobra flag parsing validates required args; no injection via flag values into selectors/paths

### CUE & Registry Handling

- Registry regex (`registryRegex`) can't be tricked into capturing an injected multi-value/`+insecure` mapping from a hostile config
- Import resolution confined to trusted deps; provider values validated before influencing behavior

### Kubernetes Client

- kubeconfig path/context explicit and validated; no unintended in-cluster vs out-of-cluster confusion
- Applied objects are pipeline output, validated; namespace explicit; no wildcard delete

---

## Execution Steps

### Full-Project Audit

1. **Map commands & trust boundaries**

   Launch an Explore subagent to identify: all Cobra commands and their inputs (flags/args/files), config + registry bootstrap, CUE load + provider handling, template scaffolding, and every K8s read/write entry point with the credentials used.

2. **Audit each dimension**

   Launch Explore subagents (parallelize where independent). Each returns findings with **file path**, **line number(s)**, **what the issue is**, **why it matters**, and **severity**.

3. **Apply technology-specific checks** (Go CLI, CUE/registry, K8s client).

4. **Deduplicate, rank, and generate report.**

### Targeted Audit (Path or Feature)

1. **Identify scope** — a path is used directly; a feature keyword (e.g. `registry`, `apply`, `templates`) is resolved to related code via an Explore subagent.

2. **Apply relevant dimensions** — skip inapplicable ones. Apply Dimension 9 (Architecture) only if the target spans a trust boundary.

3. **Generate report.**

---

## Severity Classification

| Severity | Definition | Examples |
|----------|-----------|----------|
| **CRITICAL** | Exploitable vulnerability, privilege escalation, supply-chain compromise, or credential exfiltration. Must be addressed before release. | Config-injected attacker registry delivering substituted artifacts that get applied to the cluster, path traversal via crafted module name writing outside the target dir, kubeconfig/registry creds written to logs, applying attacker-authored privileged RBAC/ServiceAccount with operator credentials, hardcoded production credentials |
| **WARNING** | Security weakness with material impact, or best-practice violation that increases attack surface. Should be addressed in the current cycle. | No digest verification on pulled modules (mutable-tag trust), `+insecure` registry use without a production guardrail, unvalidated provider CUE influencing apply, unbounded/symlink-following file reads, racy `os.Setenv("CUE_REGISTRY")`, secrets risk in verbose logging |
| **SUGGESTION** | Defense-in-depth improvement, hardening recommendation, or theoretical risk with low current exploitability. Address when convenient. | Pin modules by digest, add module-name sanitization tests, validate kubeconfig context explicitly, tighten the registry regex, add a confirm prompt before cluster apply, minor log-redaction hardening |

### Classification Heuristics

- **Exploitability**: Can a malicious artifact/config author (untrusted) trigger it, or does it need the operator's own privilege?
- **Impact**: Worst-case — cluster compromise via applied manifests, supply-chain substitution, credential leak, local file overwrite?
- **Scope**: One module, the operator's cluster, or any consumer of a poisoned registry?
- **False positives**: When uncertain, prefer SUGGESTION over WARNING, WARNING over CRITICAL.
- **Confidence**: Only report findings with >= 80% confidence. If uncertain, state the uncertainty and suggest investigation rather than assert a vulnerability.

---

## Report Format

```markdown
## Security Audit Report

### Scope
- **Target**: Full project | `<path>` | Feature: `<name>`
- **Date**: YYYY-MM-DD

### Summary
| Dimension                                  | Status              |
|--------------------------------------------|---------------------|
| D1 Input Validation & Path Handling        | N issues / Clean    |
| D2 Registry Authority & Insecure Transport | N issues / Clean    |
| D3 Artifact Integrity                      | N issues / Clean    |
| D4 Template Injection                      | N issues / Clean    |
| D5 Kubernetes Privilege & Manifest Safety  | N issues / Clean    |
| D6 Secrets & Credential Handling           | N issues / Clean    |
| D7 CUE / Provider Input Injection          | N issues / Clean    |
| D8 Supply Chain & Build                    | N issues / Clean    |
| D9 Architecture & Trust Boundaries         | N issues / Skipped  |

**Totals**: X CRITICAL · Y WARNING · Z SUGGESTION

### CRITICAL (Must fix)

1. **[Title]** — `file/path:line`
   **Dimension**: (e.g., D2 Registry Authority & Insecure Transport)
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
- **Follow the supply-chain path** — the highest-value findings are where untrusted artifacts/config flow through an insecure/unverified registry into a privileged cluster apply
- **Delegate deep analysis to Explore subagents** — protect the main context window from the volume of file reads and grep operations
- **>= 80% confidence threshold** — if uncertain, state it explicitly and suggest investigation rather than assert a vulnerability
- **Always include Positive Observations** — confirm what the CLI does right (e.g., pipeline-validated manifests, env cleanup, explicit context handling)
- **Always include Skipped / Out of Scope** — the requestor needs to know what was NOT checked
- **Include code evidence** — every CRITICAL and WARNING cites a `file:line` and shows the relevant pattern
- **Be specific in recommendations** — name the file/line and the concrete change (e.g., "verify module digest in `internal/config/loader.go` before apply; reject mutable-tag pulls in non-insecure mode")
- **Distinguish insecure-local from production** — `+insecure` is intentional for local dev; the finding is the *absence of a guardrail* preventing it in production, not the local convenience itself
- **Do not overstate severity** — a theoretical risk with no current exploitability path is a SUGGESTION, not a CRITICAL
- **Respect the target scope** — a targeted audit stays in scope; note adjacent concerns under "Skipped / Out of Scope"

## Graceful Degradation

- **Only `internal/kubernetes/` in scope** → focus D5/D6/D9; skip registry/template dimensions
- **Only `internal/config/` in scope** → focus D2/D3/D7; skip K8s/template
- **Only `internal/templates/` in scope** → focus D4 (and D1 path handling); skip registry/K8s
- **No cluster/K8s code in scope** → skip D5; note in Skipped
- **Only dependency/build files in scope** → focus D8; skip runtime dimensions
- **Single small file** → skip D9 (Architecture); note in Skipped
- Always note which checks were skipped and why
