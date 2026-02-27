## 1. Core Go Types

- [x] 1.1 Add `ModulePath string` field to `ModuleMetadata` in `internal/core/module/module.go`
- [x] 1.2 Add `LabelWorkloadType` constant (`"core.opmodel.dev/workload-type"`) to `internal/core/labels.go`
- [x] 1.3 Update `internal/core/module/module_test.go` for new `ModulePath` field

## 2. Loader Metadata Extraction

- [x] 2.1 Update `extractModuleMetadata()` in `internal/loader/module.go` to extract `metadata.modulePath` into `meta.ModulePath`
- [x] 2.2 Update FQN extraction: extract `metadata.fqn` directly from CUE evaluation (v1alpha1 computes it as `modulePath/name:version`). Remove old `metadata.apiVersion` fallback logic.
- [x] 2.3 Update debug log in `LoadModule` to log `modulePath` instead of `fqn`
- [x] 2.4 Update `internal/loader/testdata/test-module/module.cue` — use v1alpha1 metadata structure (`modulePath`, kebab-case name, `version`, v1 FQN keys in `#resources`/`#traits`)
- [x] 2.5 Update `internal/loader/testdata/test-module/cue.mod/module.cue` if needed
- [x] 2.6 Update `internal/loader/testdata/test-module-no-values/module.cue` — same v1alpha1 treatment
- [x] 2.7 Update `internal/loader/testdata/inline-values-module/module.cue` — same v1alpha1 treatment
- [x] 2.8 Update any remaining loader testdata modules
- [x] 2.9 Update `internal/loader/module_test.go` — assertions for `ModulePath`, new FQN format (`modulePath/name:version`)
- [x] 2.10 Update `internal/loader/provider_test.go` — provider `apiVersion` assertions

## 3. Builder

- [x] 3.1 Change `opmodel.dev/core@v0` to `opmodel.dev/core@v1` in `internal/builder/builder.go` (load.Instances call and all 7 error message strings)
- [x] 3.2 Update `internal/builder/builder_test.go` for v1 expectations

## 4. Config

- [x] 4.1 Update `DefaultModuleTemplate` in `internal/config/templates.go` — change `opmodel.dev/config@v0` to `opmodel.dev/config@v1`, update provider import to `@v1`
- [x] 4.2 Update `internal/config/loader_test.go` — module path assertions from `@v0` to `@v1`
- [x] 4.3 Update `internal/cmd/config/vet_test.go` — module path assertions from `@v0` to `@v1`

## 5. Templates (simple)

- [x] 5.1 Update `internal/templates/simple/module.cue.tmpl` — imports to `@v1`, add `schemas` import, metadata to `modulePath`/kebab-case name, `#Replicas` to `#Scaling`, add workload-type label, structured image config, container name, `scaling: count:` spec
- [x] 5.2 Update `internal/templates/simple/values.cue.tmpl` — structured image `{repository, tag, digest}`
- [x] 5.3 `internal/templates/simple/cue.mod/module.cue.tmpl` — no changes needed (user module stays `@v0`)

## 6. Templates (standard)

- [x] 6.1 Update `internal/templates/standard/module.cue.tmpl` — imports to `@v1`, add `schemas` import, metadata to `modulePath`/kebab-case name, structured image config type
- [x] 6.2 Update `internal/templates/standard/components.cue.tmpl` — imports to `@v1`, `#Replicas` to `#Scaling`, add workload-type labels, `replicas` to `scaling: count:`, add container name
- [x] 6.3 Update `internal/templates/standard/values.cue.tmpl` — structured images

## 7. Templates (advanced)

