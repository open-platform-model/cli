package build

import (
	"testing"

	"cuelang.org/go/cue"
	"github.com/stretchr/testify/assert"
)

func TestBuildTransformerContext(t *testing.T) {
	module := &LoadedModule{
		Name:      "my-module",
		Namespace: "default",
		Version:   "1.0.0",
		Labels:    map[string]string{"env": "prod"},
	}

	component := &LoadedComponent{
		Name:   "webapp",
		Labels: map[string]string{"workload-type": "stateless"},
		Resources: map[string]cue.Value{
			"Container": {},
		},
		Traits: map[string]cue.Value{
			"Expose": {},
		},
	}

	ctx := BuildTransformerContext(module, component)

	assert.Equal(t, "my-module", ctx.Name)
	assert.Equal(t, "default", ctx.Namespace)
	assert.Equal(t, "my-module", ctx.ModuleMetadata.Name)
	assert.Equal(t, "1.0.0", ctx.ModuleMetadata.Version)
	assert.Equal(t, "webapp", ctx.ComponentMetadata.Name)
	assert.Contains(t, ctx.ComponentMetadata.Resources, "Container")
	assert.Contains(t, ctx.ComponentMetadata.Traits, "Expose")
}

func TestTransformerContext_ToMap(t *testing.T) {
	ctx := &TransformerContext{
		Name:      "release-name",
		Namespace: "production",
		ModuleMetadata: &TransformerModuleMetadata{
			Name:    "my-module",
			Version: "2.0.0",
			Labels:  map[string]string{"team": "platform"},
		},
		ComponentMetadata: &TransformerComponentMetadata{
			Name:      "api",
			Labels:    map[string]string{"tier": "backend"},
			Resources: []string{"Container"},
			Traits:    []string{"Expose", "Scale"},
		},
	}

	m := ctx.ToMap()

	assert.Equal(t, "release-name", m["name"])
	assert.Equal(t, "production", m["namespace"])

	moduleMetadata := m["#moduleMetadata"].(map[string]any)
	assert.Equal(t, "my-module", moduleMetadata["name"])
	assert.Equal(t, "2.0.0", moduleMetadata["version"])

	componentMetadata := m["#componentMetadata"].(map[string]any)
	assert.Equal(t, "api", componentMetadata["name"])
}
