## 1. Bug Fix — kindToResource

- [ ] 1.1 In `internal/kubernetes/resource.go`, export `kindToResource` → `KindToResource` and `heuristicPluralize` → `HeuristicPluralize` (capitalize first letter of each). Update the internal call from `kindToResource` to `KindToResource`.
- [ ] 1.2 In `internal/inventory/stale.go`, delete the local `kindToResource` function (lines 178–199) and replace its call site(s) with `kubernetes.KindToResource`.
- [ ] 1.3 Check `internal/inventory/discover.go` for any call to the local `kindToResource`; if present, replace with `kubernetes.KindToResource`.
- [ ] 1.4 Run `task test:unit` — confirm all inventory and kubernetes tests pass.

## 2. Dead Code — Delete Unused Functions

- [ ] 2.1 Delete `ComputeReleaseUUID` from `internal/core/labels.go` (lines 16–21) and its `import "github.com/google/uuid"`. Delete the corresponding test in `internal/core/labels_test.go`. Run `go mod tidy` to remove the dependency.
- [ ] 2.2 Delete `ValidFormats` from `internal/output/format.go`. Remove the test(s) in `internal/output/format_test.go` that call `ValidFormats`.
- [ ] 2.3 Delete `ListTemplateFiles` from `internal/templates/embed.go`. Remove the test(s) in `internal/templates/embed_test.go` that call it.
- [ ] 2.4 Move `SetLogWriter` out of `internal/output/log.go` into `internal/output/export_test.go` (create file if it does not exist). The function touches the unexported `logger` var — keep it package-private in the test file. Verify all test files that call `SetLogWriter` still compile.
- [ ] 2.5 Move `rewriteErrorPath` from `internal/core/modulerelease/validation.go` (lines 111–122) into `internal/core/modulerelease/validation_test.go`. Verify the two test call sites still compile.
- [ ] 2.6 Delete the dead no-op block in `internal/builder/builder.go` (the `if v := result.LookupPath(...metadata.version...)` block that ends with `_ = s`). Verify `task build` still compiles.
- [ ] 2.7 In `internal/version/version.go` — `Info.String()` already exists and is well-formed. In `internal/cmd/version.go` `runVersion`, replace the manual multi-line field formatting with a single `output.Println(info.String())` call. Remove now-unused `fmt.Sprintf` calls.
- [ ] 2.8 Delete `DeleteInventory` from `internal/inventory/crud.go` (confirm it has zero production callers first with a grep). Add a comment at the `mod delete` call site explaining that the inventory Secret is tracked as a managed resource and is therefore deleted via the standard resource deletion path. Remove corresponding tests from `crud_test.go` that test `DeleteInventory` in isolation.

## 3. Dead Code — Remove Struct Fields

- [ ] 3.1 Remove `Annotations map[string]string` field and its TODO comment from `internal/core/module/module.go` (`ModuleMetadata.Annotations`). Verify no code sets or reads this field (grep for `ModuleMetadata` and `.Annotations`).
- [ ] 3.2 Remove `Annotations map[string]string` field and its TODO comment from `internal/core/modulerelease/release.go` (`ReleaseMetadata.Annotations`). Verify no code sets or reads this field.
- [ ] 3.3 Remove `Labels map[string]string` field and its TODO comment from `internal/core/module/module.go` (`ModuleMetadata.Labels`). Find and remove the setter in `internal/loader/module.go` that populates it. Verify nothing downstream reads `ModuleMetadata.Labels`.

## 4. Dead Code — Remove Non-Functional CLI Flags

- [ ] 4.1 In `internal/cmd/mod/apply.go`: remove `waitFlag` and `timeoutFlag` variable declarations. Remove `c.Flags().BoolVar(&waitFlag, "wait", ...)` and `c.Flags().DurationVar(&timeoutFlag, "timeout", ...)` registrations. Remove `wait bool, timeout time.Duration` from `runApply` function signature. Remove `waitFlag, timeoutFlag` from the `runApply(...)` call in `RunE`. Update tests in `apply_test.go` if they reference these flags.
- [ ] 4.2 In `internal/cmd/mod/delete.go`: remove `waitFlag` variable declaration. Remove `c.Flags().BoolVar(&waitFlag, "wait", ...)` registration. Remove the `_ /* wait */` blank parameter from `runDelete` function signature. Remove `waitFlag` from the `runDelete(...)` call in `RunE`. Update tests if needed.

