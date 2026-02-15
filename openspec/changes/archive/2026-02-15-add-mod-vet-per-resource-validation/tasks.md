## 1. Output Package — StatusValid and FormatVetCheck

- [x] 1.1 Add `StatusValid = "valid"` constant to `internal/output/styles.go`
- [x] 1.2 Add `"valid"` case to `statusStyle()` returning green (`82`) foreground
- [x] 1.3 Add `FormatVetCheck(label, detail string) string` function to `internal/output/styles.go` — green checkmark, label, optional right-aligned dim detail at column 34
- [x] 1.4 Add unit tests for `StatusValid` in `statusStyle()` — verify green style, verify same color as `StatusCreated`
- [x] 1.5 Add unit tests for `FormatVetCheck` — with detail, without detail, alignment consistency across multiple calls

## 2. Config Vet — Per-Check Styled Output

- [x] 2.1 Update `runConfigVet` in `internal/cmd/config_vet.go` to print `FormatVetCheck("Config file found", configPath)` after check 1 passes
- [x] 2.2 Print `FormatVetCheck("Module metadata found", moduleFile)` after check 2 passes
- [x] 2.3 Replace final `output.Println("Configuration is valid: ...")` with `FormatVetCheck("CUE evaluation passed", "")` after checks 3-4 pass
- [x] 2.4 Update `config_vet_test.go` to verify exit code 0 on valid config and exit code behavior on failures

## 3. Verbose Output — FormatResourceLine with StatusValid

- [x] 3.1 Update `writeVerboseHuman` in `internal/output/verbose.go` to render resources using `FormatResourceLine(kind, ns, name, StatusValid)` instead of plain `fmt.Sprintf`
- [x] 3.2 Update verbose output tests to verify resource lines use `FormatResourceLine` format

## 4. Mod Vet Command — New Command

- [x] 4.1 Create `internal/cmd/mod_vet.go` with `NewModVetCmd()` returning a cobra command — `Use: "vet [path]"`, `Args: cobra.MaximumNArgs(1)`, `RunE: runVet`
- [x] 4.2 Add flags: `--values`/`-f`, `--namespace`/`-n`, `--name`, `--provider`, `--strict`, `--verbose`/`-v`
- [x] 4.3 Implement `runVet` — load config, build `RenderOptions`, create pipeline, call `Render()`
- [x] 4.4 On fatal error from `Render()`, call `printValidationError` and return `ExitError{Code: 2}`
- [x] 4.5 On render errors in result, call `printRenderErrors` and return `ExitError{Code: 2}`
- [x] 4.6 On success, iterate `result.Resources` and print each with `FormatResourceLine(kind, ns, name, "valid")`
- [x] 4.7 Print final summary: `FormatCheckmark("Module valid (<N> resources)")`
- [x] 4.8 Handle `--verbose` flag — call `writeVerboseOutput(result, false)` before resource lines
- [x] 4.9 Handle warnings — print via `modLog.Warn` (same pattern as `mod build`)
- [x] 4.10 Register `mod vet` subcommand in `internal/cmd/mod.go` (or wherever `mod` subcommands are registered)

## 5. Mod Vet Tests

- [x] 5.1 Create `internal/cmd/mod_vet_test.go` with table-driven tests
- [x] 5.2 Test: valid module exits with code 0
- [x] 5.3 Test: module with CUE errors exits with code 2
- [x] 5.4 Test: module with unmatched components exits with code 2
- [x] 5.5 Test: `--strict` flag causes exit code 2 on unhandled traits

## 6. Validation Gates

- [x] 6.1 Run `task fmt` — all Go files formatted
- [x] 6.2 Run `task test` — all tests pass
- [x] 6.3 Run `task build` — binary builds successfully
- [x] 6.4 Manual smoke test: `./bin/opm mod vet` on a test fixture module

## 7. Enhancement: Values Validation Visibility

- [x] 7.1 Add values validation check line to `mod vet` output showing which values files were validated
- [x] 7.2 Handle both cases: external `--values` files and module's `values.cue`
- [x] 7.3 Display basenames for external values files (comma-separated if multiple)
- [x] 7.4 Add unit test verifying values detail logic for all cases
