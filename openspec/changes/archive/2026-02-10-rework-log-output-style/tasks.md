## 1. Centralized Style System (`internal/output/styles.go`)

- [x] 1.1 Create `internal/output/styles.go` with named `lipgloss.Color` constants: `ColorCyan(14)`, `ColorGreen(82)`, `ColorYellow(220)`, `ColorRed(196)`, `ColorBoldRed(204)`, `ColorGreenCheck(10)`
- [x] 1.2 Add semantic style variables: `StyleNoun` (cyan fg), `StyleAction` (bold), `StyleDim` (faint), `StyleSummary` (bold)
- [x] 1.3 Implement `StatusStyle(status string) lipgloss.Style` that maps `created/configured/unchanged/deleted/failed` to correct styles, with default fallback for unknown values
- [x] 1.4 Implement `FormatResourceLine(kind, namespace, name, status string) string` that renders `r:<path>  <status>` with dim prefix, cyan path, right-aligned colored status (min column width 48)
- [x] 1.5 Write table-driven tests for `StatusStyle` covering all five statuses plus unknown fallback
- [x] 1.6 Write table-driven tests for `FormatResourceLine` covering: namespaced resource, cluster-scoped resource (empty namespace), and alignment consistency

## 2. Logger Configuration (`internal/output/log.go`)

- [x] 2.1 Define `LogConfig` struct with `Verbose bool` and `Timestamps *bool` fields
- [x] 2.2 Refactor `SetupLogging` to accept `LogConfig` instead of `bool`. When `Timestamps` is nil, default to `true`. When `Verbose` is true, force timestamps on regardless
- [x] 2.3 Set `TimeFormat` to `"15:04:05"` and `ReportTimestamp` to `true` by default on the global logger
- [x] 2.4 Implement `ModuleLogger(name string) *log.Logger` that returns a child logger with styled `m:<name> >` prefix (dim `m:`, cyan name, dim `>`)
- [x] 2.5 Update the global `Logger` initialization (package-level var) to use timestamps on and `"15:04:05"` format
- [x] 2.6 Write tests for `SetupLogging`: verify timestamp default on, verify `--timestamps=false` disables, verify verbose forces timestamps on
- [x] 2.7 Write tests for `ModuleLogger`: verify returned logger has the correct prefix string

## 3. Config Schema Extension (`internal/config/`)

- [x] 3.1 Add `log.timestamps` field to the CUE config schema: `log?: { timestamps: bool | *true }`
- [x] 3.2 Extend the Go config struct to include `Log.Timestamps` field (default `true`)
- [x] 3.3 Wire config `Log.Timestamps` into `LogConfig.Timestamps` during initialization in `root.go`
- [x] 3.4 Write test: config with `log: { timestamps: false }` loads correctly
- [x] 3.5 Write test: config without `log` section defaults `timestamps` to `true`
- [x] 3.6 Write test: config with `log: { timestamps: "yes" }` fails CUE validation

## 4. CLI Flag (`internal/cmd/root.go`)

- [x] 4.1 Add `--timestamps` persistent flag (type: `*bool` / tri-state using `cobra`'s `OptionalBool` or manual `Changed()` check)
- [x] 4.2 Implement precedence resolution: flag (if set) > config `log.timestamps` > default (`true`)
- [x] 4.3 Update `initializeGlobals()` to construct `LogConfig` from resolved verbose + timestamps values and call `SetupLogging(cfg)`
- [x] 4.4 Update all existing `SetupLogging(verbose)` call sites to use new `LogConfig` struct

## 5. Migrate Apply Command (`internal/cmd/mod_apply.go`, `internal/kubernetes/apply.go`)

- [x] 5.1 In `mod_apply.go`, create a `ModuleLogger` scoped to the module name at the start of the apply operation
- [x] 5.2 Replace `output.Info(fmt.Sprintf("  %s/%s applied", ...))` calls in `kubernetes/apply.go` with `FormatResourceLine` using appropriate status (`created`, `configured`, `unchanged`)
- [x] 5.3 Determine resource status from the server-side apply response (created vs. configured vs. unchanged) and pass to `FormatResourceLine`
- [x] 5.4 Add summary line: `modLog.Info("applied successfully in <duration>")` with bold styling
- [x] 5.5 Add completion line: `output.Println("✔ Module applied")` with green checkmark styling via `StyleNoun`/`ColorGreenCheck`

## 6. Migrate Delete Command (`internal/cmd/mod_delete.go`, `internal/kubernetes/delete.go`)

- [x] 6.1 In `mod_delete.go`, create a `ModuleLogger` scoped to the module name at the start of the delete operation
- [x] 6.2 Replace `output.Info(fmt.Sprintf("  %s/%s deleted", ...))` calls in `kubernetes/delete.go` with `FormatResourceLine` using `deleted` status
- [x] 6.3 Replace `output.Info(fmt.Sprintf("  %s/%s (would delete)", ...))` dry-run lines with `FormatResourceLine` or equivalent scoped format
- [x] 6.4 Add summary line: `modLog.Info("all resources have been deleted")` with bold styling
- [x] 6.5 Add completion line: `output.Println("✔ Module deleted")` with green checkmark styling

## 7. Migrate Remaining Commands

- [x] 7.1 Review `mod_build.go`, `mod_diff.go`, `mod_status.go`, `config_vet.go` for `output.Info/Warn/Error` calls — update to use scoped loggers where a module context exists
- [x] 7.2 Update `kubernetes/apply.go` and `kubernetes/delete.go` warning lines to use scoped logger instead of global `output.Warn`

## 8. Update Existing Table Styles

- [x] 8.1 Update `table.go` to reference `ColorCyan` (or equivalent) from `styles.go` instead of hardcoded `lipgloss.Color("12")` and `lipgloss.Color("240")`

## 9. Validation Gates

- [x] 9.1 Run `task fmt` — all Go files formatted
- [x] 9.2 Run `task check` — fmt + vet + test all pass
- [x] 9.3 Run `bash docs/design/log-output-preview.sh` and visually verify the implemented output matches the preview
- [X] 9.4 Manually test `opm mod apply` and `opm mod delete` against a live cluster to verify scoped output, status colors, and timestamps
