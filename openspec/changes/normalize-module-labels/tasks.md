## 1. Label normalization (internal/kubernetes/apply.go)

- [x] 1.1 Add `strings.ToLower(meta.Name)` in `injectLabels()` at line 99 for the `LabelModuleName` label value
- [x] 1.2 Add `"strings"` to the import block if not already present

## 2. Name override normalization (internal/build/module.go)

- [x] 2.1 In `applyOverrides()`, lowercase `opts.Name` with `strings.ToLower()` before assigning to `module.Name`
- [x] 2.2 Add `"strings"` to the import block if not already present

## 3. Delete not-found error (internal/kubernetes/delete.go)

- [x] 3.1 Replace the `nil` return in `Delete()` when `len(resources) == 0` with a `NewNotFoundError()` that includes module name, namespace, and a hint about case-sensitivity
- [x] 3.2 Add import for `github.com/opmodel/cli/internal/errors`

## 4. Delete command error handling (internal/cmd/mod_delete.go)

- [x] 4.1 In `runDelete()`, handle the `ErrNotFound` error from `kubernetes.Delete()` — use `ExitCodeFromError()` to map it to exit code 5

## 5. Tests

- [x] 5.1 Add table-driven unit tests for `injectLabels()` in `internal/kubernetes/apply_test.go`: PascalCase name → lowercase label, lowercase name → unchanged, hyphenated name → unchanged
- [x] 5.2 Add table-driven unit tests for `applyOverrides()` in `internal/build/module_test.go`: uppercase override → lowercased, lowercase override → unchanged
- [x] 5.3 Add unit test for `Delete()` in `internal/kubernetes/delete_test.go`: verify `ErrNotFound` is returned when no resources are discovered
- [x] 5.4 Update any existing tests that assert on the `module.opmodel.dev/name` label value to expect lowercase

## 6. Test fixture update

- [ ] 6.1 Update `testing/blog/module.cue` to use lowercase name — BLOCKED: requires catalog change `k8s-name-constraints` to update `#FQNType` first

## 7. Validation gates

- [x] 7.1 Run `task fmt` — all Go files formatted
- [x] 7.2 Run `task test` — all tests pass
- [x] 7.3 (user will run manually) Manual smoke test: apply `testing/blog`, verify label is lowercase, delete with lowercase name, verify exit 0. Delete with wrong case, verify exit 5.
