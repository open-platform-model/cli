# Experiment: values-overlay

Explores strategies for merging module author defaults with user-provided values
files, with two hard requirements:

1. **User values always override author defaults**
2. **Multiple user values files unify in order (last file wins on conflict)**

## Background

A module author defines default values in `values.cue` inside the module
directory. Users can supply one or more values files via `--values` flags. The
pipeline must produce a single, fully-concrete value set for the release.

```
Module author defines:          User provides (0 or more):
  #config: {                      --values production.cue
    image:    string              --values hotfix.cue
    replicas: int & >=1 & <=10
    port:     int & >0 & <=65535
    debug:    bool
    env:      "dev" | "staging" | "prod"
  }
  values: #config                 // author fills this in values.cue
```

The desired priority stack, lowest to highest:

```
┌─────────────────────────────────────────────────────────────────┐
│                      PRIORITY STACK                             │
│                                                                 │
│  Layer 0 (floor):   module author defaults (values.cue)         │
│  Layer 1:           --values file[0]                            │
│  Layer 2:           --values file[1]                            │
│  ...                                                            │
│  Layer N (ceiling): --values file[N-1]  ← wins on all overlaps │
│                                                                 │
│  Keys absent from all user layers fall through to the floor.    │
└─────────────────────────────────────────────────────────────────┘
```

### Why this is non-trivial

CUE's core operator is **unification** (conjunction, `&`). Unification is
additive: it narrows two values toward the most specific common constraint.
When two sources both provide a concrete value for the same field — even the
same type, just different values — unification produces `_|_` (bottom, a
conflict error):

```
"app:latest" & "app:1.2.3"  →  _|_    (conflict)
2 & 5                        →  _|_    (conflict)
```

"Override" requires the later value to **replace** the earlier one, not
unify with it. CUE has no native replace operator; override must be
implemented either through careful ordering of CUE operations or by
moving the merge step outside CUE.

---

## Testdata

```
testdata/
  module/
    cue.mod/module.cue       CUE module declaration
    module.cue               #config schema + values: #config
    values.cue               Author defaults (loaded in isolation)
  values_partial.cue         User file — only image and replicas
  values_override.cue        User file — overlaps partial on image, adds debug/env
  values_full.cue            User file — all five fields, standalone
```

### Field matrix

| Field      | Author default | values_partial | values_override | values_full |
|------------|----------------|----------------|-----------------|-------------|
| `image`    | `app:latest`   | `app:1.2.3`    | `app:release`   | `app:2.0.0` |
| `replicas` | `2`            | `5`            | —               | `3`         |
| `port`     | `8080`         | —              | —               | `9090`      |
| `debug`    | `false`        | —              | `true`          | `false`     |
| `env`      | `"dev"`        | —              | `"staging"`     | `"prod"`    |

`—` means the file does not set that field.

---

## Approaches

### Approach A — Sequential FillPath (full struct)

**File:** `approach_a_test.go`

The naive attempt: call `FillPath("values", v)` once per layer, in priority
order.

```
step1 := base.FillPath("values", authorDefaults)
step2 := step1.FillPath("values", userValues)    // does this override?
```

**How it works:**

`FillPath` under the hood performs CUE unification at the target path.
Filling an **abstract** field (a constraint) with a concrete value narrows
it — this works. Filling an **already-concrete** field with a different
concrete value is a conflict.

```
Abstract + concrete   →  concrete  (OK: constraint narrows to value)
"app:latest" + "app:1.2.3"  →  _|_    (CONFLICT)
2 + 5                        →  _|_    (CONFLICT)
```

**Conclusion:** Sequential FillPath **cannot** implement override semantics
for overlapping fields. It works only when value sources are non-overlapping
(purely additive). Reversing the order does not help — the conflict appears
on whichever field is set second.

---

### Approach B — CUE-native priority stack (reverse abstract detection)

**File:** `approach_b_test.go`

Stays in the CUE evaluation layer but changes the operation order: process
layers from **highest priority to lowest**. Before filling any field, check
whether it is already concrete. If it is, skip — a higher-priority layer
already set it.

```go
func approachBFill(base cue.Value, layers ...cue.Value) cue.Value {
    result := base
    for i := len(layers) - 1; i >= 0; i-- {   // reverse: highest first
        for each field in layers[i]:
            if result.field is not yet concrete:
                result = result.FillPath(field, value)
    }
    return result
}
```

**Why this works:**

- Highest-priority layer runs first against the abstract base — all its
  fields are abstract, so every FillPath succeeds cleanly.
- Lower-priority layers see those fields already concrete and skip them.
- Lower-priority fields that the higher-priority layer did NOT set are
  still abstract — lower layers fill them cleanly.

**Defaults fallthrough:** include author defaults as `layers[0]` (lowest
priority). They fill any fields that no user layer set.

**Limitation:** this approach handles flat structs cleanly. Nested structs
require recursive field walking. Abstract detection via
`Validate(Concrete(true))` checks the whole subtree, which may miss
partially-concrete nested structs.

