## MODIFIED Requirements

### Requirement: RenderModule executes the common render pipeline and returns a result or ExitError

The `RenderModule` function SHALL accept a context, a `RenderModuleOpts` struct (containing args, RenderFlags, optional K8sFlags, `*config.GlobalConfig`, and verbose flag), and SHALL execute the full render pipeline preamble: resolve module path from args, validate config is loaded, resolve Kubernetes config (namespace/provider) via `config.ResolveKubernetes`, build `build.RenderOptions`, validate options, create `build.NewPipeline`, and call `pipeline.Render()`.

The `RenderModuleOpts` struct SHALL contain a `Config *config.GlobalConfig` field. It SHALL NOT contain separate `OPMConfig` and `Registry` fields. The pipeline SHALL read the registry from `Config.Registry`.

On success, it SHALL return the `*build.RenderResult`. On failure at any step, it SHALL return an `*ExitError` with the appropriate exit code.

#### Scenario: RenderModule succeeds with valid module

- **WHEN** `RenderModule` is called with a valid module path, loaded GlobalConfig, and valid RenderFlags
- **THEN** it SHALL return a non-nil `*build.RenderResult`
- **THEN** the error SHALL be `nil`

#### Scenario: RenderModule fails when Config is nil

- **WHEN** `RenderModule` is called with a nil Config
- **THEN** it SHALL return an `*ExitError` with `Code` equal to `ExitGeneralError`
- **THEN** the error message SHALL contain `"configuration not loaded"`

#### Scenario: RenderModule fails on render validation error

- **WHEN** `pipeline.Render()` returns a `*build.ReleaseValidationError`
- **THEN** `RenderModule` SHALL call the validation error formatter before returning
- **THEN** it SHALL return an `*ExitError` with `Code` equal to `ExitValidationError` and `Printed` set to `true`

#### Scenario: RenderModule defaults module path to current directory

- **WHEN** `RenderModule` is called with an empty args slice
- **THEN** the module path used for `build.RenderOptions.ModulePath` SHALL be `"."`

#### Scenario: RenderModule uses first arg as module path

- **WHEN** `RenderModule` is called with args `["./my-module"]`
- **THEN** the module path used for `build.RenderOptions.ModulePath` SHALL be `"./my-module"`

### Requirement: NewK8sClient creates a Kubernetes client or returns a connectivity ExitError

The `NewK8sClient` function SHALL accept a `*config.ResolvedKubernetesConfig` and an API warnings string. It SHALL create a `kubernetes.Client` using `kubernetes.NewClient`. On failure, it SHALL return an `*ExitError` with `Code` equal to `ExitConnectivityError`.

#### Scenario: NewK8sClient succeeds with valid kubeconfig

- **WHEN** `NewK8sClient` is called with a valid kubeconfig and context
- **THEN** it SHALL return a non-nil `*kubernetes.Client`
- **THEN** the error SHALL be `nil`

#### Scenario: NewK8sClient returns connectivity error on failure

- **WHEN** `NewK8sClient` is called and `kubernetes.NewClient` returns an error
- **THEN** it SHALL return an `*ExitError` with `Code` equal to `ExitConnectivityError`

## REMOVED Requirements

### Requirement: cmdutil ResolveKubernetes wrapper

**Reason**: Replaced by callers using `config.ResolveKubernetes` directly with `*config.GlobalConfig` in the options struct. The wrapper existed solely to unpack `*config.OPMConfig` into `*Config` + `[]string` provider names.

**Migration**: All call sites change from `cmdutil.ResolveKubernetes(cfg.OPMConfig, ...)` to `config.ResolveKubernetes(config.ResolveKubernetesOptions{Config: cfg, ...})`.
