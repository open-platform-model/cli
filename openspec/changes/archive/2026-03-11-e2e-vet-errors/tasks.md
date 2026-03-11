## 1. Setup Test Fixtures

- [x] 1.1 Create `tests/e2e/testdata/vet-errors/module/module.cue` with minimal schema.
- [x] 1.2 Create `tests/e2e/testdata/vet-errors/release/release.cue` pointing to the module.
- [x] 1.3 Create `tests/e2e/testdata/vet-errors/release/values.cue` with invalid config values.

## 2. Implement E2E Tests

- [x] 2.1 Create `tests/e2e/vet_output_test.go`.
- [x] 2.2 Add test `TestE2E_ReleaseVet_Output` to run `opm rel vet` on the fixture and assert grouped output.
- [x] 2.3 Add test `TestE2E_ModuleVet_Output` to run `opm mod vet` on the fixture and assert grouped output.

## 3. Validation

- [x] 3.1 Run `task fmt`
- [x] 3.2 Run `task lint`
- [x] 3.3 Run `task test`