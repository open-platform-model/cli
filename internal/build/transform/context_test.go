package transform

import (
	"testing"

	"cuelang.org/go/cue"
	"github.com/stretchr/testify/assert"

	"github.com/opmodel/cli/internal/build/module"
	"github.com/opmodel/cli/internal/build/release"
)

func TestNewTransformerContext(t *testing.T) {
	rel := &release.BuiltRelease{
		ReleaseMetadata: release.ReleaseMetadata{
			Name:      "my-module",
			Namespace: "default",
			UUID:      "release-uuid",
			Labels:    map[string]string{"env": "prod"},
		},
		ModuleMetadata: module.ModuleMetadata{
			Name:    "my-module",
			Version: "1.0.0",
			FQN:     "example.com/modules@v0#MyModule",
			UUID:    "module-uuid",
			Labels:  map[string]string{"env": "prod"},
		},
	}

	component := &module.LoadedComponent{
		Name:   "webapp",
		Labels: map[string]string{"workload-type": "stateless"},
		Resources: map[string]cue.Value{
			"Container": {},
		},
		Traits: map[string]cue.Value{
			"Expose": {},
		},
	}

	ctx := NewTransformerContext(rel, component)

	assert.Equal(t, "my-module", ctx.Name)
	assert.Equal(t, "default", ctx.Namespace)
	// ReleaseMetadata: release-level fields
	assert.Equal(t, "my-module", ctx.ReleaseMetadata.Name)
	assert.Equal(t, "default", ctx.ReleaseMetadata.Namespace)
	assert.Equal(t, "release-uuid", ctx.ReleaseMetadata.UUID)
	// ModuleMetadata: module-level fields
	assert.Equal(t, "1.0.0", ctx.ModuleMetadata.Version)
	assert.Equal(t, "example.com/modules@v0#MyModule", ctx.ModuleMetadata.FQN)
	assert.Equal(t, "module-uuid", ctx.ModuleMetadata.UUID)
	assert.Equal(t, "webapp", ctx.ComponentMetadata.Name)
}

func TestTransformerContext_ToMap(t *testing.T) {
	modMeta := &module.ModuleMetadata{
		Name:    "my-module",
		FQN:     "example.com/modules@v0#MyModule",
		Version: "2.0.0",
		Labels:  map[string]string{"team": "platform"},
	}
	relMeta := &release.ReleaseMetadata{
		Name:      "release-name",
		Namespace: "production",
		UUID:      "release-uuid",
		Labels:    map[string]string{"team": "platform"},
	}
	ctx := &TransformerContext{
		Name:            "release-name",
		Namespace:       "production",
		ModuleMetadata:  modMeta,
		ReleaseMetadata: relMeta,
		ComponentMetadata: &TransformerComponentMetadata{
			Name:   "api",
			Labels: map[string]string{"tier": "backend"},
		},
	}

	m := ctx.ToMap()

	assert.Equal(t, "release-name", m["name"])
	assert.Equal(t, "production", m["namespace"])

	// CUE output shape is unchanged: #moduleReleaseMetadata with same fields
	moduleReleaseMetadata := m["#moduleReleaseMetadata"].(map[string]any)
	assert.Equal(t, "release-name", moduleReleaseMetadata["name"])                   // release name
	assert.Equal(t, "production", moduleReleaseMetadata["namespace"])                // release namespace
	assert.Equal(t, "2.0.0", moduleReleaseMetadata["version"])                       // module version
	assert.Equal(t, "example.com/modules@v0#MyModule", moduleReleaseMetadata["fqn"]) // module FQN
	assert.Equal(t, "release-uuid", moduleReleaseMetadata["identity"])               // release UUID

	componentMetadata := m["#componentMetadata"].(map[string]any)
	assert.Equal(t, "api", componentMetadata["name"])
}
