# Proposal: Module-to-CRD Generation (POC)

**Status:** Implemented as proof-of-concept. Rules and scope are expected to change before productionization; no ADR yet.

## Intent

Produce a Kubernetes `CustomResourceDefinition` from an OPM module so the Kubernetes API server becomes aware of the module's `#config` contract. This unlocks a second operational model for OPM modules, complementary to the existing release-file workflow:

| Today — release manifests                             | With CRD — custom resources                                                                         |
| ----------------------------------------------------- | --------------------------------------------------------------------------------------------------- |
| Each deployment is a rendered release file.           | A module is registered once as a CRD; users create many CR instances against it.                    |
| Schema validation happens in the CLI before apply.    | Schema validation happens server-side by `kube-apiserver` against the CRD's `openAPIV3Schema`.      |
| Instance lifecycle is file-driven + `opm apply/delete`. | Instance lifecycle is native Kubernetes (`kubectl get/apply/delete mymodules.opmodel.dev`, GitOps). |
| No resource-graph controller needed.                  | A future controller can watch CR instances and render them via the existing OPM pipeline.           |

The POC covers only the CRD emission step. Subsequent work — a controller that reconciles CR instances into rendered resources — is out of scope here but is the reason this command exists.

## What was implemented

A new subcommand:

```bash
opm module crd [path] [--group string] [-o yaml|json]
```

- `path`: module directory (default `.`)
- `--group`: API group for the CRD (default `opmodel.dev`)
- `-o, --output`: `yaml` (default) or `json`

The command loads the module, converts `#config` to an OpenAPI v3 schema, embeds that in a fully-formed `CustomResourceDefinition`, and writes it to stdout.

### Files

- [pkg/crd/names.go](../../../pkg/crd/names.go) — `DeriveNames` (module name → `Kind`/`ListKind`/`plural`/`singular`) and `DeriveVersion` (module version → CRD version string).
- [pkg/crd/schema.go](../../../pkg/crd/schema.go) — `ExtractConfigSchema` extracts `#config`, invokes `cuelang.org/go/encoding/openapi`, applies Kubernetes structural-schema post-processing.
- [pkg/crd/crd.go](../../../pkg/crd/crd.go) — `BuildCRD` composes the three derivations into an `unstructured.Unstructured`.
- [internal/cmd/module/crd.go](../../../internal/cmd/module/crd.go) — Cobra command, registered on `opm module` in [mod.go](../../../internal/cmd/module/mod.go).
- [tests/integration/module-crd/main.go](../../../tests/integration/module-crd/main.go) — server-side dry-run apply against the live `kind-opm-dev` cluster.

## Design decisions (POC rules — subject to change)

These were captured with explicit "will change" comments in-code so future revisits don't have to rediscover the rationale.

### Kind, plural, singular — naive transform

`metadata.name` drives all four `spec.names.*` fields:

- Split on `-` and `_`; PascalCase each segment; concatenate → `Kind`.
- `ListKind = Kind + "List"`.
- `singular = lowercase(Kind)`.
- `plural = singular + "s"` (naive; wrong for `-y`/`-s`/`-ch`/`-sh` endings).

Deferred: irregular plurals (`policy` → `policies`), acronym preservation (`APIServer` instead of `Apiserver`), `--plural`/`--kind` overrides.

### Version — one version per CRD

Single version element per CRD, derived from `metadata.version`:

- Leading `v` tolerated; SemVer prerelease (`-...`) and build metadata (`+...`) stripped before inspecting the major.
- `major == 0` → `v1alpha1`.
- `major >= 1` → `v<major>` (e.g. `2.3.1` → `v2`).
- No alpha/beta distinction for `1.x+` prereleases.

Deferred: multiple versions per CRD (would need a different module shape), finer prerelease mapping.

### Group — flag with default

`--group` defaults to `module.opmodel.dev` (the registry already used across the CLI). Introduced as a flag rather than a hardcoded constant to keep the CLI usable by forks or private deployments without requiring a future breaking change.

### Scope & subresources

Always `Namespaced`. No `status` subresource. Both deferred; they should become module-level declarations rather than flags once the shape of controller-backed modules is clearer.

### CRD metadata name

`<plural>.<group>` (canonical Kubernetes convention).

### Provenance (labels and annotations)

A generated CRD carries enough metadata to trace back to its source OPM
module. Labels are for selection, annotations for descriptive data.

**Labels** (always present):

| Key | Value | Source |
| --- | --- | --- |
| `app.kubernetes.io/managed-by` | `opm-cli` | [`pkg/core/LabelManagedBy`](../../../pkg/core/labels.go) — standard Kubernetes "who owns this" convention |
| `module.opmodel.dev/name` | module `metadata.name` | Required module field |
| `module.opmodel.dev/version` | module `metadata.version` | Required module field |

**Annotations** (emitted only when the corresponding module field is set):

| Key | Source |
| --- | --- |
| `module.opmodel.dev/path` | `metadata.modulePath` |
| `module.opmodel.dev/fqn` | `metadata.fqn` |
| `module.opmodel.dev/description` | `metadata.description` |
| `module.opmodel.dev/uuid` | `metadata.uuid` |

**Passthrough**: `metadata.labels` and `metadata.annotations` from the module
are merged into the CRD's labels and annotations respectively, so module
authors can propagate their own keys (e.g. `app.kubernetes.io/component`).

