## Context

CUE cannot generate the `opm-secrets` component inside `core/module_release.cue` because it would create a circular import: `core → resources/config → core`. The CUE side solves this by computing `_autoSecrets` as a hidden field on `#ModuleRelease` using `schemas.#AutoSecrets`. The CLI must read this field and build the component in Go.

The builder already loads `opmodel.dev/core@v1` via `load.Instances` with `Dir: mod.ModulePath` to resolve from the module's dependency cache. The same pattern can load `opmodel.dev/resources/config@v1` to access the `#Secrets` schema.

The `_autoSecrets` field has this shape after CUE evaluation:

```text
{
    "<secretName>": {
        "<dataKey>": #Secret  // one of #SecretLiteral | #SecretK8sRef | #SecretEsoRef
    }
}
```

The target `#Secrets` component expects:

```text
spec: secrets: [secretName]: {
    name: string | *secretName  // defaulted from map key
    data: [string]: #Secret
}
```

So the mapping is: each `_autoSecrets` entry becomes `spec.secrets."<secretName>".data`.

## Goals / Non-Goals

**Goals:**

- Read `_autoSecrets` from the evaluated `#ModuleRelease` CUE value
- Build an `opm-secrets` component that matches `#SecretTransformer` requirements (correct `#resources` FQN)
- Inject the component into the build pipeline transparently
- Reserve the `opm-secrets` component name with a clear error on collision

**Non-Goals:**

- Modifying the `ModuleRelease` Go struct (no `AutoSecrets` field needed)
- Changing transformer behavior (transformers already handle all secret variants)
- Supporting secrets deeper than 3 levels of nesting (CUE-side constraint, documented)
- Adding CLI flags for secrets (fully automatic)

## Decisions

### Decision 1: FillPath over Go struct marshaling

**Choice:** Build the `opm-secrets` component via CUE `FillPath` starting from the `#Secrets` schema, rather than marshaling a Go struct to JSON.

**Rationale:** FillPath preserves CUE type information natively. The `#Secrets` schema (which is `core.#Component & {...}`) carries `#resources` with the correct FQN, `metadata.annotations` with `list-output: true`, and the `spec` shape computed via `_allFields`. Starting from this schema and filling in `metadata.name` and `spec.secrets.*.data` lets CUE handle all defaults (`name` from map key, `type: *"Opaque"`, `immutable: *false`).

**Alternative considered:** Marshal `autoSecretsComponent` Go struct to JSON, compile to CUE, unify with `#Secrets`. This adds a JSON round-trip and risks losing CUE constraint information.

### Decision 2: Dedicated `autosecrets.go` file

**Choice:** All auto-secrets logic in `internal/builder/autosecrets.go`, called from `builder.go` via a single `injectAutoSecrets()` function.

**Rationale:** Keeps `builder.go` unchanged except for a 3-line insertion. The auto-secrets logic is self-contained and independently testable. The builder remains focused on the core build flow.

### Decision 3: Use `component.ExtractComponents()` for component extraction

**Choice:** Wrap the built CUE value in `{"opm-secrets": value}` and pass to the existing public `ExtractComponents()` function, then pull the result from the map.

**Rationale:** Reuses the existing extraction logic that reads `#resources`, `#traits`, `metadata`, and `spec`. Keeps component representation consistent across user-defined and auto-generated components. `extractComponent()` is unexported, so we use the public wrapper.

### Decision 4: Lazy loading of `resources/config@v1`

**Choice:** Only load `opmodel.dev/resources/config@v1` when `_autoSecrets` is non-empty.

**Rationale:** No overhead when modules have no secrets. When they do, the second `load.Instances` call resolves from the same `opmodel.dev@v1` module already in the dependency cache. No caching required.

### Decision 5: `cue.Hid` with `opmodel.dev/core@v1` package path

**Choice:** Access `_autoSecrets` via `cue.MakePath(cue.Hid("_autoSecrets", "opmodel.dev/core@v1"))`.

**Rationale:** `_autoSecrets` is defined in `package core` at `opmodel.dev/core@v1`. The experiments directory confirms that hidden fields require the full module path as the second argument to `cue.Hid`.

## Risks / Trade-offs

- **[Risk] `_autoSecrets` shape changes in catalog** → Mitigation: The contract is defined by `schemas.#AutoSecrets` in the catalog. Changes there would be a catalog-level breaking change covered by catalog versioning.
- **[Risk] FillPath into closed `spec` fails** → Mitigation: `spec` is closed but its shape comes from `#SecretsResource.spec` which defines `secrets: [secretName=string]: #SecretSchema`. The open map pattern accepts any `secretName` key. Verified by tracing through the CUE definitions.
- **[Risk] `ExtractComponents` on synthesized value behaves differently** → Mitigation: The value is unified with `#Secrets` (which is `core.#Component &`), so it has the same structure as any user-defined component. Test confirms `#resources` extraction works.
- **[Trade-off] Second `load.Instances` call** → Acceptable. Only runs when secrets exist. Sub-package of already-resolved `opmodel.dev@v1` module.
