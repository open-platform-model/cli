# Proposal: Module → Kubernetes Resource Definitions (CRD + XRD + Composition)

**Status:** CRD emission shipped as proof-of-concept (`opm module crd`). XRD + matching Composition emission (`opm module xrd`) is proposed; not yet implemented. Rules and scope remain POC-grade until controllers land. No ADR yet.

## Intent

Produce Kubernetes schema artifacts — `CustomResourceDefinition` (CRD) and Crossplane v2 `CompositeResourceDefinition` (XRD) plus a matching Composition — directly from an OPM module's `#config`. Both commands target the same goal from different angles: let consumers pick the right integration point for their platform without authoring the schema twice. The XRD command emits the XRD and its paired Composition as a single multi-document stream; they're two halves of the same integration and applying them independently has no meaningful use case.

| Today — release manifests                             | With CRD — custom resources                                                                         | With XRD — composite resources                                                                       |
| ----------------------------------------------------- | --------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------- |
| Each deployment is a rendered release file.           | A module is registered once as a CRD; users create many CR instances against it.                    | A module is registered once as an XRD; users create XRs that Crossplane compositions reconcile.      |
| Schema validation happens in the CLI before apply.    | Schema validation happens server-side by `kube-apiserver` against the CRD's `openAPIV3Schema`.      | Same server-side validation; Crossplane additionally wires composition selection and status fields.  |
| Instance lifecycle is file-driven + `opm apply/delete`. | Instance lifecycle is native Kubernetes (`kubectl get/apply/delete mymodules.module.opmodel.dev`, GitOps). | Instance lifecycle is native Kubernetes + Crossplane reconciliation of composed resources.          |
| No resource-graph controller needed.                  | A future OPM controller can watch CR instances and render them via the existing OPM pipeline.       | Crossplane + a composition provide the reconciliation; the OPM module is the schema source of truth. |

CRD covers the path where a dedicated OPM controller eventually watches instances. XRD covers the path where Crossplane is already the platform's composition engine, so the module doubles as the XR contract.

## What's implemented (CRD)

```bash
opm module crd [path] [--group string] [-o yaml|json]
```

- `path`: module directory (default `.`)
- `--group`: API group for the CRD (default `module.opmodel.dev`)
- `-o, --output`: `yaml` (default) or `json`

The command loads the module, converts `#config` to an OpenAPI v3 schema, embeds that in a fully-formed `CustomResourceDefinition`, and writes it to stdout.

### Files (CRD, shipped)

- [pkg/k8sgen/names.go](../../../pkg/k8sgen/names.go) — `DeriveNames` (module name → `Kind`/`ListKind`/`plural`/`singular`) and `DeriveVersion` (module version → CRD version string).
- [pkg/k8sgen/schema.go](../../../pkg/k8sgen/schema.go) — `ExtractConfigSchema` extracts `#config`, invokes `cuelang.org/go/encoding/openapi`, applies Kubernetes structural-schema post-processing.
- [pkg/k8sgen/crd.go](../../../pkg/k8sgen/crd.go) — `BuildCRD` composes the three derivations into an `unstructured.Unstructured`.
- [internal/cmd/module/crd.go](../../../internal/cmd/module/crd.go) — Cobra command, registered on `opm module` in [mod.go](../../../internal/cmd/module/mod.go).
- [tests/integration/module-crd/main.go](../../../tests/integration/module-crd/main.go) — server-side dry-run apply against the live `kind-opm-dev` cluster.

## What's proposed (XRD + Composition)

```bash
opm module xrd [path]
  [--group string] [--scope Namespaced|Cluster|LegacyCluster]
  [--comp-function string] [--comp-step string] [--comp-input-api string]
  [-o yaml|json]
```

- `--group` defaults to `module.opmodel.dev` — same as `crd` so authors don't juggle two defaults. Controls both the XRD's `spec.group` and the Composition's `spec.compositeTypeRef.apiVersion`.
- `--scope` defaults to `Namespaced` (Crossplane v2 default). `Cluster` and `LegacyCluster` are accepted for completeness; most users should leave it.
- `--comp-function` defaults to `function-opm` — the Crossplane composition function that consumes OPM modules.
- `--comp-step` defaults to `render-opm-module` — the pipeline step name.
- `--comp-input-api` defaults to `template.fn.crossplane.io/v1beta1` — the apiVersion function-opm expects on its `Input`.

The command loads the module, extracts the `#config` schema (shared with `crd`), wraps it under `properties.spec` for the XRD, builds a matching Composition that binds to the XRD via `compositeTypeRef` and points the pipeline's `function-opm` step at the module's published coordinate, and emits both manifests as a single multi-document stream to stdout.

