## 1. Add loader.LoadProvider()

- [x] 1.1 Create `internal/loader/provider.go` with `LoadProvider(cueCtx, name, providers)` returning `*core.Provider`
- [x] 1.2 Implement `extractProviderMetadata()` — extract `name`, `description`, `version`, `minVersion`, `labels` from `metadata.*` and `apiVersion`/`kind` from root; use config key as `Metadata.Name` fallback
- [x] 1.3 Move `parseTransformers()`, `extractTransformer()`, `buildFQN()`, `extractLabelsField()`, `extractCueValueMap()`, `sortedKeys()` from `internal/provider/provider.go` into `internal/loader/provider.go`
- [x] 1.4 Ensure `LoadProvider()` sets `CueCtx` on the returned `*core.Provider` and builds `Transformers` as `map[string]*core.Transformer` (keyed by transformer name)

## 2. Migrate tests

- [x] 2.1 Create `internal/loader/provider_test.go` — port all tests from `internal/provider/provider_test.go`
- [x] 2.2 Update test assertions: `lp.Name` → `lp.Metadata.Name`, `lp.Transformers[0]` → `lp.Transformers["<name>"]` (slice access → map lookup)
- [x] 2.3 Add tests for new metadata extraction: verify `Metadata.Version`, `Metadata.Description`, `Metadata.Labels` are populated when present in CUE value
- [x] 2.4 Add test: config key used as `Metadata.Name` fallback when `metadata.name` absent in CUE value

## 3. Update pipeline

- [x] 3.1 In `internal/pipeline/pipeline.go`, replace `"github.com/opmodel/cli/internal/provider"` import with `"github.com/opmodel/cli/internal/loader"`
- [x] 3.2 Replace `provider.Load(...)` call with `loader.LoadProvider(...)`
- [x] 3.3 Remove the 6-line transformer slice→map conversion bridge (lines that build `transformerMap` and construct `coreProvider` manually)
- [x] 3.4 Update Phase 2/4 comments in `Render()` to reflect the new flow

## 4. Delete internal/provider

- [x] 4.1 Delete `internal/provider/provider.go`
- [x] 4.2 Delete `internal/provider/types.go`
- [x] 4.3 Delete `internal/provider/provider_test.go`
- [x] 4.4 Remove the (now empty) `internal/provider/` directory

## 5. Validation

- [x] 5.1 Run `task fmt` — all files formatted
- [x] 5.2 Run `task lint` — no lint errors
- [x] 5.3 Run `task test:unit` — all unit tests pass