## 5. Trivial DRY — Single-File Fixes

- [ ] 5.1 In `internal/kubernetes/apply.go`, delete the unexported `boolPtr` function and replace its call site(s) with `output.BoolPtr`. Add the `output` import if not already present.
- [ ] 5.2 In `internal/output/styles.go`, extract `lipgloss.NewStyle().Foreground(colorGreenCheck).Render("✔")` into a package-level `var styledGreenCheck = ...`. Replace the two inline constructions in `FormatCheckmark` (line ~136) and `FormatVetCheck` (line ~211) with `styledGreenCheck`.
- [ ] 5.3 In `internal/cmd/mod/build.go` and `internal/cmd/mod/vet.go`, change `"resolving config: %w"` to `"resolving kubernetes config: %w"` to match the message used in `apply.go`, `delete.go`, `diff.go`, and `status.go`.
- [ ] 5.4 In `internal/cmdutil/flags_test.go`, remove the duplicate `ResolveModulePath` assertions that are already covered in `internal/cmdutil/render_test.go`.
- [ ] 5.5 Delete the 4 orphaned doc comments in `internal/output/manifest.go` (before lines 99 and 119) and `internal/output/split.go` (before lines 72 and 101).

## 6. DRY — Fix Core Method Usage

- [ ] 6.1 In `internal/core/provider/provider.go`, function `buildMatchReason` (lines ~206–221): replace the 2 inline sorted-key-extraction blocks for `tf.RequiredResources` and `tf.RequiredTraits` with calls to `tf.GetRequiredResources()` and `tf.GetRequiredTraits()`. Remove the now-unused `sort` import if applicable.
- [ ] 6.2 In `internal/core/transformer/execute.go` (lines ~41–48): replace the inline nil-guard block that extracts `tfFQN` with a call to `match.Transformer.GetFQN()`. Remove the now-dead nil-check lines.

## 7. DRY — Fix apply.go Module Path

- [ ] 7.1 Read `internal/inventory/changeid.go` to confirm how `ComputeChangeID` uses the module path and whether `""` vs `"."` produces different output.
- [ ] 7.2 If `""` and `"."` are equivalent in `ComputeChangeID`: replace the inline args extraction in `internal/cmd/mod/apply.go` (lines ~182–184) with `modulePath := cmdutil.ResolveModulePath(args)`.
- [ ] 7.3 If they differ: add a comment documenting the intentional `""` default in `apply.go` and leave as-is.

## 8. DRY — Extract CUE String-Map Helper

- [ ] 8.1 Create `internal/loader/cue_util.go` with an unexported `extractCUEStringMap(v cue.Value, field string) (map[string]string, error)` function matching the spec in `specs/loader-cue-helpers/spec.md`.
- [ ] 8.2 In `internal/loader/module.go`, replace the inline CUE label-extraction loop with a call to `extractCUEStringMap`.
- [ ] 8.3 In `internal/loader/provider.go`, replace the inline CUE label-extraction loop (the `metadata.labels` one, not the `extractLabelsField` helper) with a call to `extractCUEStringMap`.
- [ ] 8.4 In `internal/builder/builder.go`, replace the inline CUE label-extraction loop with a call to `extractCUEStringMap` (this may require importing the `loader` package or moving the helper to a shared location — if a circular import results, move `cue_util.go` to `internal/cmdutil/cueutil.go` or inline an unexported copy in `builder`).
- [ ] 8.5 Run `task test:unit` to confirm loader and builder tests pass.

## 9. DRY — Consolidate Resource Sort

