package build

import (
	"testing"

	"cuelang.org/go/cue"
	"github.com/stretchr/testify/assert"
)

func TestNewTransformerContext_PropagatesAnnotations(t *testing.T) {
	release := &BuiltRelease{
		Metadata: ReleaseMetadata{
			Name:      "my-module",
			Namespace: "default",
			Version:   "1.0.0",
			Labels:    map[string]string{},
		},
	}

	component := &LoadedComponent{
		Name:   "volumes-component",
		Labels: map[string]string{},
		Annotations: map[string]string{
			"transformer.opmodel.dev/list-output": "true",
		},
		Resources: map[string]cue.Value{},
		Traits:    map[string]cue.Value{},
	}

	ctx := NewTransformerContext(release, component)

	assert.Equal(t, "true", ctx.ComponentMetadata.Annotations["transformer.opmodel.dev/list-output"])
}

func TestNewTransformerContext_EmptyAnnotations(t *testing.T) {
	release := &BuiltRelease{
		Metadata: ReleaseMetadata{
			Name:      "my-module",
			Namespace: "default",
			Version:   "1.0.0",
			Labels:    map[string]string{},
		},
	}

	component := &LoadedComponent{
		Name:        "simple-component",
		Labels:      map[string]string{},
		Annotations: map[string]string{},
		Resources:   map[string]cue.Value{},
		Traits:      map[string]cue.Value{},
	}

	ctx := NewTransformerContext(release, component)

	assert.Empty(t, ctx.ComponentMetadata.Annotations)
}

func TestTransformerContext_ToMap_WithAnnotations(t *testing.T) {
	ctx := &TransformerContext{
		Name:      "release-name",
		Namespace: "production",
		ModuleReleaseMetadata: &TransformerModuleReleaseMetadata{
			Name:      "my-module",
			Namespace: "production",
			Version:   "2.0.0",
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
		ModuleReleaseMetadata: &TransformerModuleReleaseMetadata{
			Name:      "my-module",
			Namespace: "production",
			Version:   "2.0.0",
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