### Files (XRD + Composition, proposed)

- `pkg/k8sgen/` — **rename of `pkg/crd/`** to reflect the broader responsibility. Contents unchanged except for the new `xrd.go` and `composition.go`. The shared helpers (`ExtractConfigSchema`, `DeriveNames`, `DeriveVersion`, `buildProvenance`, `applyStructuralSchemaRules`, `lookupString*`, `toAnyMap`) are consumed by all builders.
- `pkg/k8sgen/xrd.go` — `BuildXRD(modVal, XRDOptions) (*unstructured.Unstructured, error)`.
- `pkg/k8sgen/composition.go` — `BuildComposition(modVal, CompositionOptions) (*unstructured.Unstructured, error)`.
- `internal/cmd/module/xrd.go` — Cobra command, registered alongside `NewModuleCRDCmd` in [mod.go](../../../internal/cmd/module/mod.go). Builds both manifests and passes them as `[]*unstructured.Unstructured{xrd, composition}` to `cmdutil.WriteManifestOutput`.
- `tests/integration/module-xrd/main.go` — server-side dry-run apply against a cluster with Crossplane v2 installed.

## Design decisions

### Shared (CRD + XRD)

#### Kind, plural, singular — naive transform

`metadata.name` drives all four name fields for both CRD and XRD:

- Split on `-` and `_`; PascalCase each segment; concatenate → `Kind`.
- `ListKind = Kind + "List"`.
- `singular = lowercase(Kind)`.
- `plural = singular + "s"` (naive; wrong for `-y`/`-s`/`-ch`/`-sh` endings).

**XRD-specific:** Do **not** prefix `X` to the kind. The `X` convention existed in Crossplane v1 to distinguish XRs from claims; v2 removes claims entirely, so the prefix is vestigial. Users who want it can rename via module `metadata.name`.

Deferred: irregular plurals (`policy` → `policies`), acronym preservation (`APIServer` instead of `Apiserver`), `--plural`/`--kind` overrides.

#### Version — one version per definition

Single version element, derived from `metadata.version`:

- Leading `v` tolerated; SemVer prerelease (`-...`) and build metadata (`+...`) stripped before inspecting the major.
- `major == 0` → `v1alpha1`.
- `major >= 1` → `v<major>` (e.g. `2.3.1` → `v2`).
- No alpha/beta distinction for `1.x+` prereleases.

Deferred: multiple versions per definition, finer prerelease mapping.

#### Group — flag with default

`--group` defaults to `module.opmodel.dev` on both commands. A flag (rather than a constant) keeps the CLI usable by forks or private deployments without a breaking change.

#### Provenance (labels and annotations)

Both CRD and XRD carry enough metadata to trace back to the source module. Labels are for selection, annotations for descriptive data.

**Labels** (always present):

| Key | Value | Source |
| --- | --- | --- |
| `app.kubernetes.io/managed-by` | `opm-cli` | [`pkg/core/LabelManagedBy`](../../../pkg/core/labels.go) |
| `module.opmodel.dev/name` | module `metadata.name` | Required module field |
| `module.opmodel.dev/version` | module `metadata.version` | Required module field |

**Annotations** (emitted only when the corresponding module field is set):

| Key | Source |
| --- | --- |
| `module.opmodel.dev/path` | `metadata.modulePath` |
| `module.opmodel.dev/fqn` | `metadata.fqn` |
| `module.opmodel.dev/description` | `metadata.description` |
| `module.opmodel.dev/uuid` | `metadata.uuid` |

**Passthrough**: module `metadata.labels` / `metadata.annotations` are merged into the emitted manifest. **Collision rule**: OPM-owned keys win over module-declared keys. A test guards this invariant in [pkg/k8sgen/crd_test.go](../../../pkg/k8sgen/crd_test.go). The same invariant must hold for XRD.

### CRD-specific

#### Scope, subresources, metadata name

- `spec.scope` is always `Namespaced`. No `status` subresource. Both deferred; they should become module-level declarations rather than flags once controller-backed modules stabilize.
- `metadata.name` uses the canonical `<plural>.<group>` convention.

### XRD-specific (Crossplane v2 only)

#### apiVersion and kind

- `apiVersion: apiextensions.crossplane.io/v2`
- `kind: CompositeResourceDefinition`

v1 is deliberately unsupported. v2 removes claims and defaults XRs to `Namespaced`, which matches how modules are intended to be consumed.

#### Scope — flag with `Namespaced` default

