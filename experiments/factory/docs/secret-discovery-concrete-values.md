# Secret Discovery and CUE Concrete Value Propagation

## Problem

`_autoSecrets` in `#ModuleRelease` (and by extension `#BundleRelease`) always
produced `data: {}` in the emitted K8s `Secret` resource, even when concrete
literal values were supplied in the consumer's `values.cue`.

The symptom: `task run` and `task run:bundle` produced no `kind: Secret` resource
in their output, despite the `opm-secrets` component being correctly built and
matched by the secret transformer.

---

## Investigation

### Hypothesis 1: `_autoSecrets` captures schema constraints, not concrete values

The initial hypothesis was that the bug lived in `_autoSecrets` in
`module_release.cue`:

```cue
let unifiedModule = #module & {#config: values}
_autoSecrets: (schemas.#AutoSecrets & {#in: unifiedModule.#config}).out
```

The theory: `unifiedModule.#config` is a definition field, so CUE returns its
schema-constraint form rather than the concrete unified value — causing
`#DiscoverSecrets` to capture `value!: string` (open constraint) instead of
`value: "minecraft"` (concrete).

**Testing revealed this was wrong.** `cue eval` and `cue export` on `_autoSecrets`
both produce fully concrete output:

```json
{
  "server-secrets": {
    "rcon-password": {
      "$opm": "secret",
      "$secretName": "server-secrets",
      "$dataKey": "rcon-password",
      "value": "minecraft"
    }
  }
}
```

`_autoSecrets` itself is correct. The concrete value survives `#DiscoverSecrets`
and `#GroupSecrets` just fine. All patterns tested (PatternA through PatternD)
produced identical concrete output.

### Hypothesis 2 (correct): comprehension variable in `#OpmSecretsComponent`

Tracing the pipeline further revealed the data is lost one step later, in
`core/helpers/autosecrets.cue`, inside `#OpmSecretsComponent`:

```cue
// The broken code
for secretName, data in #secrets {
    (secretName): schemas.#SecretSchema & {
        name: secretName
        data: data   // ← 'data' does NOT carry concrete values here
    }
}
```

The comprehension variable `data` is bound to the map value for each iteration.
When used directly as a field value in a definition unification
(`schemas.#SecretSchema & {data: data}`), CUE does **not** preserve its concrete
content — the `data` field in `#SecretSchema` (typed as `[string]: #Secret`)
sees only the schema constraint of each entry, not the concrete `value: "minecraft"`.

The result: `spec.secrets["server-secrets"].data` is `{}` after `cue export`
(all entries non-concrete, stripped by export).

**Confirmed by isolation test:**

```cue
// BROKEN: data loses concrete values when passed directly into definition unification
for secretName, data in #secrets {
    (secretName): schemas.#SecretSchema & {
        name: secretName
        data: data         // exports as data: {}
    }
}

// FIXED: let-capture forces eager evaluation, preserving concrete values
for secretName, data in #secrets {
    let _d = data
    (secretName): schemas.#SecretSchema & {
        name: secretName
        data: _d           // exports correctly with all entries
    }
}
```

---

## Root Cause: Comprehension Variables and `let` Capture

CUE comprehension variables (`for k, v in expr`) are **lazy references** into
the iterated value. When you pass a comprehension variable directly as an input
field to a definition, CUE threads the reference — not the evaluated concrete
value — through the unification. The definition's own type constraint
(`[string]: #Secret`) then dominates, and the concrete entries are lost.

A `let` binding within the comprehension forces **eager evaluation** in the
current scope. `let _d = data` captures the concrete struct value before it
is passed into the definition unification. The concrete entries survive.

This is the same pattern already documented for `#ImmutableName` in
`schemas/config.cue`:

> **Definition fields lose concrete values when forwarded via `{#data: #data}`
> — only the constraint propagates. Regular fields carry concrete values through
> unification chains.**

The `#ImmutableName` fix was `let _d = data` before passing into `#ContentHash`.
The fix here is identical: `let _d = data` before passing into `#SecretSchema`.

---

## The Fix

One-line change in `core/helpers/autosecrets.cue`, applied to both the
`#resources` block and the `spec` block:

```cue
// Before (broken)
for secretName, data in #secrets {
    (secretName): schemas.#SecretSchema & {
        name: secretName
        data: data
    }
}

// After (fixed)
for secretName, data in #secrets {
    let _d = data   // let-capture preserves concrete #Secret entries
    (secretName): schemas.#SecretSchema & {
        name: secretName
        data: _d
    }
}
```

---

## Decision Log: Approaches Considered for `_autoSecrets`

Before the actual root cause was found, several approaches were considered and
tested for fixing the hypothesised `_autoSecrets` concreteness problem.
All were tested against the real CUE module packages.

### Option A — `resolvedConfig` on `#ModuleRelease` directly

```cue
// In module_release.cue
resolvedConfig: unifiedModule.#config & values
_autoSecrets: (schemas.#AutoSecrets & {#in: resolvedConfig}).out
```

**Result in testing:** Concrete. But not needed — `_autoSecrets` was already
producing concrete output. This approach was set aside.

### Option B — Widen `#in` constraint

Widening `#in: {...}` to `#in: _` in `#DiscoverSecrets`/`#AutoSecrets`.

**Result in testing:** Attempted and reverted. Changing `#in: {...}` to `#in: _`
causes `_autoSecrets` to fail in the bundle path with
`cannot range over #in (incomplete type _)` — CUE requires `for` comprehension
targets to be concrete struct types, and `_` (top) is not iterable. The `{...}`
constraint serves two roles: it accepts open structs AND it ensures the value is
iterable. Dropping it breaks the comprehension. Not viable without a more complex
restructure.

### Option C — Regular field on `#Module`

```cue
// In module.cue
resolvedConfig: #config
```

**Result in testing:** Concrete. But not needed for the same reason as Option A.

### Option D — `let` binding overlay at call site

```cue
let _input = (#module.#config) & values
_autoSecrets: (schemas.#AutoSecrets & {#in: _input}).out
```

**Result in testing:** Concrete. But not needed — all patterns were already
producing concrete output from `_autoSecrets`.

**Key insight from Option D testing:** The bug was NOT in `_autoSecrets` at all.
Testing Option D is what revealed this — when all four patterns (A, B, C, D)
produced identical correct output, the investigation moved downstream to
`#OpmSecretsComponent`, where the actual bug was found.

---

## General Pattern for Comprehension Variable Capture

When iterating a map and passing each value into a definition unification, always
`let`-capture the comprehension variable first:

```cue
// WRONG: comprehension variable passed directly into definition unification
for k, v in someMap {
    (k): #SomeDefinition & {field: v}   // v may lose concrete content
}

// RIGHT: let-capture before use
for k, v in someMap {
    let _v = v
    (k): #SomeDefinition & {field: _v}  // _v preserves concrete content
}
```

This applies whenever:

- The map values are structs with concrete scalar fields
- The definition field being unified with (`field`) is typed (e.g. `[string]: #Secret`)
- Concrete values need to survive into `cue export` / `Syntax(cue.Final())`

---

## Affected Files

| File | Change |
|------|--------|
| `core/helpers/autosecrets.cue` | `let _d = data` in both comprehensions inside `#OpmSecretsComponent` |

## Test Verification

```bash
task run        # standalone ModuleRelease
task run:bundle # BundleRelease
```

Both now emit:

```yaml
kind: Secret
stringData:
  rcon-password: minecraft          # standalone
  rcon-password: change-me-in-production  # bundle
```
