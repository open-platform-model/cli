# Tasks: cli-kernel-adoption

Phases mirror design.md's migration plan: A (config, D39) → B (platform) → C (kernel swap) → D (parity + cleanup). Keep `task check` green at every phase boundary.

## 1. Phase A — `~/.opm` simplification (D39)

- [x] 1.1 Shrink the embedded config schema (`internal/config/schema/config.cue`): remove `providers` and `cacheDir`; validation error for a present `providers:` field names the removed field and hints `opm config init`
- [x] 1.2 Collapse `config.Load` to single-pass: delete `BootstrapRegistry`, `configHasProviders`, `extractProviders`, and the `CUE_REGISTRY` staging; registry resolves by ordinary flag > env > config precedence after parse
- [x] 1.3 (amended during implementation) Keep `Providers`/`CueContext` on `GlobalConfig` as documented legacy-shim fields — `config.Load` never populates `Providers` — instead of dropping them now: dropping in Phase A would force rewiring every render-path consumer twice. The fields, `resolveProvider`, and the resolved `Provider` field are deleted in Phase C (task 3.6) together with their consumers
- [x] 1.4 Rewrite `templates.go`: scalar-only `DefaultConfigTemplate`; new `DefaultPlatformTemplate` seeding `opmodel.dev/catalogs/opm` (`>=1.0.0-0 <2.0.0-0`) and `opmodel.dev/catalogs/kubernetes` (`>=1.1.0-0 <2.0.0-0`); delete `DefaultModuleTemplate`
- [x] 1.5 Rework `opm config init`: write `config.cue` + `platform.cue`, no `cue.mod/`, no tidy; update init tests
- [x] 1.6 Add the embedded platform projection schema (name, type, registry map with enable/filter.range/allow/deny; no imports allowed) in `internal/config/schema/`
- [x] 1.7 Rework `opm config vet`: validate both files; missing `platform.cue` is a note, not a failure; stale `providers`/`cue.mod` produces the migration hint; update vet tests
- [x] 1.8 Update `config` unit tests for single-pass load and removed fields; `task check` green

## 2. Phase B — platform resolution (D11/D12/D17/D21/D22)

- [x] 2.1 Create `internal/platform`: decode function (CUE file bytes or unstructured CR spec map → `synth.PlatformInput`), shared by all three sources; unit tests with table-driven fixtures
- [x] 2.2 Implement `Resolve` with precedence `--platform` file > cluster `Platform` CR (cluster-facing commands only) > `~/.opm/platform.cue`; warn on cluster→local fallback; return resolved source for provenance reporting
- [x] 2.3 (flag surface landed; call-site consumption wires in Phase C) `--platform <file>` registered on `RenderFlags` + `InstanceFileFlags` (module build/apply/vet, instance apply/build/diff/vet); offline-never-cluster and provenance printing are encoded in `platform.Resolve`/`Resolution.Describe` and become user-visible when Phase C rewires render through them
- [x] 2.4 Implement solo-cluster write-if-absent: plain dynamic-client `Create` of the `cluster` Platform from the resolved local spec, field manager `opm-cli`, `AlreadyExists` = success-noop, forbidden = warning; unit tests for 409 and 403 paths
- [x] 2.5 Wire `SynthesizePlatform` → `Materialize` on the invocation kernel behind `Resolve` (registry from resolved config); integration test materializing the seeded default platform against a local registry

## 3. Phase C — kernel adoption (D9 + 0002 carryover)

- [x] 3.1 (pulled forward into Phase B — `internal/platform` needs `synth.PlatformInput`) Add `github.com/open-platform-model/library` to `go.mod` (kernel only; verify no `opm-operator`, controller-runtime, or Flux edges appear in `go.mod`/`go.sum`); construct one `kernel.Kernel` per invocation at workflow entry
- [ ] 3.2 Port the CLI's render golden/fixture tests to drive the kernel path (side-by-side, old pipeline still in place) and record output diffs; review every diff as intended-kernel-behavior vs regression before proceeding
- [ ] 3.3 Rewire `internal/workflow/render`: instance-file path via kernel instance loading + `ProcessModuleInstance`; module-dir path via `LoadModulePackage` + `SynthesizeInstance`; registry refs via `AcquireModuleFromRegistry`; values resolution feeds a `cue.Value` (adapter or kernel `Source`s per design LD2); runtime identity `opm-cli`
- [ ] 3.4 Rewire synthesis (`opm module build` / `opm instance build <dir>`): kernel `SynthesizeInstance`, emitted kind `ModuleInstance`, no synthetic wrapper module, no `#ModuleRelease`/`modulerelease@v1` references anywhere in production code
- [ ] 3.5 Rewire `internal/workflow/apply` to consume kernel results (resources + digests) with the existing SSA apply/prune/CR-status flow untouched
- [ ] 3.6 Delete `pkg/render`, `pkg/provider`, `pkg/loader`'s provider/synth/match code, and the Phase A shim fields (`GlobalConfig.Providers`/`CueContext`, `resolveProvider`, resolved `Provider`); remove the `--provider` flag; fix all compile errors by rewiring callers to kernel/workflow seams
- [ ] 3.7 Update/retire tests of deleted packages; adapt `internal/workflow` tests to kernel fixtures; `task check` green
- [ ] 3.8 Update `mod vet` / `instance vet` paths to kernel validation (`ValidateModuleValues*` / `ProcessModuleInstance` concreteness), preserving debugValues selection behavior

## 4. Phase D — parity, digests, cleanup

- [ ] 4.1 Read the operator's digest computation and mirror it: `lastAppliedRenderDigest` over kernel-finalized manifests with the same canonical serialization; upgrade `lastAppliedSourceDigest` to the kernel content digest (replace C1's module-reference stopgap in `internal/workflow/apply`)
- [ ] 4.2 Add the D30 parity integration check (kind + registry gated): CLI render digest ≡ operator render digest for a fixture instance + Platform spec; explicit skew report when `library`/`opm-operator` CUE lines differ
- [ ] 4.3 Sweep docs (`QUICKSTART.md`, `README.md`, command help): `--platform` replaces `--provider`; `~/.opm` two-file layout; registry needed for catalog materialization on `build`
- [ ] 4.4 Verify end-to-end on the kind cluster: `opm config init` (fresh `~/.opm`), `opm instance apply` with cluster Platform present, absent (write-if-absent fires), and `--platform` override; `opm instance build` offline
- [ ] 4.5 Record the landing in `enhancements/0006/config.yaml` history (slice C2) with a note on the D30 gate result; flag `cue-binary-integration` for withdrawal/re-scope
