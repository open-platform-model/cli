## Context

The `internal/output/` package currently provides a thin wrapper around `charmbracelet/log` with no custom styling. The global `Logger` is configured in `log.go` with timestamps off by default and no semantic color system. Resource operations (apply, delete) emit plain `output.Info()` calls with no status suffixes or scoped prefixes. The only lipgloss usage is in `table.go` for table headers.

A detailed visual style specification has been developed in `docs/design/log-output-style.md` with a companion color preview script. This design document covers the technical approach to implementing that specification.

The existing two-channel architecture (log messages → stderr, data output → stdout) is preserved. JSON output is unaffected by this change.

## Goals / Non-Goals

**Goals:**

- Implement the color palette and line format defined in `docs/design/log-output-style.md`
- Create a centralized style system in `internal/output/styles.go` with named color constants and semantic style constructors
- Provide a `FormatResourceLine` helper that renders `r:<Kind/ns/name>  <status>` with right-aligned, color-coded status suffixes
- Provide a scoped logger factory (`ModuleLogger`) that injects the `m:<module> >` prefix
- Enable RFC 3339 time-only timestamps (`15:04:05`) by default
- Add `--timestamps` flag and `log.timestamps` config field with precedence: flag > config > default (true)
- Keep the existing `Debug/Info/Warn/Error/Fatal` convenience functions working unchanged for non-scoped callers

**Non-Goals:**

- Changing JSON/structured machine output format
- Adding spinners, progress bars, or interactive TUI elements
- Changing the table rendering in `table.go` (separate concern)
- Adding log-level configuration beyond the existing `--verbose` flag
- Implementing the completion line (`✔ Module applied`) — that is a command-level concern using existing `output.Println`, not a logger feature

## Decisions

### 1. New `styles.go` with lipgloss color constants

**Decision**: Create `internal/output/styles.go` containing all named `lipgloss.Color` values and semantic style constructors.

**Why**: Currently colors are scattered (table headers hardcode `"12"`, `"240"`). A central palette ensures consistency and makes future theming possible. All color decisions from the style spec are encoded in one place.

**Alternatives considered**:

- Inline colors at call sites — rejected: leads to inconsistency and makes palette changes painful.
- A theme config file — rejected: YAGNI (Principle VII). Named Go constants are sufficient for now.

**Structure**:

```go
// Color palette
var (
    ColorCyan      = lipgloss.Color("14")
    ColorGreen     = lipgloss.Color("82")
    ColorYellow    = lipgloss.Color("220")
    ColorRed       = lipgloss.Color("196")
    ColorBoldRed   = lipgloss.Color("204")
    ColorGreenCheck = lipgloss.Color("10")
)

// Semantic styles
var (
    StyleNoun      = lipgloss.NewStyle().Foreground(ColorCyan)
    StyleAction    = lipgloss.NewStyle().Bold(true)
    StyleDim       = lipgloss.NewStyle().Faint(true)
    StyleSummary   = lipgloss.NewStyle().Bold(true)
)

// StatusStyle returns the lipgloss style for a resource status.
func StatusStyle(status string) lipgloss.Style { ... }
```

### 2. `FormatResourceLine` helper for right-aligned status

**Decision**: Add a `FormatResourceLine(kind, namespace, name, status string) string` function that constructs the full `r:<path>  <status>` string with the status suffix padded to a fixed column.

**Why**: Right-alignment of status words is the primary scanability affordance. Doing this in a helper ensures every resource line across apply, delete, and status commands looks identical.

**Alignment strategy**: Compute a target column width based on the longest resource path in the current batch. Fall back to a minimum width of 48 characters. The status word starts at `targetWidth + 2` (two-space gap).

**Alternative considered**:

- Fixed global column width — rejected: resource paths vary wildly in length. A per-batch calculation adapts better.

### 3. Scoped logger via `ModuleLogger(name string) *log.Logger`

**Decision**: Add a factory function that returns a `charmbracelet/log.Logger` with the `m:<name> >` prefix baked in. Commands call `modLog := output.ModuleLogger("my-app")` and use `modLog.Info(...)` instead of the global `output.Info(...)`.

