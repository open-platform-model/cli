package transform

import (
	"testing"

	"cuelang.org/go/cue"
	"github.com/stretchr/testify/assert"

	"github.com/opmodel/cli/internal/build/component"
	"github.com/opmodel/cli/internal/build/module"
	"github.com/opmodel/cli/internal/build/release"
)

func TestNewTransformerContext_PropagatesAnnotations(t *testing.T) {
	rel := &release.BuiltRelease{
		ReleaseMetadata: release.ReleaseMetadata{
			Name:      "my-module",
			Namespace: "default",
			Labels:    map[string]string{},
		},
		ModuleMetadata: module.ModuleMetadata{
			Name:    "my-module",
			Version: "1.0.0",
			Labels:  map[string]string{},
		},
	}

	comp := &component.Component{
		Name:   "volumes-component",
		Labels: map[string]string{},
		Annotations: map[string]string{
			"transformer.opmodel.dev/list-output": "true",
		},
		Resources: map[string]cue.Value{},
		Traits:    map[string]cue.Value{},
	}

	ctx := NewTransformerContext(rel, comp)

	assert.Equal(t, "true", ctx.ComponentMetadata.Annotations["transformer.opmodel.dev/list-output"])
}

func TestNewTransformerContext_EmptyAnnotations(t *testing.T) {
	rel := &release.BuiltRelease{
		ReleaseMetadata: release.ReleaseMetadata{
			Name:      "my-module",
			Namespace: "default",
			Labels:    map[string]string{},
		},
		ModuleMetadata: module.ModuleMetadata{
			Name:    "my-module",
			Version: "1.0.0",
			Labels:  map[string]string{},
		},
	}

	comp := &component.Component{
		Name:        "simple-component",
		Labels:      map[string]string{},
		Annotations: map[string]string{},
		Resources:   map[string]cue.Value{},
		Traits:      map[string]cue.Value{},
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
		ReleaseMetadata: &release.ReleaseMetadata{
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
		ReleaseMetadata: &release.ReleaseMetadata{
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