`--scope` accepts `Namespaced` (default) | `Cluster` | `LegacyCluster`. `LegacyCluster` exists only for users migrating from v1 and is expected to be rare.

#### Schema wrapping

`#config` is wrapped under `properties.spec` so the emitted `openAPIV3Schema` matches what Crossplane requires:

```yaml
openAPIV3Schema:
  type: object
  properties:
    spec:
      type: object
      properties: { <config properties> }
      required:   [ <config required>  ]
  required:
    - spec
```

`properties.status` is **not** emitted in POC scope; compositions own status in v2.

#### Per-version flags

Exactly one version element, with `served: true` and `referenceable: true`. `storage` is deliberately **not** emitted — XRD v2 removed the field, and `referenceable` maps onto the underlying CRD's `spec.versions[*].storage` internally. Multi-version XRDs inherit the same deferral as multi-version CRDs.

#### Reserved-field check

Before emission, reject a `#config` whose top-level `properties` includes `crossplane` (Crossplane v2 reserves `spec.crossplane.*` and `status.crossplane.*`). The error names the offending field and points at the module's `#config`.

### Composition-specific

#### apiVersion and kind

- `apiVersion: apiextensions.crossplane.io/v1` (note: v1, not v2 — Crossplane v2 kept Composition on v1 even as XRD moved to v2).
- `kind: Composition`.

#### Binding to the XRD

The XRD ↔ Composition relationship is carried entirely by `spec.compositeTypeRef`:

- `compositeTypeRef.apiVersion`: `<opts.Group>/<derivedVersion>` — must match the XRD's group + version.
- `compositeTypeRef.kind`: `DeriveNames(metadata.name).Kind` — must match the XRD's `spec.names.kind`.

Because both are built in the same `runModuleXRD` invocation from the same module metadata, drift is structurally impossible. A `TestBuildComposition_PairsWithXRD` unit test pins this invariant by calling both builders and asserting the references line up.

#### Composition name

`metadata.name = metadata.name` verbatim (e.g. `simple-module`). The `compositeTypeRef` does the structural binding, so the Composition name carries no load; keeping it short and readable is preferred over the `<plural>.<group>` convention used for CRDs and XRDs.

#### Pipeline — single step against function-opm

