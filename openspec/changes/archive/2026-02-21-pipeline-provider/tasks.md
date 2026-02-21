## 1. Types

- [x] 1.1 Create `internal/provider/types.go` defining `LoadedProvider` struct with `Name string`, `Transformers []*core.Transformer`, and `Requirements() []string` method
- [x] 1.2 Confirm no `LoadedTransformer` type is needed (all criteria already on `*core.Transformer`)

## 2. Core Implementation

- [x] 2.1 Create `internal/provider/provider.go` with `Load(cueCtx *cue.Context, name string, providers map[string]cue.Value) (*LoadedProvider, error)`
- [x] 2.2 Implement auto-select: if `name` is empty and `len(providers) == 1`, select the single provider automatically
- [x] 2.3 Implement error for empty name with multiple providers: `"provider name must be specified (available: [...])"`
- [x] 2.4 Implement error for provider not found: `"provider %q not found (available: [...])"`
- [x] 2.5 Implement transformer iteration: walk `providerValue.LookupPath("transformers").Fields()` and call `extractCoreTransformer` per entry
- [x] 2.6 Return error if provider has zero transformers after iteration
- [x] 2.7 Copy CUE extraction helpers from legacy (`extractLabelsField`, `extractMapKeys`, `extractCueValueMap`) into the new package

## 3. Tests

- [x] 3.1 Test: named provider found — returns `*LoadedProvider` with correct transformer count
- [x] 3.2 Test: provider name not found — returns error listing available providers
- [x] 3.3 Test: auto-select when exactly one provider configured — succeeds without specifying name
- [x] 3.4 Test: empty name with multiple providers — returns error
- [x] 3.5 Test: transformer FQN construction — `kubernetes#deployment` format
- [x] 3.6 Test: transformer with required and optional criteria — all fields populated on `*core.Transformer`
- [x] 3.7 Test: transformer with no optional criteria — empty slices/maps, no error
- [x] 3.8 Test: provider with no transformers — returns error
- [x] 3.9 Test: `Requirements()` returns FQN slice matching loaded transformers

## 4. Validation

- [x] 4.1 Run `task fmt` — all files formatted
- [x] 4.2 Run `task test` — all tests pass
