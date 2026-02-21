## 1. Types

- [ ] 1.1 Create `internal/provider/types.go` defining `LoadedProvider` struct with `Name string`, `Transformers []*core.Transformer`, and `Requirements() []string` method
- [ ] 1.2 Confirm no `LoadedTransformer` type is needed (all criteria already on `*core.Transformer`)

## 2. Core Implementation

- [ ] 2.1 Create `internal/provider/provider.go` with `Load(cueCtx *cue.Context, name string, providers map[string]cue.Value) (*LoadedProvider, error)`
- [ ] 2.2 Implement auto-select: if `name` is empty and `len(providers) == 1`, select the single provider automatically
- [ ] 2.3 Implement error for empty name with multiple providers: `"provider name must be specified (available: [...])"`
- [ ] 2.4 Implement error for provider not found: `"provider %q not found (available: [...])"`
- [ ] 2.5 Implement transformer iteration: walk `providerValue.LookupPath("transformers").Fields()` and call `extractCoreTransformer` per entry
- [ ] 2.6 Return error if provider has zero transformers after iteration
- [ ] 2.7 Copy CUE extraction helpers from legacy (`extractLabelsField`, `extractMapKeys`, `extractCueValueMap`) into the new package

## 3. Tests

- [ ] 3.1 Test: named provider found — returns `*LoadedProvider` with correct transformer count
- [ ] 3.2 Test: provider name not found — returns error listing available providers
- [ ] 3.3 Test: auto-select when exactly one provider configured — succeeds without specifying name
- [ ] 3.4 Test: empty name with multiple providers — returns error
- [ ] 3.5 Test: transformer FQN construction — `kubernetes#deployment` format
- [ ] 3.6 Test: transformer with required and optional criteria — all fields populated on `*core.Transformer`
- [ ] 3.7 Test: transformer with no optional criteria — empty slices/maps, no error
- [ ] 3.8 Test: provider with no transformers — returns error
- [ ] 3.9 Test: `Requirements()` returns FQN slice matching loaded transformers

## 4. Validation

- [ ] 4.1 Run `task fmt` — all files formatted
- [ ] 4.2 Run `task test` — all tests pass
