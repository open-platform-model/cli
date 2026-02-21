## 1. Verify Current State

- [ ] 1.1 Confirm all files in `internal/build/` match the design doc inventory (pipeline.go, types.go, errors.go, module/, release/, transform/, testdata/)
- [ ] 1.2 Grep for any direct imports of `internal/build` subpackages beyond the 6 known files (`internal/build/module`, `internal/build/release`, `internal/build/transform`)

## 2. Move Directory

- [ ] 2.1 Run `git mv internal/build internal/legacy` to rename the directory while preserving file history

## 3. Update Import Paths

- [ ] 3.1 Update `internal/cmdutil/render.go` — replace `internal/build` → `internal/legacy`
- [ ] 3.2 Update `internal/cmdutil/render_test.go` — replace `internal/build` → `internal/legacy`
- [ ] 3.3 Update `internal/cmdutil/output.go` — replace `internal/build` → `internal/legacy`
- [ ] 3.4 Update `internal/cmdutil/output_test.go` — replace `internal/build` → `internal/legacy`
- [ ] 3.5 Update `internal/cmd/mod/verbose_output_test.go` — replace `internal/build` → `internal/legacy`
- [ ] 3.6 Update `experiments/module-full-load/single_load_test.go` — replace `internal/build` → `internal/legacy`
- [ ] 3.7 Update any subpackage imports found in step 1.2 (`internal/build/module`, `internal/build/release`, `internal/build/transform`)

## 4. Validate

- [ ] 4.1 Run `task build` — confirm binary compiles with no errors
- [ ] 4.2 Run `task test` — confirm all tests pass
- [ ] 4.3 Run `task check` — confirm fmt + vet + test all pass

## 5. Commit

- [ ] 5.1 Stage all changes (`git mv` output + 6 import path updates) and commit atomically: `refactor(build): move internal/build to internal/legacy`
