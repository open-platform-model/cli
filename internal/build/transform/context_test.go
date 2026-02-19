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
		Metadata: release.Metadata{
			Name:            "my-module",
			Namespace:       "default",
			Version:         "1.0.0",
			FQN:             "example.com/modules@v0#MyModule",
			Identity:        "module-uuid",
			ReleaseIdentity: "release-uuid",
			Labels:          map[string]string{"env": "prod"},
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
	assert.Equal(t, "my-module", ctx.ModuleReleaseMetadata.Name)
	assert.Equal(t, "default", ctx.ModuleReleaseMetadata.Namespace)
	assert.Equal(t, "1.0.0", ctx.ModuleReleaseMetadata.Version)
	assert.Equal(t, "example.com/modules@v0#MyModule", ctx.ModuleReleaseMetadata.FQN)
	assert.Equal(t, "release-uuid", ctx.ModuleReleaseMetadata.Identity)
	assert.Equal(t, "webapp", ctx.ComponentMetadata.Name)
}

func TestTransformerContext_ToMap(t *testing.T) {
	ctx := &TransformerContext{
		Name:      "release-name",
		Namespace: "production",
		ModuleReleaseMetadata: &TransformerModuleReleaseMetadata{
			Name:      "my-module",
			Namespace: "production",
			FQN:       "example.com/modules@v0#MyModule",
			Version:   "2.0.0",
			Identity:  "release-uuid",
			Labels:    map[string]string{"team": "platform"},
		},
		ComponentMetadata: &TransformerComponentMetadata{
			Name:   "api",
			Labels: map[string]string{"tier": "backend"},
		},
	}

	m := ctx.ToMap()

	assert.Equal(t, "release-name", m["name"])
	assert.Equal(t, "production", m["namespace"])

	moduleReleaseMetadata := m["#moduleReleaseMetadata"].(map[string]any)
	assert.Equal(t, "my-module", moduleReleaseMetadata["name"])
	assert.Equal(t, "production", moduleReleaseMetadata["namespace"])
	assert.Equal(t, "2.0.0", moduleReleaseMetadata["version"])
	assert.Equal(t, "example.com/modules@v0#MyModule", moduleReleaseMetadata["fqn"])
	assert.Equal(t, "release-uuid", moduleReleaseMetadata["identity"])

	componentMetadata := m["#componentMetadata"].(map[string]any)
	assert.Equal(t, "api", componentMetadata["name"])
}
