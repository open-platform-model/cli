## 1. Output Package — StatusValid and FormatVetCheck

- [ ] 1.1 Add `StatusValid = "valid"` constant to `internal/output/styles.go`
- [ ] 1.2 Add `"valid"` case to `statusStyle()` returning green (`82`) foreground
- [ ] 1.3 Add `FormatVetCheck(label, detail string) string` function to `internal/output/styles.go` — green checkmark, label, optional right-aligned dim detail at column 34
- [ ] 1.4 Add unit tests for `StatusValid` in `statusStyle()` — verify green style, verify same color as `StatusCreated`
- [ ] 1.5 Add unit tests for `FormatVetCheck` — with detail, without detail, alignment consistency across multiple calls

## 2. Config Vet — Per-Check Styled Output

- [ ] 2.1 Update `runConfigVet` in `internal/cmd/config_vet.go` to print `FormatVetCheck("Config file found", configPath)` after check 1 passes
- [ ] 2.2 Print `FormatVetCheck("Module metadata found", moduleFile)` after check 2 passes
- [ ] 2.3 Replace final `output.Println("Configuration is valid: ...")` with `FormatVetCheck("CUE evaluation passed", "")` after checks 3-4 pass
- [ ] 2.4 Update `config_vet_test.go` to verify exit code 0 on valid config and exit code behavior on failures

## 3. Verbose Output — FormatResourceLine with StatusValid

- [ ] 3.1 Update `writeVerboseHuman` in `internal/output/verbose.go` to render resources using `FormatResourceLine(kind, ns, name, StatusValid)` instead of plain `fmt.Sprintf`
- [ ] 3.2 Update verbose output tests to verify resource lines use `FormatResourceLine` format

## 4. Mod Vet Command — New Command

- [ ] 4.1 Create `internal/cmd/mod_vet.go` with `NewModVetCmd()` returning a cobra command — `Use: "vet [path]"`, `Args: cobra.MaximumNArgs(1)`, `RunE: runVet`
- [ ] 4.2 Add flags: `--values`/`-f`, `--namespace`/`-n`, `--name`, `--provider`, `--strict`, `--verbose`/`-v`
- [ ] 4.3 Implement `runVet` — load config, build `RenderOptions`, create pipeline, call `Render()`
- [ ] 4.4 On fatal error from `Render()`, call `printValidationError` and return `ExitError{Code: 2}`
- [ ] 4.5 On render errors in result, call `printRenderErrors` and return `ExitError{Code: 2}`
- [ ] 4.6 On success, iterate `result.Resources` and print each with `FormatResourceLine(kind, ns, name, "valid")`
- [ ] 4.7 Print final summary: `FormatCheckmark("Module valid (<N> resources)")`
- [ ] 4.8 Handle `--verbose` flag — call `writeVerboseOutput(result, false)` before resource lines
- [ ] 4.9 Handle warnings — print via `modLog.Warn` (same pattern as `mod build`)
- [ ] 4.10 Register `mod vet` subcommand in `internal/cmd/mod.go` (or wherever `mod` subcommands are registered)

## 5. Mod Vet Tests

- [ ] 5.1 Create `internal/cmd/mod_vet_test.go` with table-driven tests
- [ ] 5.2 Test: valid module exits with code 0
- [ ] 5.3 Test: module with CUE errors exits with code 2
- [ ] 5.4 Test: module with unmatched components exits with code 2
- [ ] 5.5 Test: `--strict` flag causes exit code 2 on unhandled traits

## 6. Validation Gates

- [ ] 6.1 Run `task fmt` — all Go files formatted
- [ ] 6.2 Run `task test` — all tests pass
- [ ] 6.3 Run `task build` — binary builds successfully
- [ ] 6.4 Manual smoke test: `./bin/opm mod vet` on a test fixture module
