Registry-backed tests in this file run with the canonical registry mapping exported (`CUE_REGISTRY` and `OPM_REGISTRY` set to `testing.opmodel.dev=ghcr.io/open-platform-model,opmodel.dev=ghcr.io/open-platform-model,registry.cue.works`, `Taskfile.yml:18`); without it they skip instead of failing. A test that reports SKIP does not count as passing.

Before each section, merge fresh `origin/main` if it moved. After merging a sibling change that touched `internal/cmd/instance` (order-instance-apply-by-weight, protect-crds-and-namespaces-in-prune-and-delete), re-run `task docs:reference` instead of hand-resolving the generated page.

## 1. Refuse the mismatched namespace

- [x] 1.1 In `internal/workflow/render/render.go`, add `refuseNamespaceOverride(instancePath, declared string, ns config.ResolvedField) error` as in design.md "Where the guard runs": nil unless `ns.Source` is `config.SourceFlag` or `config.SourceEnv` and `ns.Value != declared`; otherwise a `*pkgerrors.ValidationError` whose `Message` names `--namespace` or `OPM_NAMESPACE`, both values and the instance argument, and whose `Details` gives the fix (edit `metadata.namespace` in the instance file). No enhancement or ADR reference in the message.
- [x] 1.2 Call it in `FromInstanceFile` right after `AcquireInstanceFromDir` succeeds and before `instanceDepsOf`/`resolvePlatformEnv`, when `inst.Metadata != nil`; on error `printValidationError` and return `&opmexit.ExitError{Code: opmexit.ExitValidationError, Err: err, Printed: true}`. Leave `renderInstance` and `FromModule` unchanged.
- [x] 1.3 Add a table test `TestRefuseNamespaceOverride` in `internal/workflow/render/render_test.go` (pure, no registry) with rows: flag matches (nil), flag differs (error naming `--namespace` and both values), env differs (error naming `OPM_NAMESPACE` and both values), no override with source config (nil, values differ), no override with source default (nil, values differ), flag matches while `Shadowed[SourceEnv]` differs (nil: the flag wins and the env value is not compared). Assert every error mentions `metadata.namespace`. Add a row through `captureValidationOutput` (`validation_test.go`) asserting the printed form: `render failed: --namespace ...` in the log stream and the `metadata.namespace` guidance in the details stream.
- [x] 1.4 Add `TestNamespaceOverride_InstanceRefusedModuleAllowed` in `internal/workflow/render/skip_test.go` over the skip-unprovided fixture (`skipFixture`, with `SkipUnprovided: true`): `FromInstanceFile` with namespace `{Value: "staging", Source: SourceFlag}` returns an `ExitError` with code 2 naming both namespaces; with `Source: SourceEnv` the same naming `OPM_NAMESPACE`; with the fixture's own namespace as a flag it renders; `FromModule` with `{Value: "staging", Source: SourceFlag}` renders and `result.Instance.Namespace == "staging"` (module path still allowed). Verify it reports PASS, not SKIP.
- [x] 1.5 Set `metadata.namespace` in `internal/workflow/render/testdata/skip-unprovided/instance/instance.cue` to `"opm-skip-unprovided-itest"` (design.md "Fixture that relied on the split"); use that constant in 1.4 for the matching row. Re-run `go test -count=1 -run 'TestSkipUnprovided' ./internal/workflow/render/` and verify PASS, not SKIP.
- [x] 1.6 With `kind-opm-dev` up (`task cluster:status`), run `go run tests/integration/skip-unprovided/main.go` and verify it exits 0. This program is not in CI.
- [x] 1.7 `task fmt`, `task lint`, `task test` and `task openspec:check` green, then commit `fix(render): refuse a namespace override that disagrees with the instance file`

## 2. Say so in the -n help text

- [x] 2.1 In `internal/cmd/instance/apply.go`, `build.go`, `diff.go` and `vet.go`, change the `-n` help from "Target namespace" to "Namespace; must equal the instance file's metadata.namespace" (one shared constant in the package, e.g. beside `offlineFlagHelp`). Drop the `-n production` example from `vet`'s `Long`.
- [x] 2.2 Add a test in `internal/cmd/instance/instance_test.go` that each of the four commands' `namespace` flag has that usage string.
- [x] 2.3 Run `task docs:reference` and verify the diff in `docs/site/reference/cli/opm-instance.md` touches only the four `--namespace` rows and the dropped `vet` example.
- [x] 2.4 `task fmt`, `task lint`, `task test`, `task docs:reference:check` and `task openspec:check` green, then commit `docs(cmd): state that instance -n must match metadata.namespace`

## 3. Archive the change on this branch

- [x] 3.1 Run `openspec archive refuse-mismatched-instance-namespace --yes`, so the archive and the synced main spec ride the implementing PR; nothing is pushed to `main`.
- [x] 3.2 Verify `task openspec:check` is green and `openspec/specs/inst-commands/spec.md` carries the requirement "instance render commands refuse a namespace override that disagrees with the instance file" with all seven scenarios.
- [x] 3.3 Commit `chore(openspec): archive refuse-mismatched-instance-namespace`