**Conclusion:** Works correctly and stays in CUE-land. Suitable when the
values struct is flat. For production use, the Go-level merge (Approach C/D)
is simpler and more robust.

---

### Approach C — Go map merge + single FillPath recompile

**File:** `approach_c_test.go`

Escapes the CUE evaluation layer for the merge step entirely:

```
1. Extract each value layer as a Go map (via JSON marshalling).
2. Deep-merge all maps in Go — later maps win on conflict (last-wins).
3. Compile the merged map back to a CUE value.
4. FillPath the compiled result onto the abstract base ONCE.
```

```go
func approachCMerge(base cue.Value, layers ...cue.Value) (cue.Value, error) {
    merged := map[string]any{}
    for _, layer := range layers {         // lowest priority → highest
        merged = deepMerge(merged, cueToMap(layer))
    }
    mergedVal := ctx.CompileBytes(jsonMarshal(merged))
    return base.FillPath("values", mergedVal), nil
}
```

**Why it cannot conflict:** the CUE base is touched exactly once, with an
already-resolved concrete struct. There is no opportunity for CUE
unification to see two concrete values for the same field.

**Trade-off:** the JSON round-trip (`CUE → JSON → CUE`) evaluates any CUE
expressions in the values files before merging. Values files that contain
only concrete scalars (the common case) are unaffected. CUE-native
expressions (`replicas: 2 + 1`) are evaluated first, producing `3`, and
the expression is not preserved.

**Limitation:** without author defaults as a base layer, a partial user file
leaves abstract gaps. `Validate(Concrete(true))` will fail if the module's
`#config` requires all fields to be concrete.

**Conclusion:** Reliable, conflict-free, last-wins semantics. The simplest
fully-correct approach for override behaviour. Pair with Approach D for
defaults fallthrough.

---

### Approach D — Defaults-as-base layer (recommended)

**File:** `approach_d_test.go`

Extends Approach C by treating the author's `values.cue` as **layer 0**
(the floor) in the Go-level merge. This is the only semantic addition:

```go
func approachDMerge(base, authorDefaults cue.Value, userLayers ...cue.Value) (cue.Value, error) {
    allLayers := append([]cue.Value{authorDefaults}, userLayers...)
    return approachCMerge(base, allLayers...)
}
```

**Consequences:**

| Scenario                            | Approach C        | Approach D        |
|-------------------------------------|-------------------|-------------------|
| No user files                       | abstract gaps     | defaults used     |
| Partial user file                   | abstract gaps     | gaps filled       |
| Full user file                      | user wins         | user wins         |
| Two overlapping user files          | last wins         | last wins         |
| `Validate(Concrete(true))`          | fails (partial)   | passes (if defaults cover #config) |

**Schema enforcement:** the Go-level merge does not validate values against
`#config`. The FillPath onto the abstract base applies the CUE schema
constraint, but the JSON round-trip may cause integer range constraints
(e.g. `int & <=10`) to not catch out-of-range values immediately. Explicit
schema validation via `mod.Config.Unify(mergedVal)` should be performed
after the merge, before FillPath.

**Conclusion:** This is the recommended production approach. It gives:

- Conflict-free override semantics
- Correct last-file-wins for multiple user files
- Defaults fallthrough for partial user files
- A fully concrete result suitable for `Validate(Concrete(true))`

---

## Summary comparison

```
┌────────────────┬──────────────┬──────────┬──────────────────┬──────────────┐
│ Approach       │ Override     │ Defaults │ Multi-file       │ CUE-native   │
│                │ semantics    │ fallthru │ last-wins        │              │
├────────────────┼──────────────┼──────────┼──────────────────┼──────────────┤
│ A: Sequential  │ [ ] FAILS    │ [ ]      │ [ ] FAILS        │  [x]         │
│    FillPath    │ (conflicts)  │          │ (conflicts)      │              │
├────────────────┼──────────────┼──────────┼──────────────────┼──────────────┤
│ B: CUE-native  │ [x] works    │ [x]      │ [x] works (flat) │  [x]         │
│    priority    │              │          │ [ ] nested?      │              │
│    stack       │              │          │                  │              │
├────────────────┼──────────────┼──────────┼──────────────────┼──────────────┤
│ C: Go map      │ [x] works    │ [ ]      │ [x] works        │  [ ]         │
│    merge       │              │          │                  │ (JSON round- │
│                │              │          │                  │  trip)       │
├────────────────┼──────────────┼──────────┼──────────────────┼──────────────┤
│ D: Defaults-   │ [x] works    │ [x]      │ [x] works        │  [ ]         │
│    as-base     │              │          │                  │ (JSON round- │
│    (RECOMMENDED│              │          │                  │  trip)       │
│    for prod)   │              │          │                  │              │
└────────────────┴──────────────┴──────────┴──────────────────┴──────────────┘
```

## Running

```bash
cd experiments/values-overlay
go test ./... -v
```

Or a single approach:

```bash
go test -v -run TestApproachD
```
