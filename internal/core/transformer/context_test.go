package transformer

import (
	"testing"

	"cuelang.org/go/cue"
	"github.com/stretchr/testify/assert"

	"github.com/opmodel/cli/internal/core/component"
	"github.com/opmodel/cli/internal/core/module"
	"github.com/opmodel/cli/internal/core/modulerelease"
)

func TestNewTransformerContext(t *testing.T) {
	rel := &modulerelease.ModuleRelease{
		Metadata: &modulerelease.ReleaseMetadata{
			Name:      "my-module",
			Namespace: "default",
			UUID:      "release-uuid",
			Labels:    map[string]string{"env": "prod"},
		},
		Module: module.Module{
			Metadata: &module.ModuleMetadata{
				Name:    "my-module",
				Version: "1.0.0",
				FQN:     "example.com/modules@v0#MyModule",
				UUID:    "module-uuid",
				Labels:  map[string]string{"env": "prod"},
			},
		},
	}

	comp := &component.Component{
		Metadata: &component.ComponentMetadata{
			Name:   "webapp",
			Labels: map[string]string{"workload-type": "stateless"},
		},
		Resources: map[string]cue.Value{
			"Container": {},
		},
		Traits: map[string]cue.Value{
			"Expose": {},
		},
	}

	ctx := NewTransformerContext(rel, comp)

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

func TestNewTransformerContext_NameOverride(t *testing.T) {
	// Verifies spec scenario: when --name overrides the module name, the
	// TransformerContext carries the canonical module name in ModuleMetadata
	// and the release name in ReleaseMetadata — independently.
	rel := &modulerelease.ModuleRelease{
		Metadata: &modulerelease.ReleaseMetadata{
			Name:      "my-app-staging",
			Namespace: "staging",
			UUID:      "release-uuid",
		},
		Module: module.Module{
			Metadata: &module.ModuleMetadata{
				Name:    "my-app",
				Version: "1.0.0",
				FQN:     "example.com/modules@v0#MyApp",
				UUID:    "module-uuid",
			},
		},
	}

	comp := &component.Component{
		Metadata: &component.ComponentMetadata{
			Name:        "api",
			Labels:      map[string]string{},
			Annotations: map[string]string{},
		},
		Resources: map[string]cue.Value{},
		Traits:    map[string]cue.Value{},
	}

	ctx := NewTransformerContext(rel, comp)

	assert.Equal(t, "my-app-staging", ctx.Name, "top-level Name should be the release name")
	assert.Equal(t, "my-app-staging", ctx.ReleaseMetadata.Name, "ReleaseMetadata.Name should be the release name")
	assert.Equal(t, "my-app", ctx.ModuleMetadata.Name, "ModuleMetadata.Name should be the canonical module name, not the release name")
}

func TestTransformerContext_ToMap(t *testing.T) {
	modMeta := &module.ModuleMetadata{
		Name:    "my-module",
		FQN:     "example.com/modules@v0#MyModule",
		Version: "2.0.0",
		Labels:  map[string]string{"team": "platform"},
	}
	relMeta := &modulerelease.ReleaseMetadata{
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

func TestNewTransformerContext_PropagatesAnnotations(t *testing.T) {
	rel := &modulerelease.ModuleRelease{
		Metadata: &modulerelease.ReleaseMetadata{
			Name:      "my-module",
			Namespace: "default",
			Labels:    map[string]string{},
		},
		Module: module.Module{
			Metadata: &module.ModuleMetadata{
				Name:    "my-module",
				Version: "1.0.0",
				Labels:  map[string]string{},
			},
		},
	}

	comp := &component.Component{
		Metadata: &component.ComponentMetadata{
			Name:   "volumes-component",
			Labels: map[string]string{},
			Annotations: map[string]string{
				"transformer.opmodel.dev/list-output": "true",
			},
		},
		Resources: map[string]cue.Value{},
		Traits:    map[string]cue.Value{},
	}

	ctx := NewTransformerContext(rel, comp)

	assert.Equal(t, "true", ctx.ComponentMetadata.Annotations["transformer.opmodel.dev/list-output"])
}

func TestNewTransformerContext_EmptyAnnotations(t *testing.T) {
	rel := &modulerelease.ModuleRelease{
		Metadata: &modulerelease.ReleaseMetadata{
			Name:      "my-module",
			Namespace: "default",
			Labels:    map[string]string{},
		},
		Module: module.Module{
			Metadata: &module.ModuleMetadata{
				Name:    "my-module",
				Version: "1.0.0",
				Labels:  map[string]string{},
			},
		},
	}

	comp := &component.Component{
		Metadata: &component.ComponentMetadata{
			Name:        "simple-component",
			Labels:      map[string]string{},
			Annotations: map[string]string{},
		},
		Resources: map[string]cue.Value{},
		Traits:    map[string]cue.Value{},
	}

	ctx := NewTransformerContext(rel, comp)

	assert.Empty(t, ctx.ComponentMetadata.Annotations)
}

func TestTransformerContext_ToMap_WithAnnotations(t *testing.T) {
	ctx := &TransformerContext{
		Name:      "release-name",
		Namespace: "production",
		ModuleMetadata: &module.ModuleMetadata{
			Name:    "my-module",
			Version: "2.0.0",
		},
		ReleaseMetadata: &modulerelease.ReleaseMetadata{
			Name:      "release-name",
			Namespace: "production",
		},
		ComponentMetadata: &TransformerComponentMetadata{
			Name: "volumes-component",
			Annotations: map[string]string{
				"transformer.opmodel.dev/list-output": "true",
			},
		},
	}

	m := ctx.ToMap()

	componentMetadata := m["#componentMetadata"].(map[string]any)
	annotations, ok := componentMetadata["annotations"]
	assert.True(t, ok, "annotations should be present in component metadata map")
	annotationsMap := annotations.(map[string]string)
	assert.Equal(t, "true", annotationsMap["transformer.opmodel.dev/list-output"])
}

func TestTransformerContext_ToMap_WithoutAnnotations(t *testing.T) {
	ctx := &TransformerContext{
		Name:      "release-name",
		Namespace: "production",
		ModuleMetadata: &module.ModuleMetadata{
			Name:    "my-module",
			Version: "2.0.0",
		},
		ReleaseMetadata: &modulerelease.ReleaseMetadata{
			Name:      "release-name",
			Namespace: "production",
		},
		ComponentMetadata: &TransformerComponentMetadata{
			Name:        "simple-component",
			Annotations: map[string]string{},
		},
	}

	m := ctx.ToMap()

	componentMetadata := m["#componentMetadata"].(map[string]any)
	_, ok := componentMetadata["annotations"]
	assert.False(t, ok, "annotations should not be present when empty")
}
