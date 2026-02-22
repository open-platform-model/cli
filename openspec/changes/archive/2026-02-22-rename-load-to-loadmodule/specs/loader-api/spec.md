## MODIFIED Requirements

### Requirement: Module loader function name
The `internal/loader` package SHALL export a function named `LoadModule` (not `Load`) for loading a CUE module into a `*core.Module`. The function signature SHALL remain identical: `func LoadModule(cueCtx *cue.Context, modulePath, registry string) (*core.Module, error)`.

#### Scenario: Call site compiles with new name
- **WHEN** any package calls `loader.LoadModule(ctx, path, registry)`
- **THEN** it compiles and behaves identically to the previous `loader.Load(ctx, path, registry)`

#### Scenario: Old name is not exported
- **WHEN** code references `loader.Load(...)`
- **THEN** the Go compiler reports an undefined symbol error
