## 1. Create `internal/transformer/` package

- [ ] 1.1 Create `internal/transformer/warnings.go` with package declaration `package transformer`
- [ ] 1.2 Implement `CollectWarnings(plan *core.TransformerMatchPlan) []string`, copied verbatim from `collectWarnings()` in `internal/legacy/build/pipeline.go`

## 2. Test `CollectWarnings`

- [ ] 2.1 Create `internal/transformer/warnings_test.go` with table-driven tests covering:
  - Trait handled by at least one matched transformer → not warned
  - Trait unhandled by all matched transformers → warned
  - Component with no matched transformers → no warnings emitted
  - Multiple components, mixed handled/unhandled traits → only truly unhandled traits warned

## 3. Validation

- [ ] 3.1 Run `task fmt` — verify no formatting issues
- [ ] 3.2 Run `task test` — verify all tests pass including new `internal/transformer` tests
