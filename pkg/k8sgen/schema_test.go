package k8sgen

import (
	"testing"

	"cuelang.org/go/cue/cuecontext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractConfigSchema_SimpleStruct(t *testing.T) {
	t.Parallel()

	ctx := cuecontext.New()
	modVal := ctx.CompileString(`
metadata: {name: "demo", version: "0.1.0"}
#config: {
	name: string
	replicas: int | *3
	tag?: string
}
`)
	require.NoError(t, modVal.Err())

	schema, err := ExtractConfigSchema(modVal)
	require.NoError(t, err)

	assert.Equal(t, "object", schema["type"])

	props, ok := schema["properties"].(map[string]any)
	require.True(t, ok, "expected properties to be an object")
	assert.Contains(t, props, "name")
	assert.Contains(t, props, "replicas")
	assert.Contains(t, props, "tag")

	nameProp := props["name"].(map[string]any)
	assert.Equal(t, "string", nameProp["type"])

	replicasProp := props["replicas"].(map[string]any)
	assert.Equal(t, "integer", replicasProp["type"])
	assert.EqualValues(t, 3, replicasProp["default"])

	required, ok := schema["required"].([]any)
	require.True(t, ok)
	assert.ElementsMatch(t, []any{"name", "replicas"}, required)
	assert.NotContains(t, required, "tag")
}

func TestExtractConfigSchema_NestedStruct(t *testing.T) {
	t.Parallel()

	ctx := cuecontext.New()
	modVal := ctx.CompileString(`
#config: {
	db: {
		host: string
		port: int | *5432
	}
}
`)
	require.NoError(t, modVal.Err())

	schema, err := ExtractConfigSchema(modVal)
	require.NoError(t, err)

	props := schema["properties"].(map[string]any)
	db := props["db"].(map[string]any)
	assert.Equal(t, "object", db["type"])

	dbProps := db["properties"].(map[string]any)
	assert.Contains(t, dbProps, "host")
	assert.Contains(t, dbProps, "port")

	portProp := dbProps["port"].(map[string]any)
	assert.EqualValues(t, 5432, portProp["default"])
}

func TestExtractConfigSchema_IgnoresSiblingModuleFields(t *testing.T) {
	t.Parallel()

	// openapi.Gen rejects non-definition top-level fields; verify that our
	// wrapping strategy lets a realistic module (with metadata, debugValues,
	// etc. alongside #config) produce a schema without error.
	ctx := cuecontext.New()
	modVal := ctx.CompileString(`
metadata: {name: "demo", version: "1.0.0"}
debugValues: {name: "example"}
resources: {}
#config: {
	name: string
}
`)
	require.NoError(t, modVal.Err())

	schema, err := ExtractConfigSchema(modVal)
	require.NoError(t, err)
	assert.Equal(t, "object", schema["type"])

	props := schema["properties"].(map[string]any)
	assert.Contains(t, props, "name")
	assert.NotContains(t, props, "metadata")
	assert.NotContains(t, props, "debugValues")
}

func TestExtractConfigSchema_DisjunctionRetainsObjectType(t *testing.T) {
	t.Parallel()

	// CUE emits a merged properties set with a root `oneOf` for required
	// combinations. Root type must still be "object" after processing.
	ctx := cuecontext.New()
	modVal := ctx.CompileString(`
#A: {kind: "A", x: string}
#B: {kind: "B", y: int}
#config: #A | #B
`)
	require.NoError(t, modVal.Err())

	schema, err := ExtractConfigSchema(modVal)
	require.NoError(t, err)
	assert.Equal(t, "object", schema["type"])
	assert.Contains(t, schema, "oneOf")
}

func TestExtractConfigSchema_MissingConfig(t *testing.T) {
	t.Parallel()

	ctx := cuecontext.New()
	modVal := ctx.CompileString(`metadata: {name: "demo"}`)
	require.NoError(t, modVal.Err())

	_, err := ExtractConfigSchema(modVal)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no #config definition")
}

// TestExtractConfigSchema_EmbedInDisjunction_Actionable reproduces the
// pattern used by schemas.#Secret in the OPM catalog (a struct embedding
// another definition, used inside a disjunction) — CUE's openapi emitter
// cannot represent this and fails with "unsupported op". The wrapper in
// ExtractConfigSchema converts that into an actionable error; this test
// pins the message so regressions don't silently drop the hint.
func TestExtractConfigSchema_EmbedInDisjunction_Actionable(t *testing.T) {
	t.Parallel()

	ctx := cuecontext.New()
	modVal := ctx.CompileString(`
#Base: {$opm: "secret", $name!: string}
#A: {#Base, value!: string}
#B: {#Base, remote!: string}
#Secret: #A | #B

#config: {
	creds: #Secret
}
`)
	require.NoError(t, modVal.Err())

	_, err := ExtractConfigSchema(modVal)
	require.Error(t, err)

	msg := err.Error()
	assert.Contains(t, msg, "openapi encoder cannot express")
	assert.Contains(t, msg, "embeds another definition and appears inside a disjunction")
	assert.Contains(t, msg, "schemas.#Secret")
	assert.Contains(t, msg, "Raw emitter error")
	// Original emitter error must remain wrapped for %w / errors.Is consumers.
	assert.Contains(t, msg, "unsupported op")
}

func TestExtractConfigSchema_NonObjectConfig(t *testing.T) {
	t.Parallel()

	// A #config that isn't a struct cannot be a CRD root; the structural
	// schema rule must reject it.
	ctx := cuecontext.New()
	modVal := ctx.CompileString(`#config: string`)
	require.NoError(t, modVal.Err())

	_, err := ExtractConfigSchema(modVal)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "type: object at the root")
}

func TestApplyStructuralSchemaRules(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    map[string]any
		want     map[string]any
		wantErr  string
		mutation string
	}{
		{
			name:  "object type preserved",
			input: map[string]any{"type": "object", "properties": map[string]any{}},
			want:  map[string]any{"type": "object", "properties": map[string]any{}},
		},
		{
			name:  "missing type is set to object",
			input: map[string]any{"properties": map[string]any{}},
			want:  map[string]any{"type": "object", "properties": map[string]any{}},
		},
		{
			name:  "additionalProperties false is stripped at root",
			input: map[string]any{"type": "object", "additionalProperties": false},
			want:  map[string]any{"type": "object"},
		},
		{
			name:  "additionalProperties true is preserved",
			input: map[string]any{"type": "object", "additionalProperties": true},
			want:  map[string]any{"type": "object", "additionalProperties": true},
		},
		{
			name: "additionalProperties as schema is preserved",
			input: map[string]any{
				"type":                 "object",
				"additionalProperties": map[string]any{"type": "string"},
			},
			want: map[string]any{
				"type":                 "object",
				"additionalProperties": map[string]any{"type": "string"},
			},
		},
		{
			name:    "non-object type is rejected",
			input:   map[string]any{"type": "string"},
			wantErr: "type: object at the root",
		},
		{
			name:    "non-string type is rejected",
			input:   map[string]any{"type": 42},
			wantErr: "type: object at the root",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := applyStructuralSchemaRules(tt.input)
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, tt.input)
		})
	}
}
