## 1. Command scaffold

- [x] 1.1 Create `internal/cmd/module/apply.go` with `NewModuleApplyCmd(cfg *config.GlobalConfig) *cobra.Command`, modelled on `internal/cmd/release/apply.go`. Wire the cobra command, flag struct(s), `Use`/`Short`/`Long`/`Example` strings, `Args: cobra.MaximumNArgs(1)`, and a `RunE` that delegates to a `runModuleApply` function.
- [x] 1.2 Register flags: `-f`/`--values` (repeatable), `--provider`, `--name`, `-n`/`--namespace`, `--kubeconfig`, `--context`, `--dry-run`, `--create-namespace`, `--no-prune`, `--force`. Reuse `cmdutil.RenderFlags` and `cmdutil.K8sFlags` where they fit; add a local `--name` flag (matching `internal/cmd/module/build.go`).
- [x] 1.3 Register the new command on `NewModuleCmd` in `internal/cmd/module/mod.go` (`c.AddCommand(NewModuleApplyCmd(cfg))`).

## 2. Runner implementation

- [x] 2.1 In `runModuleApply`, validate the positional argument: reject file inputs with the same error shape as `runModuleBuild` (point users to `opm release apply <file>`). Resolve module path via `cmdutil.ResolveModulePath`.
- [x] 2.2 Call `config.ResolveKubernetes(...)` to produce `*config.ResolvedKubernetesConfig` from kubeconfig/context/namespace/provider flags. Return an exit error if resolution fails.
- [x] 2.3 Call `render.FromModule` with `ModuleOpts{ModulePath, ValuesFiles, Name, K8sConfig, Config}` to produce `*render.Result`. Surface validation errors via existing `printValidationError`/`opmexit.ExitError{Printed: true}` pattern.
- [x] 2.4 Show render banner via `render.ShowOutput(result, render.ShowOutputOpts{Verbose: cfg.Flags.Verbose})`.
- [x] 2.5 Build the Kubernetes client (`cmdutil.NewK8sClient(k8sConfig, cfg.Log.Kubernetes.APIWarnings)`) and a release-scoped logger (`output.ReleaseLogger(result.Release.Name)`).
- [x] 2.6 Call `workflowapply.Execute` with the same `Request` shape used in `release apply`: pass `Result`, `K8sClient`, `Log`, `Options{DryRun, CreateNS, NoPrune, Force, SuccessUpToDateMessage: "Release up to date", SuccessAppliedMessage: "Release applied"}`, `ChangeDescriptor{Path: <absolute module dir>, ValuesStr: "", Version: result.Module.Version, Local: true}`, and `ModuleName`/`ModuleUUID` from `result.Module`.
- [x] 2.7 Resolve module path to an absolute path before passing as `ChangeDescriptor.Path` (use `filepath.Abs`).

## 3. Help text and UX

- [x] 3.1 Help text MUST explicitly note: synth release defaults to `<module>-debug`; `--name` and `--namespace` participate in release identity (different values = different releases); for persistent deploys, author a `release.cue` and use `opm release apply`; when switching surfaces, delete the synthetic release first to avoid orphan inventory.
- [x] 3.2 Include at least three usage examples in `Long`: default invocation, `--name` override, dry-run.

## 4. Unit tests

- [x] 4.1 Create `internal/cmd/module/apply_test.go` covering: flag wiring (all flags registered with expected types/defaults), file-argument rejection error path, missing-debugValues error path, and that `--dry-run` propagates.
- [x] 4.2 Add a `apply_test.go` table-driven test that exercises invalid flag combinations and returns the expected exit codes (validation error vs general error).

## 5. Integration test

- [x] 5.1 Add or extend an integration program under `tests/integration/` that runs `opm module apply` against the local `kind-opm-dev` cluster using one of the example modules in `examples/` (or `tests/testdata/`). Verify: resources are applied, inventory Secret is created with the synthetic UUID, re-running marks resources unchanged, dry-run leaves no inventory.
- [x] 5.2 Verify the prune path: modify the rendered set (drop one component or remove one resource) and re-apply; assert the stale resource is pruned and inventory revision increments.

## 6. Validation gates

- [x] 6.1 Run `task fmt` and ensure no formatting drift.
- [x] 6.2 Run `task vet`.
- [x] 6.3 Run `task lint` — fix any new lint findings (watch `gocyclo` on `runModuleApply` — keep it under threshold by extracting helpers if needed).
- [x] 6.4 Run `task test:unit` and confirm all new tests pass.
- [ ] 6.5 Run `task test:integration` against `kind-opm-dev`. (Deferred to user — agent may not mutate live cluster.)
- [x] 6.6 Run `openspec validate module-apply --strict` and resolve any spec-validation errors.
