## 1. Verify Current State

- [x] 1.1 Confirm all files in `internal/build/` match the design doc inventory (pipeline.go, types.go, errors.go, module/, release/, transform/, testdata/)
- [x] 1.2 Grep for any direct imports of `internal/build` subpackages beyond the 6 known files (`internal/build/module`, `internal/build/release`, `internal/build/transform`)

## 2. Move Directory

- [x] 2.1 Run `git mv internal/build internal/legacy` to rename the directory while preserving file history

## 3. Update Import Paths

- [x] 3.1 Update `internal/cmdutil/render.go` — replace `internal/build` → `internal/legacy`
- [x] 3.2 Update `internal/cmdutil/render_test.go` — replace `internal/build` → `internal/legacy`
- [x] 3.3 Update `internal/cmdutil/output.go` — replace `internal/build` → `internal/legacy`
- [x] 3.4 Update `internal/cmdutil/output_test.go` — replace `internal/build` → `internal/legacy`
- [x] 3.5 Update `internal/cmd/mod/verbose_output_test.go` — replace `internal/build` → `internal/legacy`
- [x] 3.6 Update `experiments/module-full-load/single_load_test.go` — replace `internal/build` → `internal/legacy` (file had no such import — skipped)
- [x] 3.7 Update any subpackage imports found in step 1.2 (`internal/build/module`, `internal/build/release`, `internal/build/transform`)

## 4. Validate

- [x] 4.1 Run `task build` — confirm binary compiles with no errors
- [x] 4.2 Run `task test` — confirm all tests pass
- [x] 4.3 Run `task check` — confirm fmt + vet + test all pass (lint failures are pre-existing baseline, not introduced by this change)

## 5. Commit

- [ ] 5.1 Stage all changes (`git mv` output + 6 import path updates) and commit atomically: `refactor(build): move internal/build to internal/legacy`
