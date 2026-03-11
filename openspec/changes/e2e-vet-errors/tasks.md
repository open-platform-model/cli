## 1. Setup Test Fixtures

- [ ] 1.1 Create `tests/e2e/testdata/vet-errors/module/module.cue` with minimal schema.
- [ ] 1.2 Create `tests/e2e/testdata/vet-errors/release/release.cue` pointing to the module.
- [ ] 1.3 Create `tests/e2e/testdata/vet-errors/release/values.cue` with invalid config values.

## 2. Implement E2E Tests

- [ ] 2.1 Create `tests/e2e/vet_output_test.go`.
- [ ] 2.2 Add test `TestE2E_ReleaseVet_Output` to run `opm rel vet` on the fixture and assert grouped output.
- [ ] 2.3 Add test `TestE2E_ModuleVet_Output` to run `opm mod vet` on the fixture and assert grouped output.

## 3. Validation

- [ ] 3.1 Run `task fmt`
- [ ] 3.2 Run `task lint`
- [ ] 3.3 Run `task test`