Exactly one pipeline step. The step calls `function-opm` with an input describing the module to render. Input shape (matches function-opm's Go contract):

```yaml
input:
  apiVersion: template.fn.crossplane.io/v1beta1
  kind: Input
  module:
    path:    <metadata.modulePath>/<metadata.name>   # full module identifier
    version: <metadata.version>                      # SemVer, verbatim (no 'v' prefix)
```

`metadata.modulePath` is **required** on the module (unlike for CRD/XRD provenance, where it is optional) — without it the Composition has no module coordinate to pass the function. The command fails cleanly when it's missing.

`--comp-function`, `--comp-step`, and `--comp-input-api` override the defaults for users running forks or different function variants.

#### No schema

The Composition carries no OpenAPI schema. The CUE-encoder `#Secret`-shape limitation documented under "Known limitations" does not apply to the Composition emitter (it only affects `ExtractConfigSchema`, which the Composition does not call).

## How it works

1. Load the module via `loader.LoadModulePackage`.
2. Validate `metadata.name` and `metadata.version` are present and non-empty.
3. Derive `Kind`/`ListKind`/`plural`/`singular` + version string.
4. Run `ExtractConfigSchema`: rewrap `#config`, run `openapi.Generate(..., {ExpandReferences: true})`, post-process for structural-schema conformance (root `type: object`, strip root `additionalProperties: false`).
5. **CRD**: embed the schema directly as `openAPIV3Schema`.
   **XRD**: wrap under `properties.spec`, add `required: [spec]`, run the reserved-field check.
6. For `xrd`: additionally build a matching Composition with `BuildComposition` — skips `ExtractConfigSchema`, reads `metadata.modulePath` (required), and assembles the pipeline step.
7. Compose as `unstructured.Unstructured` and write via `cmdutil.WriteManifestOutput`. The `xrd` command passes a two-element slice so both manifests are emitted as a single multi-document stream.

`k8s.io/apiextensions-apiserver` and `crossplane-runtime` are intentionally not imported; all three manifests are emitted as `unstructured.Unstructured` with API constants inlined. Keeps the dep graph small, and YAML output is identical to the typed form.

## Verification

### CRD (shipped)

- `pkg/k8sgen`: 58 subtests covering derivation, schema extraction, structural-schema post-processing, full assembly, and error paths.
- `internal/cmd/module`: 6 command-level tests for `crd`.
- `task test:unit` and `task lint` clean.
- Server-side dry-run apply via [tests/integration/module-crd/main.go](../../../tests/integration/module-crd/main.go) against `kind-opm-dev` — the real confidence win; the API server accepts the emitted schema.

### XRD + Composition (planned, mirrors CRD)

- `pkg/k8sgen/xrd_test.go`: assembly, schema wrapping under `spec`, `--scope` values, provenance, reserved-field check, error paths.
- `pkg/k8sgen/composition_test.go`: assembly, `compositeTypeRef` pairing with `BuildXRD` output (shared invariant test), flag defaults vs. overrides, module-path assembly across registry depths, provenance, error paths including missing `metadata.modulePath`.
- `internal/cmd/module/xrd_test.go`: flag registration (including `--comp-*`), end-to-end against the `simple-module` fixture asserting both manifests land in stdout separated by a YAML document marker, custom group propagating to both the XRD and the Composition's `compositeTypeRef`, `--scope Cluster`, `--comp-*` overrides, JSON output, invalid path/format.
- `tests/integration/module-xrd/main.go`: server-side dry-run apply, requires Crossplane v2 preinstalled on the target cluster. Document the prerequisite in the integration task's README snippet.

## Known limitations

### CUE encoder: embed-in-disjunction pattern

Carries forward unchanged. `cuelang.org/go/encoding/openapi` cannot emit a schema for a struct that embeds another definition inside a disjunction branch. Surfaces as `unsupported op . for object type (...)`.

The OPM catalog's [`schemas.#Secret`](../../../../catalog/opm/v1alpha1/schemas/config.cue) uses that shape:

```cue
#Secret: #SecretLiteral | #SecretK8sRef
#SecretLiteral: { #SecretType,  value!: string }
#SecretK8sRef:  { #SecretType,  secretName!: string, remoteKey!: string }
```

Any `#config` that (transitively) references `#Secret` will hit it. The CLI catches this error and returns an actionable message naming the pattern. Applies identically to XRD since `ExtractConfigSchema` is shared.

Paths forward: document (current), rewrite `schemas.#Secret` as a single struct with optional variants, preprocess the `#config` tree, or file upstream. None chosen yet.

### CUE encoder: bounds + default on numeric fields (resolved via fork)

Fixed upstream by [cue-lang/cue#4331](https://github.com/cue-lang/cue/pull/4331) but not yet in a tagged `cuelang.org/go` release. [go.mod](../../../go.mod) carries a `replace` directive pointing at the `opm-cli` branch of [github.com/orvis98/cue](https://github.com/orvis98/cue); drop it once the fix ships upstream. Applies identically to XRD.

### jsonschema encoder — not yet viable

`cuelang.org/go/encoding/jsonschema.Generate` (new in v0.15.0) would let us drop the fork and target JSON Schema draft 2020-12 directly. Spiked against v0.15.4 and v0.16.1 and rejected: the encoder does not emit `default` values and stack-overflows when forced to inline references (both blockers for CRD/XRD output). Revisit when upstream lands `default` support.

### Derivation naiveté

Irregular plurals, acronym preservation, multi-version definitions, status subresources / `properties.status` — all deferred.

## Out of scope / future work

- **OPM controller.** The reason `crd` exists is to enable a future controller that watches CR instances and renders them via the OPM pipeline. Not in this change.
- **Claims.** Explicitly not supported; Crossplane v2 removes them.
- **ADR.** Deliberately skipped for the POC. Once derivation rules, schema-wrapping conventions, and scope semantics stabilize across both emitters, capture as a single ADR covering both.
- **Module schema fields for resource-definition metadata.** Consider moving `group`, `scope`, `plural` overrides, `subresources`, `status` schema, Crossplane composition defaults into module-level declarations so invocations of `crd` and `xrd` are deterministic without flag coordination.
- **Export-format generalization.** If similar outputs are needed for other ecosystems (KRO `ResourceGraphDefinition`, Helm chart values schema, etc.), both `crd` and `xrd` might unify under `opm module export --format=...`.

## References

- Command long help — `opm module crd --help` (and `xrd --help` once shipped)
- [Crossplane v2 Composite Resource Definitions](https://docs.crossplane.io/latest/composition/composite-resource-definitions/)
- [Crossplane v2 Compositions](https://docs.crossplane.io/latest/composition/compositions/)
- [Crossplane v2 What's New](https://docs.crossplane.io/latest/whats-new/)
- README entry — [README.md](../../../README.md#L49)