**Why**: The `m:<module>` scope prefix must appear on every line within a module operation. charmbracelet/log's `WithPrefix()` handles this natively — the prefix is rendered with the configured style (dim for the `m:` prefix, cyan for the name).

**How prefix styling works**: charmbracelet/log applies the `Styles.Prefix` style to the prefix string. We set a custom `Styles` on the scoped logger where `Prefix` uses faint + cyan formatting for the module name portion. The `>` separator is included in the prefix string.

**Alternative considered**:

- Prepending scope text in each `fmt.Sprintf` call — rejected: error-prone, duplicated, and can't leverage charmbracelet/log's built-in prefix rendering.
- Thread-local / context-based scope — rejected: overengineered for a CLI where operations are sequential.

### 4. Timestamps on by default, configurable via flag and config

**Decision**: Change `ReportTimestamp` default from `false` to `true`. Set `TimeFormat` to `"15:04:05"`. Add a `--timestamps` persistent flag (default: unset) and read `log.timestamps` from the CUE config.

**Precedence**: `--timestamps` flag > `config.log.timestamps` > default (`true`).

**Why**: Timestamps help correlate log lines with wall-clock time during operations that take seconds to minutes. The RFC 3339 time-only format (`15:04:05`) is compact enough to not waste horizontal space.

**Alternative considered**:

- Full RFC 3339 with date (`2006-01-02T15:04:05`) — rejected: date is noise for a CLI session.
- Keep timestamps off by default — rejected: users consistently need timing info for apply/delete operations.

### 5. `SetupLogging` signature change

**Decision**: Expand `SetupLogging` to accept a config struct instead of a single `bool`:

```go
type LogConfig struct {
    Verbose    bool
    Timestamps *bool // nil = use default (true)
}

func SetupLogging(cfg LogConfig) { ... }
```

**Why**: The function currently only takes `verbose`. Adding timestamp control (and potentially future options) requires a struct to avoid a growing parameter list.

**Migration**: All existing callers pass `SetupLogging(LogConfig{Verbose: verbose})` — timestamps default to true when the pointer is nil. This is backward-compatible in behavior (timestamps were off before but are now on by default, which is the intended change).

### 6. Keep charmbracelet/log defaults for level labels

**Decision**: Do not override `MaxWidth(4)`. Level labels render as `DEBU`, `INFO`, `WARN`, `ERRO`, `FATA` — the library defaults. Level colors (`63`, `86`, `192`, `204`, `134`) are also preserved.

**Why**: These defaults are well-chosen and recognizable. The 4-char width creates a clean fixed column. No customization cost.

### 7. Resource status as message suffix, not key-value pair

**Decision**: Resource statuses (`created`, `configured`, etc.) are rendered as a styled suffix in the message string passed to the logger, not as a charmbracelet/log key-value pair.

**Why**: charmbracelet/log renders key-value pairs with `key=value` syntax and applies its own key/value styles. We need the status word to appear right-aligned with custom per-status coloring, which requires it to be part of the message string, not structured data.

**Implication**: The `FormatResourceLine` helper returns a pre-styled string. The caller passes it as the message: `modLog.Info(output.FormatResourceLine(...))`.

## Risks / Trade-offs

**[Risk] Faint/dim ANSI may not render on all terminals** → Mitigation: `lipgloss` handles terminal capability detection. On dumb terminals, faint text degrades to normal rather than becoming invisible. The color palette uses 256-color codes which are universally supported on modern terminals.

**[Risk] Right-alignment breaks with very long resource names** → Mitigation: The per-batch column calculation adapts to the longest name. If a name exceeds the minimum column width, the status word simply shifts right. This degrades gracefully — the status is still visible, just further right.

**[Risk] `SetupLogging` signature change is breaking for internal callers** → Mitigation: This is an internal API, not public. All callers are within `internal/cmd/root.go`. The migration is a single-line change at each call site.

**[Trade-off] Scoped loggers create new logger instances per module operation** → Accepted: charmbracelet/log loggers are lightweight (they just wrap an `io.Writer` with options). Creating one per `mod apply` invocation has negligible cost. This is simpler than a mutable global scope.

**[Trade-off] Status words are styled strings, not structured data** → Accepted: This means JSON output cannot trivially extract the status from the human log. But JSON output has its own structured format where status is a proper field — the two output paths are intentionally separate.
