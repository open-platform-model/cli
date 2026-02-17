## Context

The CLI currently uses `charmbracelet/log` (v0.4.2) for human-readable terminal output with styled text, colors via `lipgloss`, and Unicode symbols. All output goes to two channels: logs/errors on stderr, data on stdout. This works well for interactive terminal use but breaks CI/CD integration where JSON parsing is standard.

Current architecture:
- `internal/output` package provides all output primitives
- Global `*log.Logger` configured once in `root.go`'s `PersistentPreRunE`
- ~70+ call sites use `output.Debug/Info/Warn/Error`
- Commands print results via `output.Println()` with styled formatters
- `internal/config` already has precedence resolution pattern (flag > env > config > default)
- Two commands (`mod build`, `mod status`) already have per-command `-o` flags for data format

## Goals / Non-Goals

**Goals:**
- Enable CI/CD usage: `export OPM_FORMAT=json` → all commands emit machine-parseable JSON
- Preserve existing human-readable text mode as the default
- Zero changes to ~70+ existing log call sites (leverage charmbracelet/log's built-in JSONFormatter)
- Consistent JSON structure across all commands (shared envelope)
- Keep `--format` and `-o` independent (different concerns)

**Non-Goals:**
- Mixing text and JSON in a single invocation (format is global, set once)
- NDJSON streaming output (use single-shot envelope per command)
- Migrating from `charmbracelet/log` to `log/slog` (unnecessary, current lib supports JSON)
- Adding `-o json` to every command (only commands with data outputs get envelopes)

## Decisions

### Decision 1: Use `charmbracelet/log`'s built-in JSONFormatter

**Alternatives considered:**
- **A. Migrate to `log/slog`** — Stdlib, native JSON handler, well-integrated. But requires replacing all ~70+ call sites and the entire logger setup. High effort, no benefit over the existing solution.
- **B. Write custom formatter** — Full control. But `charmbracelet/log` already has `JSONFormatter` that works. Why reinvent it?
- **C. Use built-in JSONFormatter (chosen)** — Zero call-site changes. Single line: `logger.SetFormatter(log.JSONFormatter)` in `SetupLogging()`.

**Rationale:** `charmbracelet/log` v0.4.2 already has `JSONFormatter`. Setting it switches all log output to structured JSON automatically:
```json
{"time":"10:04:05","level":"info","prefix":"m:jellyfin","msg":"transformer matched","component":"app"}
```
No call-site changes. No new dependencies. Minimal code diff.

### Decision 2: Shared JSON envelope for commands without `-o`

Commands that don't have their own data serialization flag get a standard envelope:

```json
{
  "command": "mod apply",
  "success": true,
  "result": { ... },
  "warnings": ["..."],
  "errors": ["..."]
}
```

**Alternatives considered:**
- **A. NDJSON (newline-delimited JSON)** — Streaming-friendly. But adds complexity for commands that aren't long-running. Most CI scripts just want `jq .success`.
- **B. Separate envelope per artifact type** — Per-command structure. But inconsistent for parsers. Every CI tool needs different logic per command.
- **C. Shared envelope (chosen)** — Single, predictable structure. `result` field is command-specific, but top-level keys are universal.

**Rationale:** CI/CD scripts want consistency. `jq .success` works across all commands. Result data goes in `result` field, shape varies per command. Simple, predictable.

### Decision 3: Commands with `-o` skip the envelope

`mod build` and `mod status` already have `-o yaml|json|table` flags. These control data output format (manifests, status records). In JSON mode, we only switch their logs to JSON — data output follows `-o` as-is.

**Example:**
```bash
opm mod build --format json -o yaml   # stderr: JSON logs, stdout: raw YAML manifests
opm mod build --format text -o json   # stderr: styled logs, stdout: raw JSON manifests
```

**Alternatives considered:**
- **A. Envelope everything** — Wrap `-o` output in envelope. But this breaks pipes: `opm mod build -o yaml | kubectl apply -f -` expects raw YAML, not `{"result":{"manifests":"..."}}`.
- **B. `-o` overrides `--format`** — If `-o` set, ignore `--format` entirely. But then logs stay styled in CI, defeating the purpose.
- **C. Skip envelope for commands with `-o` (chosen)** — `--format` controls logs/errors on stderr, `-o` controls data on stdout. Independent concerns.

**Rationale:** `--format` is about CLI chrome (logs, errors, command results). `-o` is about data serialization (manifests, status). When both exist, honor both. Data commands (`build`, `status`) emit raw data. Action commands (`apply`, `delete`) emit result envelopes.

### Decision 4: `--format` flag (not `--output` or `--log-format`)

**Alternatives considered:**
- **A. `--log-format`** — Descriptive for logs. But undersells it — format affects stdout results too, not just logs.
- **B. `--output`** — Short, clear. But conflicts with per-command `-o` flags (`mod build -o yaml`). Confusing to have global `--output` and local `-o`.
- **C. `--format` (chosen)** — Clear scope: this is about the format of the entire CLI interaction (logs + results). Doesn't collide with `-o`.

**Rationale:** `--format` is global (affects logs, errors, command results). `-o` is per-command (affects data serialization). Distinct names for distinct concerns.

### Decision 5: Resolution via flag > env > config > default

Follow the existing pattern from `internal/config/resolver.go`:

| Source | Example | Precedence |
|--------|---------|------------|
| Flag | `--format json` | 1 (highest) |
| Env | `OPM_FORMAT=json` | 2 |
| Config | `config.cue: format: "json"` | 3 |
| Default | `"text"` | 4 (lowest) |

**Rationale:** Consistent with every other config value in the CLI. CI scripts can `export OPM_FORMAT=json` once and all `opm` invocations follow. No need to `--format json` on every command.

### Decision 6: Suppress `Println`/`Print` in JSON mode

In JSON mode, `output.Println()` / `output.Print()` become no-ops. All stdout data must go through `WriteResult()`.

**Alternatives considered:**
- **A. Buffer Println output, include in envelope** — Collect styled strings, shove into `result.output[]`. But then you get `["✔ Module applied"]` in JSON, which is meaningless. The checkmark is visual, not semantic.
- **B. Keep Println, emit raw text to stdout** — Mix JSON logs (stderr) and text results (stdout). Violates the format contract.
- **C. Suppress Println, require structured results (chosen)** — Commands must build result structs and call `WriteResult()`. Forces structured thinking.

**Rationale:** In JSON mode, stdout should be valid JSON. Styled strings like `"✔ Module applied"` aren't useful to parsers. Commands must emit structured data via `WriteResult()`.

### Decision 7: Interactive commands require flag-only mode in JSON mode

`mod init` and `config init` use `charmbracelet/huh` for interactive prompts. In JSON mode, prompts are suppressed. All inputs must come from flags.

**Alternatives considered:**
- **A. Allow prompts, emit JSON after** — Prompt on stderr (it's a TTY stream anyway), then emit JSON result. But in CI, there's no TTY. Prompts would block forever.
- **B. Auto-fill defaults** — Skip prompts, use defaults for everything. But users expect to provide inputs. Silent defaults are surprising.
- **C. Require flags in JSON mode (chosen)** — If `output.IsJSON()` and required flags missing, return error: `"interactive prompts not supported in JSON mode; provide --template, --path, etc."`

**Rationale:** JSON mode signals non-interactive use (CI/CD). Prompts would block. Requiring flags makes the expectation explicit.

### Decision 8: Error JSON structure mirrors DetailError fields

`DetailError` already has structured fields:
```go
type DetailError struct {
    Type     string
    Message  string
    Location string
    Field    string
    Context  map[string]string
    Hint     string
    Cause    error
}
```

In JSON mode, `DetailError.MarshalJSON()` emits these as-is. The error goes into the envelope's `errors` array.

**Rationale:** DetailError already captures structured error info. Don't invent a new format — just serialize what's there.

## Risks / Trade-offs

**Risk: Commands with `-o` have inconsistent output shape vs commands without `-o`**
- `mod build --format json -o yaml` → stdout is raw YAML
- `mod apply --format json` → stdout is JSON envelope
→ **Mitigation:** Documented behavior. CI scripts know which command they're calling. Data commands (build, status) are for generating artifacts. Action commands (apply, delete) are for reporting outcomes. Different purposes, different shapes.

**Risk: Envelope buffering breaks streaming for long operations**
- Commands must collect full result before emitting envelope
- For `mod apply` with 50 resources, user sees no output until the end
→ **Mitigation:** Logs still stream to stderr in real-time (whether text or JSON). Stdout waits for final envelope. This is standard for tools with JSON output (kubectl, terraform).

**Risk: Adding new result fields is a breaking change for parsers**
- If we add `result.new_field` later, parsers might break
→ **Mitigation:** Version the envelope? Add `"version": "v1"` to envelope. Or document that `result` is command-specific and may evolve. For now, ship without version field. Add if needed.

**Risk: Interactive command flag requirement confuses users**
- User runs `opm mod init --format json` expecting prompts, gets error
→ **Mitigation:** Clear error message: `"interactive prompts not supported in JSON mode; provide all required flags: --template, --path"`. List missing flags.

**Trade-off: No JSON output for logs in text mode**
- Some users might want JSON logs but text results
→ **Accepted:** `--format` is global. It's all-or-nothing. Adding separate `--log-format` and `--result-format` flags violates simplicity (Principle VII). If needed later, we can split it. YAGNI for now.

**Trade-off: Verbose JSON output from `mod build --verbose-json` is different from `--format json`**
- `--verbose-json` is the existing build-specific flag, outputs a different JSON structure
→ **Accepted:** `--verbose-json` is for debugging the build process (transformer matches, component metadata). `--format json` is for CI integration (command success/failure). Different use cases. Keep both.

## Migration / Deployment

No migration needed. This is purely additive:
- Default format is `text` — all existing behavior unchanged
- New `--format` flag is optional
- Config file `format` field is optional
- Commands gain new JSON branches, old text branches untouched

Rollout:
1. Merge infrastructure (output package, config, root flag)
2. Merge per-command JSON results incrementally (start with `version`, `mod vet` — low risk)
3. Document in CLI help and README
4. CI examples in docs

Rollback: Remove `--format` flag, revert config schema, delete `WriteResult` paths. Text mode always worked, so reverting is safe.