- [ ] 9.1 Check the import graph: does `internal/pipeline` import `internal/inventory` or vice versa? Run `go list -f '{{.Imports}}' github.com/opmodel/cli/internal/pipeline` and `go list -f '{{.Imports}}' github.com/opmodel/cli/internal/inventory`.
- [ ] 9.2 **If no circular import**: in `internal/pipeline/pipeline.go`, delete the unexported `sortResources` function and replace its call site with `inventory.SortResources`. Add the `inventory` import.
- [ ] 9.3 **If circular import**: move `SortResources` from `internal/inventory/digest.go` to `internal/core/resource.go` (rename to `SortResources`, keeping it exported). Update `internal/inventory/digest.go` to call `core.SortResources`. Update `internal/pipeline/pipeline.go` to call `core.SortResources` (delete local `sortResources`).
- [ ] 9.4 Decide on `sortResourceInfos` in `internal/output/manifest.go` (currently a 3-key variant): either add group+kind sort keys to align it with the 5-key sort, or add an explicit comment: `// Intentional 3-key sort for display purposes; does not need to match apply order.`
- [ ] 9.5 Run `task test:unit` to confirm pipeline and inventory tests pass.

## 10. DRY — cmdutil.ResolveInventory Helper

- [ ] 10.1 In `internal/cmdutil/`, create a new file (e.g., `inventory.go`) with the `ResolveInventory` function matching the spec in `specs/cmdutil-inventory-resolver/spec.md`. Signature: `func ResolveInventory(ctx context.Context, client *kubernetes.Client, rsf *ReleaseSelectorFlags, namespace string, ignoreNotFound bool, log *log.Logger) (*inventory.InventorySecret, []*core.Resource, error)`.
- [ ] 10.2 In `internal/cmd/mod/delete.go`, replace the inventory-resolution switch block (lines ~131–165) with a call to `cmdutil.ResolveInventory`. Remove the now-redundant local variable declarations.
- [ ] 10.3 In `internal/cmd/mod/status.go`, replace the equivalent block (lines ~133–166) with a call to `cmdutil.ResolveInventory`.
- [ ] 10.4 Write unit tests for `cmdutil.ResolveInventory` in `internal/cmdutil/inventory_test.go` covering the scenarios in the spec (success by name, success by ID, not-found+ignore, not-found+error, k8s error, discovery error).
- [ ] 10.5 Run `task test:unit` to confirm delete, status, and cmdutil tests pass.

## 11. Structural — Transformer Type Cleanup

- [ ] 11.1 In `internal/core/transformer/transformer.go`, remove `APIVersion string` from `TransformerMetadata` (it duplicates `Transformer.APIVersion`). Find any code that reads `transformer.Metadata.APIVersion` and update it to read `transformer.APIVersion` directly.
- [ ] 11.2 Check all in-progress changes (`transformer-matching-v2`, `mod-tree`) for references to `TransformerComponentMetadata` type to assess risk before proceeding with 11.3.
- [ ] 11.3 In `internal/core/transformer/context.go`, replace the `TransformerComponentMetadata` struct with `component.ComponentMetadata` (import the `component` package). Update `TransformerContext.Components` map type from `map[string]TransformerComponentMetadata` to `map[string]component.ComponentMetadata`. Update `NewTransformerContext` to embed `*comp.Metadata` directly (by dereferencing or copying). Remove the now-unused `TransformerComponentMetadata` type definition.
- [ ] 11.4 Run `task test:unit` to confirm transformer tests pass.

## 12. Validation Gates

- [ ] 12.1 Run `task fmt` — confirm no formatting issues.
- [ ] 12.2 Run `task vet` — confirm no vet issues.
- [ ] 12.3 Run `task lint` — confirm golangci-lint passes.
- [ ] 12.4 Run `task test:unit` — confirm all unit tests pass.
- [ ] 12.5 Run `task build` — confirm binary builds cleanly.
- [ ] 12.6 Run `go mod tidy` and verify `go.mod` / `go.sum` are clean (especially that `github.com/google/uuid` is gone after task 2.1).
