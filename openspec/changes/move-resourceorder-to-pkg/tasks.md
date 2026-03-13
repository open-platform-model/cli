## 1. Move resourceorder to pkg

- [ ] 1.1 Create `pkg/resourceorder/weights.go` from `internal/resourceorder/weights.go`: change package declaration to `resourceorder` (same name, new location)
- [ ] 1.2 Create `pkg/resourceorder/weights_test.go` from `internal/resourceorder/weights_test.go`: change package declaration

## 2. Update internal callers

- [ ] 2.1 Update `internal/inventory/stale.go`: change import from `internal/resourceorder` to `pkg/resourceorder`
- [ ] 2.2 Update `internal/inventory/digest.go`: change import from `internal/resourceorder` to `pkg/resourceorder`
- [ ] 2.3 Update `internal/output/manifest.go`: change import from `internal/resourceorder` to `pkg/resourceorder`
- [ ] 2.4 Update `internal/kubernetes/delete.go`: change import from `internal/resourceorder` to `pkg/resourceorder`

## 3. Remove old package

- [ ] 3.1 Delete `internal/resourceorder/` directory

## 4. Validation

- [ ] 4.1 Run `task build` — confirm compilation succeeds
- [ ] 4.2 Run `task test` — confirm all tests pass
- [ ] 4.3 Run `task lint` — confirm linter passes

## 5. Commits

- [ ] 5.1 Commit tasks 1.1–1.2, 2.1–2.4, 3.1: `refactor(resourceorder): move resourceorder from internal to pkg`