- [x] 7.1 Update `internal/templates/advanced/module.cue.tmpl` — imports to `@v1`, add `schemas` import, metadata to `modulePath`/kebab-case name, structured image config
- [x] 7.2 Update `internal/templates/advanced/components.cue.tmpl` — `replicas` to `scaling: count:`, `storage: size:` to `volumes: data: persistentClaim: size:`
- [x] 7.3 Update `internal/templates/advanced/components/web.cue.tmpl` — imports to `@v1`, `#Replicas` to `#Scaling`, add workload-type label, `replicas` to `scaling: count:`
- [x] 7.4 Update `internal/templates/advanced/components/api.cue.tmpl` — imports to `@v1`, `#Replicas` to `#Scaling`, add workload-type label, `replicas` to `scaling: count:`
- [x] 7.5 Update `internal/templates/advanced/components/worker.cue.tmpl` — imports to `@v1`, `#Replicas` to `#Scaling`, add workload-type label, `replicas` to `scaling: count:`
- [x] 7.6 Update `internal/templates/advanced/components/db.cue.tmpl` — imports to `@v1`, `#PersistentVolume` to `#Volumes`, add workload-type label `"stateful"`, restructure storage to volumes
- [x] 7.7 Update `internal/templates/advanced/values.cue.tmpl` — structured images

## 8. Template Tests

- [x] 8.1 Update `internal/templates/embed_test.go` — verify rendered output matches v1alpha1 template content

## 9. Pipeline Tests and Fixtures

- [x] 9.1 Update `internal/pipeline/testdata/test-module/module.cue` — v1alpha1 metadata, FQN keys
- [x] 9.2 Update `internal/pipeline/testdata/test-module/cue.mod/module.cue` if needed
- [x] 9.3 Update `internal/pipeline/pipeline_test.go` — inline CUE provider strings with v1 FQNs
- [x] 9.4 Update `internal/pipeline/types.go` — doc comment FQN examples from `@v0` to `@v1`

## 10. Core Package Tests

- [x] 10.1 Update `internal/core/component/component_test.go` — FQN assertions, blueprint FQN references
- [x] 10.2 Update `internal/core/transformer/context_test.go` — expected context map output
- [x] 10.3 Update `internal/core/transformer/execute_test.go` — transform expectations
- [x] 10.4 Update `internal/core/module/module_test.go` — if not already covered in 1.3

## 11. Command Tests

- [x] 11.1 Update `internal/cmdutil/output_test.go` — FQN strings in test data
- [x] 11.2 Update `internal/cmd/mod/verbose_output_test.go` — FQN strings
- [x] 11.3 Update `internal/cmd/mod/vet_test.go` — module path assertions

## 12. Test Fixtures

- [x] 12.1 Update `tests/fixtures/valid/multi-values-module/module.cue` — v1alpha1 metadata, FQN keys
- [x] 12.2 Update `tests/fixtures/valid/multi-values-module/cue.mod/module.cue` — deps to `@v1`
- [x] 12.3 Update any other valid/invalid test fixtures that reference `@v0`

## 13. Examples

- [x] 13.1 Update `examples/jellyfin/` — module.cue, components.cue, cue.mod/module.cue, values.cue
- [x] 13.2 Update `examples/webapp-ingress/` — full v1alpha1 rewrite
- [x] 13.3 Update `examples/values-layering/` — full v1alpha1 rewrite
- [x] 13.4 Update `examples/multi-tier-module/` — full v1alpha1 rewrite
- [x] 13.5 Update `examples/minecraft/` — full v1alpha1 rewrite (incl. values_testing, values_production, values_forge)
- [x] 13.6 Update `examples/multi-package-module/` — full v1alpha1 rewrite
- [x] 13.7 Update `examples/blueprint-module/` — converted from blueprints to raw resources/traits (blueprints@v1 not yet published)
- [x] 13.8 Update `examples/blog/` — full v1alpha1 rewrite
- [x] 13.9 Update `examples/app-config/` — full v1alpha1 rewrite
- [x] 13.10 Update `examples/minecraft-values/` — structured image format in all value override files

## 14. Validation

- [x] 14.1 Run `task fmt` — verify all Go files formatted
- [x] 14.2 Run `task lint` — verify golangci-lint passes
- [x] 14.3 Run `task test:unit` — verify all unit tests pass (19/19 packages pass)
- [x] 14.4 Grep codebase for remaining `@v0` references in production code (excluding experiments/) — only intentional user-module identity declarations (`module: "xxx@v0"`) and one out-of-scope integration test comment remain
