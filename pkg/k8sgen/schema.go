package k8sgen

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"cuelang.org/go/cue"
	"cuelang.org/go/encoding/openapi"
)

// ExtractConfigSchema generates an OpenAPI v3 schema for the module's
// #config definition and returns it as a map suitable for embedding in a
// CRD's spec.versions[].schema.openAPIV3Schema field.
//
// Behavior:
//   - Looks up #config on modVal; returns an error if it is absent.
//   - Wraps #config in a minimal top-level value before invoking
//     cuelang.org/go/encoding/openapi, because openapi.Gen rejects
//     non-definition top-level fields (a real module has metadata,
//     debugValues, resources, etc.).
//   - Expands references so the emitted schema is self-contained.
//   - Applies Kubernetes structural-schema post-processing
//     (see applyStructuralSchemaRules).
func ExtractConfigSchema(modVal cue.Value) (map[string]any, error) {
	if !modVal.Exists() {
		return nil, fmt.Errorf("module value does not exist")
	}

	configVal := modVal.LookupPath(cue.ParsePath("#config"))
	if !configVal.Exists() {
		return nil, fmt.Errorf("module has no #config definition")
	}

	ctx := modVal.Context()
	wrap := ctx.CompileString("#config: _").FillPath(cue.ParsePath("#config"), configVal)
	if err := wrap.Err(); err != nil {
		return nil, fmt.Errorf("preparing #config for OpenAPI emission: %w", err)
	}

	file, err := openapi.Generate(wrap, &openapi.Config{
		ExpandReferences: true,
	})
	if err != nil {
		return nil, wrapOpenAPIError(err)
	}

	docVal := ctx.BuildFile(file)
	if err := docVal.Err(); err != nil {
		return nil, fmt.Errorf("building OpenAPI document: %w", err)
	}
	docJSON, err := docVal.MarshalJSON()
	if err != nil {
		return nil, fmt.Errorf("marshaling OpenAPI document: %w", err)
	}

	var parsed struct {
		Components struct {
			Schemas map[string]json.RawMessage `json:"schemas"`
		} `json:"components"`
	}
	if err := json.Unmarshal(docJSON, &parsed); err != nil {
		return nil, fmt.Errorf("parsing OpenAPI output: %w", err)
	}

	// CUE's default NameFunc strips the leading '#' from definition names,
	// so #config is emitted under the key "config".
	raw, ok := parsed.Components.Schemas["config"]
	if !ok {
		return nil, fmt.Errorf(
			"OpenAPI output missing config schema (got keys %v)",
			sortedKeys(parsed.Components.Schemas),
		)
	}

	var schema map[string]any
	if err := json.Unmarshal(raw, &schema); err != nil {
		return nil, fmt.Errorf("unmarshaling config schema: %w", err)
	}

	if err := applyStructuralSchemaRules(schema); err != nil {
		return nil, err
	}
	return schema, nil
}

// applyStructuralSchemaRules mutates schema so that it satisfies the
// structural-schema requirements the Kubernetes CRD API server enforces on
// spec.versions[].schema.openAPIV3Schema.
//
// Rules applied at the root:
//  1. type must be "object". If absent, it is set; if present and not
//     "object", an error is returned (the #config is not representable as
//     a CRD spec, e.g. #config: string).
//  2. additionalProperties: false is stripped. Pruning of unknown fields is
//     governed by the CRD's preserveUnknownFields setting; leaving the
//     boolean form here risks conflicting with it.
//
// CUE's openapi emitter already produces type: object for struct
// definitions and does not emit additionalProperties: false for closed
// structs, so this is defensive for future CUE versions and for users
// that inject raw OpenAPI fragments via attributes.
func applyStructuralSchemaRules(schema map[string]any) error {
	if t, ok := schema["type"]; ok {
		s, isString := t.(string)
		if !isString || s != "object" {
			return fmt.Errorf(
				"#config schema has root type %v; CRDs require type: object at the root",
				t,
			)
		}
	} else {
		schema["type"] = "object"
	}

	if v, ok := schema["additionalProperties"]; ok {
		if b, isBool := v.(bool); isBool && !b {
			delete(schema, "additionalProperties")
		}
	}
	return nil
}

// wrapOpenAPIError turns a raw cuelang.org/go/encoding/openapi failure into a
// user-facing error. The emitter surfaces some CUE patterns it cannot
// represent as OpenAPI v3 with terse messages; this wrapper recognizes the
// ones we've cataloged and hints at the rewrite a module author can apply.
// Unrecognized failures fall through to a generic message that still blames
// the encoder, not the user — the raw error is always preserved via %w.
//
// Cataloged patterns:
//
//   - `unsupported op .` (literal dot): a struct embeds another definition
//     and appears inside a disjunction branch (e.g. `#A: {#Base, x!: string}`
//     | `#B: {#Base, y!: string}`). The OPM catalog's `schemas.#Secret`
//     uses this shape.
//
//   - `unsupported op for number`: a numeric field combines a bound
//     conjunction with a default value (e.g. `replicas: int & >=1 | *1`).
//     Even a single bound is enough to trigger it. Fixed in the orvis98/cue
//     fork which go.mod currently pins via `replace` (cherry-picked onto
//     v0.16.1); kept here as a guard against version regressions and for
//     anyone building without the replace directive. See cue-lang/cue#4305.
func wrapOpenAPIError(err error) error {
	msg := err.Error()

	switch {
	case strings.Contains(msg, "unsupported op ."):
		return fmt.Errorf(
			"generating OpenAPI for #config: CUE's openapi encoder cannot express this #config. "+
				"This is a known limitation when a struct embeds another definition and appears inside a disjunction "+
				"(e.g. `#A: {#Base, x!: string}` | `#B: {#Base, y!: string}`). "+
				"`schemas.#Secret` in the OPM catalog uses this shape, so any #config that references it will fail here. "+
				"Workaround: rewrite the offending definition as a single struct with optional variant fields, "+
				"or remove the reference from #config.\nRaw emitter error:\n%w",
			err,
		)

	// NOTE: solved by https://github.com/cue-lang/cue/pull/4331
	case strings.Contains(msg, "unsupported op for number"):
		return fmt.Errorf(
			"generating OpenAPI for #config: CUE's openapi encoder cannot express a numeric field that combines a bound conjunction with a default value "+
				"(e.g. `replicas: int & >=1 & <=10 | *1`). Even a single bound plus a default triggers this. "+
				"Workaround: choose either bounds (`replicas: int & >=1 & <=10`) OR a default (`replicas: int | *1`) for each numeric field — not both.\n"+
				"Raw emitter error:\n%w",
			err,
		)
	}

	return fmt.Errorf(
		"generating OpenAPI for #config: the CUE openapi encoder failed, likely a limitation of cuelang.org/go/encoding/openapi "+
			"rather than an issue with the module itself.\nRaw emitter error:\n%w",
		err,
	)
}

func sortedKeys(m map[string]json.RawMessage) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