**Collision rule**: OPM-owned keys always win over module-declared keys. A
module cannot shadow `app.kubernetes.io/managed-by` or any
`module.opmodel.dev/*` key; a test guards this invariant in
[pkg/crd/crd_test.go](../../../pkg/crd/crd_test.go). This prevents a module
from, e.g., claiming to be managed by a different tool.

## How it works

1. Load the module via `loader.LoadModulePackage`.
2. Validate `metadata.name` and `metadata.version` are present and non-empty.
3. Derive `Kind`/`ListKind`/`plural`/`singular` + CRD `version`.
4. Rewrap `#config` in a minimal top-level value and run `openapi.Generate(..., {ExpandReferences: true})`. Rewrapping is necessary because the encoder rejects non-definition top-level fields, which any real module has (`metadata`, `debugValues`, resources, etc.).
5. Post-process the schema for structural-schema conformance:
   - Root `type: object` is enforced if absent; a non-object root is rejected (e.g. `#config: string`).
   - Root `additionalProperties: false` is stripped (defensive; conflicts with `preserveUnknownFields`).
6. Compose the CRD as `unstructured.Unstructured` and write via `cmdutil.WriteManifestOutput`.

`k8s.io/apiextensions-apiserver` was intentionally not imported; the manifest is emitted as `unstructured.Unstructured` with three API constants inlined. Keeps the dep graph small, and YAML output is identical to the typed form.

## Verification

### Unit + command tests

- `pkg/crd`: 58 subtests covering derivation edge cases, schema extraction, structural-schema post-processing, full assembly, and error paths.
- `internal/cmd/module`: 6 command-level tests — flag registration, end-to-end against the `simple-module` fixture (captures stdout via `os.Pipe`), custom group, JSON output, invalid path, invalid output format.
- `task test:unit` and `task lint` are clean.

### Server-side structural-schema validation

The integration program at [tests/integration/module-crd/main.go](../../../tests/integration/module-crd/main.go) submits the generated CRD to the live cluster with `kubernetes.Apply({DryRun: true})`. Server-side dry-run applies run the full CRD validation chain, including structural-schema checks that unit tests cannot replicate. The `simple-module` fixture passes:

```
3. Creating Kubernetes client (context: kind-opm-dev)...  OK
4. Server-side dry-run apply (validates structural schema)...
   INFO r:CustomResourceDefinition/simplemodules.opmodel.dev  + created
   OK: applied=1 created=1
```

This is the real confidence win: the K8s API server accepts our emitted schema.

## Known limitations

### CUE encoder: embed-in-disjunction pattern

`cuelang.org/go/encoding/openapi` cannot emit a schema for a struct that embeds another definition and appears inside a disjunction branch. The failure surfaces as `unsupported op . for object type (...)`.

The OPM catalog's [`schemas.#Secret`](../../../../catalog/opm/v1alpha1/schemas/config.cue) uses exactly that shape:

```cue
#Secret: #SecretLiteral | #SecretK8sRef
#SecretLiteral: { #SecretType,  value!: string }
#SecretK8sRef:  { #SecretType,  secretName!: string, remoteKey!: string }
```

Any `#config` that (transitively) references `#Secret` will hit it — e.g. the `zot_registry` example. The CLI now catches this error and returns an actionable message naming the pattern, citing `schemas.#Secret`, and suggesting a rewrite (single struct with optional variant fields). The raw emitter output is preserved via `%w` so the offending struct is visible.

Paths forward, not chosen yet:

1. Document and accept the limitation (current state).
2. Rewrite `schemas.#Secret` as a single struct with optional variants — trades compile-time mutual-exclusivity for encoder compatibility. Needs its own discussion.
3. Walk the `#config` tree before emission and rewrite problematic subtrees. Complex; schemas would diverge from the module's actual contract.
4. File upstream at `cuelang.org/go`.

### CUE encoder: bounds + default on numeric fields (resolved via fork)

A related encoder bug surfaced as `unsupported op for number` when emitting numeric fields that combine a bound conjunction with a default (e.g. `replicas: int & >=1 & <=10 | *1`). Fixed upstream by [cue-lang/cue#4331](https://github.com/cue-lang/cue/pull/4331) but not yet in a tagged `cuelang.org/go` release. To unblock CRD generation for modules that use this pattern, [go.mod](../../../go.mod) carries a `replace` directive pointing `cuelang.org/go` at the `opm-cli` branch of [github.com/orvis98/cue](https://github.com/orvis98/cue). The guard at [pkg/crd/schema.go:160](../../../pkg/crd/schema.go#L160) is kept as a regression tripwire; drop the replace directive once the fix ships upstream.

### Derivation naiveté (as noted above)

Irregular plurals, acronym preservation, multi-version CRDs, status subresources — all deferred.

## Out of scope / future work

- **Controller.** The reason this command exists is to enable a future controller that watches CR instances and renders them via the OPM pipeline. Not in this change.
- **ADR.** Deliberately skipped for the POC. If/when the derivation rules and schema-shape expectations stabilize, capture as an ADR.
- **Module schema fields for CRD metadata.** Consider moving `group`, `scope`, `plural` overrides, `subresources` into module-level declarations so two invocations of `opm module crd` against the same module produce the same CRD without flag coordination.
- **Export-format generalization.** If similar outputs are needed for other ecosystems (KRO `ResourceGraphDefinition`, Helm chart values schema, etc.), `opm module crd` might generalize to `opm module export --format=crd`.

## References

- Command long help — `opm module crd --help`
- README entry — [README.md](../../../README.md#L49)
