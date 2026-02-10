# Log Output Style Specification

> Design document for the human-readable log output system in the OPM CLI.
> This does **not** cover JSON/structured machine output.

## 1. Overview

The OPM CLI log output is designed for terminal-first scanability. It draws
inspiration from [Timoni](https://timoni.sh)'s bundle output, adapted to OPM's
domain model (modules and resources rather than bundles and instances).

### Design Principles

1. **Scanability over decoration** — The trailing status word with color coding
   is the primary affordance. Users can scan the right edge of output to see
   what changed.
2. **Noun highlighting** — All identifiable things (modules, resources,
   namespaces) share one highlight color. Actions are bold. This creates a
   consistent visual grammar.
3. **Change visibility hierarchy** — Created > configured > unchanged. Larger
   events are more visible through brighter, more saturated colors.
4. **Minimal chrome** — No boxes, borders, spinners, or progress bars in the
   log flow. A single `✔` checkmark marks completion.
5. **Idempotency-aware** — Re-runs produce calm, low-noise output where only
   actual changes pop out visually.

## 2. Line Structure

Every log line follows a strict format:

```
TIMESTAMP  LEVEL  SCOPE > MESSAGE
```

### Segments

| Segment       | Example                              | Notes                                    |
|---------------|--------------------------------------|------------------------------------------|
| **Timestamp** | `15:04:05`                           | RFC 3339 time-only, 24h format (Go layout `15:04:05`). Always shown by default. |
| **Level**     | `INFO`, `WARN`, `ERRO`, `DEBU`       | 4-char uppercase (charmbracelet/log default `MaxWidth(4)`). Fixed-width column. |
| **Scope**     | `m:my-app`                           | Module-scoped context, prefixed `m:`.     |
| **Sub-scope** | `r:Deployment/production/my-app`     | Resource-scoped context, prefixed `r:`. Optional — not all lines have it. |
| **Separator** | `>`                                  | Literal `>` in dim text. Separates each scope level and the message. |
| **Message**   | `applying module opm.dev/my-app ...` | Human-readable action/status text.        |

### Level Labels: Human vs JSON

| Level   | Human output (4-char) | JSON output (full word) |
|---------|-----------------------|-------------------------|
| Debug   | `DEBU`                | `debug`                 |
| Info    | `INFO`                | `info`                  |
| Warning | `WARN`                | `warn`                  |
| Error   | `ERRO`                | `error`                 |
| Fatal   | `FATA`                | `fatal`                 |

The 4-char labels are the charmbracelet/log `MaxWidth(4)` default. No custom
override needed — this is what the library produces out of the box.

### Hierarchy

Hierarchy is conveyed through the structured prefix fields, not indentation.
Output stays left-aligned and grep-friendly.

```
TIMESTAMP  LEVEL  m:<module> > <message>
TIMESTAMP  LEVEL  m:<module> > r:<Kind/ns/name>           <status>
```

## 3. Color Palette

### Log Levels (charmbracelet/log defaults — preserved)

| Level   | Label  | ANSI 256 | Visual      | Rendering |
|---------|--------|----------|-------------|-----------|
| Debug   | `DEBU` | `63`     | Purple      | Bold      |
| Info    | `INFO` | `86`     | Teal-green  | Bold      |
| Warning | `WARN` | `192`    | Yellow-lime | Bold      |
| Error   | `ERRO` | `204`    | Pink-red    | Bold      |
| Fatal   | `FATA` | `134`    | Magenta     | Bold      |

### Semantic Colors

| Element                           | Color              | ANSI Code     | Style     |
|-----------------------------------|--------------------|---------------|-----------|
| **Timestamp**                     | Dim gray           | Faint / `240` | —         |
| **Scope prefix** (`m:`, `r:`)    | Dim gray           | Faint         | —         |
| **Separator** (`>`)              | Dim gray           | Faint         | —         |
| **Module/resource name**          | Cyan               | `14`          | —         |
| **Namespace values**              | Cyan               | `14`          | —         |
| **Action verbs** (`applying`, `installing`, `upgrading`, `deleting`) | White | Default fg | **Bold** |
| **Version strings**               | White              | Default fg    | —         |
| **Summary lines** (`resources are ready`, `applied successfully in Xs`) | White | Default fg | **Bold** |
| **Checkmark** (`✔`)             | Green              | `10`          | —         |

### Resource Status Suffixes

These appear right-aligned at the end of resource lines. They are the primary
visual signal for what happened.

| Status        | Color          | ANSI Code     | Style     | Rationale                                  |
|---------------|----------------|---------------|-----------|--------------------------------------------|
| `created`     | **Bright green** | `82` or `10` | —         | Largest event — a new resource exists       |
| `configured`  | **Yellow**     | `220`         | —         | Medium event — something changed            |
| `unchanged`   | **Dim gray**   | Faint / `240` | —         | Non-event — fades into background           |
| `deleted`     | **Red**        | `196` or `9`  | —         | Destructive action, always visible          |
| `failed`      | **Bright red** | `204`         | **Bold**  | Error state, must not be missed             |

**Visibility hierarchy**: `created` > `deleted` > `configured` > `failed` > `unchanged`

The intent is that in a calm idempotent re-run, only the resources that
actually changed are visually prominent.

## 4. Message Patterns

### Section Flow (Apply)

```
TIMESTAMP  INFO  m:<module> > applying module <module-path> version <version>
TIMESTAMP  INFO  m:<module> > installing|upgrading <name> in namespace <namespace>
TIMESTAMP  INFO  m:<module> > r:<Kind/ns/name>                         <status>
TIMESTAMP  INFO  m:<module> > r:<Kind/ns/name>                         <status>
...
TIMESTAMP  INFO  m:<module> > resources are ready
TIMESTAMP  INFO  m:<module> > applied successfully in <duration>
✔ Module applied
```

### Section Flow (Delete)

```
TIMESTAMP  INFO  m:<module> > deleting <name> in namespace <namespace>
TIMESTAMP  INFO  m:<module> > r:<Kind/ns/name>                         deleted
TIMESTAMP  INFO  m:<module> > r:<Kind/ns/name>                         deleted
...
TIMESTAMP  INFO  m:<module> > all resources have been deleted
✔ Module deleted
```

### Error Lines

```
TIMESTAMP  ERRO  m:<module> > r:<Kind/ns/name>                        failed
TIMESTAMP  ERRO  m:<module> > apply failed: <error message>
```

### Warning Lines

```
TIMESTAMP  WARN  m:<module> > <warning message>
```

### Debug Lines (only with --verbose or config)

```
TIMESTAMP  DEBU  m:<module> > rendering module module=<path> namespace=<ns>
TIMESTAMP  DEBU  m:<module> > loaded provider name=<name>
```

## 5. Typography Rules

| Technique     | Used for                                              |
|---------------|-------------------------------------------------------|
| **Bold**      | Action verbs, summary/completion lines, level prefixes |
| **Faint/dim** | Timestamps, scope prefixes (`m:`, `r:`), separators, `unchanged` status |
| **Cyan**      | All identifiable nouns: module paths, resource names, namespace names |
| **No underline, no italic** | Not used anywhere                      |

## 6. Configuration

### Timestamp Display

Timestamps are **on by default**. They can be controlled via:

1. **Config file** (`~/.opm/config.cue`):
   ```cue
   log: {
       timestamps: bool | *true
   }
   ```

2. **CLI flag** (overrides config):
   ```
   --timestamps=false
   ```

### Verbosity

- Default: `INFO` level, timestamps on.
- `--verbose` / `-v`: `DEBU` level, timestamps on, caller info enabled.
- `--timestamps=false`: Any level, timestamps off.

## 7. Implementation Notes

### charmbracelet/log Customization Required

1. **Level rendering**: Keep the default `MaxWidth(4)` — no override needed.
   The library produces `DEBU`, `INFO`, `WARN`, `ERRO`, `FATA` out of the box.
2. **Timestamp format**: Set `TimeFormat` to `"15:04:05"` (RFC 3339 time-only).
3. **Prefix**: Use the logger's `WithPrefix()` or `With()` to inject the
   `m:<module>` scope.

### Centralized Style Definitions

A new `styles.go` file should be created in `internal/output/` containing:

- All ANSI color constants as named `lipgloss.Color` values.
- Semantic style constructors (e.g., `StatusStyle(status string) lipgloss.Style`).
- A `FormatResourceLine(kind, ns, name, status string) string` helper that
  handles right-alignment of the status suffix.

### Two-Channel Architecture (preserved)

| Channel          | Destination | Content                                  |
|------------------|-------------|------------------------------------------|
| **Log messages** | `os.Stderr` | All `INFO`/`WARN`/`ERRO`/`DEBU` lines   |
| **Data output**  | `os.Stdout` | Manifests, tables, file trees, diffs      |

The `✔ Module applied` completion line goes to **stdout** (it is data/result,
not a log message).

## 8. Examples

Run the preview script to see these with actual terminal colors:

```bash
bash docs/design/log-output-preview.sh
```

### Fresh Apply

```
15:04:05  INFO  m:my-app > applying module opm.dev/my-app version 1.2.0
15:04:05  INFO  m:my-app > installing my-app in namespace production
15:04:05  INFO  m:my-app > r:Namespace/production                              created
15:04:05  INFO  m:my-app > r:ServiceAccount/production/my-app                  created
15:04:05  INFO  m:my-app > r:ConfigMap/production/my-app-config                created
15:04:05  INFO  m:my-app > r:Deployment/production/my-app                      created
15:04:05  INFO  m:my-app > r:Service/production/my-app                         created
15:04:05  INFO  m:my-app > resources are ready
15:04:05  INFO  m:my-app > applied successfully in 8s
✔ Module applied
```

### Idempotent Re-run (partial change)

```
15:05:12  INFO  m:my-app > applying module opm.dev/my-app version 1.2.0
15:05:12  INFO  m:my-app > upgrading my-app in namespace production
15:05:12  INFO  m:my-app > r:Namespace/production                              unchanged
15:05:12  INFO  m:my-app > r:ServiceAccount/production/my-app                  unchanged
15:05:12  INFO  m:my-app > r:ConfigMap/production/my-app-config                configured
15:05:13  INFO  m:my-app > r:Deployment/production/my-app                      unchanged
15:05:13  INFO  m:my-app > r:Service/production/my-app                         unchanged
15:05:13  INFO  m:my-app > resources are ready
15:05:13  INFO  m:my-app > applied successfully in 3s
✔ Module applied
```

### Delete

```
15:06:30  INFO  m:my-app > deleting my-app in namespace production
15:06:30  INFO  m:my-app > r:Service/production/my-app                         deleted
15:06:30  INFO  m:my-app > r:Deployment/production/my-app                      deleted
15:06:31  INFO  m:my-app > r:ConfigMap/production/my-app-config                deleted
15:06:31  INFO  m:my-app > r:ServiceAccount/production/my-app                  deleted
15:06:31  INFO  m:my-app > r:Namespace/production                              deleted
15:06:31  INFO  m:my-app > all resources have been deleted
✔ Module deleted
```

### Error

```
15:07:00  INFO  m:my-app > applying module opm.dev/my-app version 1.2.0
15:07:00  INFO  m:my-app > upgrading my-app in namespace production
15:07:00  INFO  m:my-app > r:Namespace/production                              unchanged
15:07:05  ERRO  m:my-app > r:Deployment/production/my-app                      failed
15:07:05  ERRO  m:my-app > apply failed: context deadline exceeded
```
