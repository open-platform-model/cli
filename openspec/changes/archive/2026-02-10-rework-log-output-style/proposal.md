## Why

The current human-readable log output has almost no styling — no timestamps by default, no color-coded resource statuses, no scoped context prefixes, and no visual distinction between resource creation, modification, and no-op. When applying or deleting modules with many resources, the output is a wall of undifferentiated text. Users cannot quickly scan what actually changed. Timoni's bundle output demonstrates that structured, color-coded log lines with trailing status words make Kubernetes operations dramatically more scannable.

## What Changes

- Add a scoped log line format: `TIMESTAMP LEVEL m:<module> > r:<Kind/ns/name> <status>` with hierarchical chevron separators
- Add color-coded resource status suffixes (`created`, `configured`, `unchanged`, `deleted`, `failed`) with a visibility hierarchy where larger events (created) are brighter than non-events (unchanged)
- Add semantic coloring: cyan for identifiable nouns (modules, resources, namespaces), bold for action verbs, dim for structural chrome (timestamps, separators, scope prefixes)
- Enable timestamps by default using RFC 3339 time-only format (`15:04:05`)
- Add `--timestamps` flag and `log.timestamps` config field to control timestamp display
- Add centralized style definitions (`styles.go`) with named color constants and semantic style constructors
- Add a `FormatResourceLine` helper for consistent right-aligned status suffix rendering
- **FR-016 refinement**: The existing requirement says "structured logging to stderr with colors" — this change specifies _which_ colors, _what_ structure, and _how_ scoping works

## Capabilities

### New Capabilities

- `log-output-style`: Defines the human-readable log line format, color palette, semantic styling rules, resource status suffixes, scoped context prefixes, and centralized style system. Covers the `internal/output/` package styling layer.
- `log-timestamps`: Defines the timestamp display behavior — on by default, RFC 3339 time-only format, controllable via `--timestamps` flag and `log.timestamps` config field.

### Modified Capabilities

- `config`: Adding `log.timestamps` field to the CUE config schema at `~/.opm/config.cue`.

## Impact

- **`internal/output/`**: New `styles.go` file with color constants and style constructors. Changes to `log.go` for timestamp defaults and scoped logger creation.
- **`internal/cmd/root.go`**: New `--timestamps` flag, wired into `SetupLogging`.
- **`internal/config/`**: Extended CUE schema to include `log.timestamps`.
- **`internal/kubernetes/`** (apply, delete): Callers must adopt the new `FormatResourceLine` helper and scoped logger for resource status output.
- **`internal/cmd/mod/`** (apply, delete): Commands must create module-scoped loggers and emit status suffixes.
- **SemVer**: MINOR — adds new flag, new config field, and new visual behavior. No breaking changes to existing flags or output format. JSON output is unaffected.
- **Dependencies**: No new dependencies. Uses existing `charmbracelet/log` and `charmbracelet/lipgloss`.
