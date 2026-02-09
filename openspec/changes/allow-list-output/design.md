## Context

The transformer workflow produces Kubernetes resources from OPM components. Each transformer's `#transform` function receives a component and context, then returns an `output`. The CUE schema currently constrains `output: {...}` (struct-only), but the Go executor (`internal/build/executor.go:226-266`) already handles three output shapes:

1. **Single resource** — struct with `apiVersion` (e.g., Deployment)
2. **Map of resources** — struct without `apiVersion`, keyed by name (e.g., PVC per volume)
3. **List of resources** — `[...]` array of resources

Case 3 (list) cannot be triggered today because `output: {...}` rejects lists at CUE evaluation time. Case 2 (map) works by coincidence — the PVC transformer already produces map output, but the schema comment says "Must be a single provider-specific resource."

The change introduces a `"transformer.opmodel.dev/list-output"?: bool` annotation on `#Component` as an explicit opt-in that loosens the `output` constraint for transformers matched to that component.

**Relevant code paths:**

- CUE schema: `opmodel.dev/core` — `#Component`, `#Transformer`, `#TransformerContext`
- `internal/build/release_builder.go:134-190` — `extractComponent()` builds `LoadedComponent`
- `internal/build/context.go` — `TransformerContext`, `TransformerComponentMetadata`, `NewTransformerContext()`
- `internal/build/executor.go:136-269` — `executeJob()` injects component/context via FillPath, decodes output
- `internal/build/matcher.go` — matching logic (unaffected)

## Goals / Non-Goals

**Goals:**

- Allow components to declare that their transformers may produce list or map outputs via `"transformer.opmodel.dev/list-output"?: bool`
- Update `#Transformer.#transform.output` to accept `{...} | [...]` when the annotation is set
- Propagate the annotation from component through the transformer context so CUE transformers can reference it
- Keep the Go executor's existing three-case output decoding unchanged (it already works)
- Add test coverage for list and map output scenarios

**Non-Goals:**

- Changing the matcher logic — the annotation is about output shape, not matching criteria
- Adding annotation infrastructure beyond this single annotation — keep it targeted
- Modifying existing transformers to use list output — they can opt in independently
- Validating that a transformer's actual output shape matches the annotation — CUE handles this

## Decisions

### 1. Annotation on `#Component` rather than on `#Transformer`

**Decision:** The opt-in lives on the component (`"transformer.opmodel.dev/list-output"?: bool`) rather than the transformer.

**Rationale:** Components are authored by module developers who know their data shape (e.g., volumes are a map). Transformers are provided by platform operators and should remain generic. A single transformer (e.g., PVC) should be able to produce single or multi-resource output depending on the component it processes — the component signals the intent.

**Alternative considered:** A `listOutput` field on `#Transformer.#transform` — rejected because it would require transformer authors to anticipate all possible output shapes, coupling transformers to specific component patterns.

### 2. Propagate annotation via `#TransformerContext.#componentMetadata`

**Decision:** Add an `annotations` map to `#TransformerContext.#componentMetadata` and populate it from the component's annotation. The CUE transformer can then conditionally set its output constraint based on the annotation value.

**Implementation path:**

1. **CUE (`opmodel.dev/core`):**
   - Add `"transformer.opmodel.dev/list-output"?: bool` to `#Component`
   - Add `annotations?: [string]: bool` to `#TransformerContext.#componentMetadata`
   - Change `#transform.output` from `{...}` to `{...} | [...{...}]` — the disjunction allows either shape

2. **Go (`internal/build/`):**
   - `LoadedComponent` — add `Annotations map[string]string` field
   - `release_builder.go:extractComponent()` — extract annotation: `value.LookupPath(cue.ParsePath("\"transformer.opmodel.dev/list-output\""))`
   - `context.go:TransformerComponentMetadata` — add `Annotations map[string]string` field
   - `context.go:NewTransformerContext()` — copy annotations from component to context
   - `context.go:ToMap()` — include annotations in the component metadata map
   - `executor.go` — no changes needed; output decoding already handles all three cases

**Rationale:** Piping annotations through the context keeps the data flow consistent with how labels, resources, and traits are already propagated. Transformers that need to inspect the annotation can access `#context.#componentMetadata.annotations`.

**Alternative considered:** Directly inspecting the component CUE value for the annotation in the executor — rejected because it bypasses the established context injection pattern and would not be visible to CUE transformers.

### 3. Loosen `output` constraint unconditionally in CUE

**Decision:** Change `#transform.output` from `{...}` to `{...} | [...{...}]` for all transformers, rather than conditionally based on the annotation.

**Rationale:** The Go executor already handles both shapes. Making the CUE schema match reality is simpler than adding conditional constraint logic. The annotation's primary purpose is documentation and intent signaling, not enforcement — a transformer that returns a list for a component without the annotation will still work. This avoids complex CUE conditional logic while formalizing what already works.

**Alternative considered:** Conditional `output` constraint using `if #component."transformer.opmodel.dev/list-output"` — rejected because CUE conditional constraints on definitions are complex, fragile, and violate Principle VII (simplicity). The annotation serves as documentation and forward-compatibility, not as a hard gate.

### 4. Annotation extraction uses quoted CUE path lookup

**Decision:** Extract the annotation in Go using `value.LookupPath(cue.ParsePath("\"transformer.opmodel.dev/list-output\""))` with the quoted field name.

**Rationale:** The annotation uses a dotted domain name which requires quoting in CUE field access. This is consistent with how other domain-namespaced fields (like labels) would be accessed.

## Risks / Trade-offs

**[Risk] Loosening `output` allows accidental list output from transformers that intend single-resource** → The annotation documents intent even though the constraint is unconditional. Transformer authors writing `output: { apiVersion: ... }` will naturally produce single resources. The risk of accidental list output is low because it requires explicit `[...]` syntax in CUE.

**[Risk] Annotation is optional, so existing map-producing transformers (PVC) continue working without it** → This is intentional. Existing behavior is preserved. Module authors can add the annotation when they want to be explicit. Migration is opt-in.

**[Trade-off] Unconditional schema loosening vs. conditional enforcement** → We chose simplicity. The annotation is a convention, not a constraint. If enforcement becomes needed later, it can be added without breaking changes.

**[Trade-off] Generic `annotations` map vs. dedicated boolean field on `LoadedComponent`** → Using a generic annotations map is more extensible for future annotations but slightly more complex to extract. A dedicated `ListOutput bool` field would be simpler but creates precedent for adding a new field per annotation. We use the generic map for forward-compatibility.
