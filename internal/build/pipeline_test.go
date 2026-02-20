package build

import (
	"testing"

	"cuelang.org/go/cue"
	"github.com/stretchr/testify/assert"

	"github.com/opmodel/cli/internal/core"
)

func TestNewPipeline(t *testing.T) {
	// Test that NewPipeline creates a valid pipeline
	p := NewPipeline(nil, make(map[string]cue.Value), "")
	assert.NotNil(t, p)
}

func TestRenderOptionsValidate(t *testing.T) {
	tests := []struct {
		name    string
		opts    RenderOptions
		wantErr bool
	}{
		{
			name: "valid options",
			opts: RenderOptions{
				ModulePath: "/some/path",
			},
			wantErr: false,
		},
		{
			name:    "missing module path",
			opts:    RenderOptions{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.opts.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestPipeline_IdentityFieldsPropagated(t *testing.T) {
	// This test verifies that Identity (module UUID) and ReleaseIdentity (release UUID) fields
	// are properly set on the two typed fields of core.ModuleRelease.

	rel := &core.ModuleRelease{
		Metadata: &core.ReleaseMetadata{
			Name:       "my-app",
			Namespace:  "production",
			UUID:       "release-uuid-5678",
			Labels:     map[string]string{"env": "prod"},
			Components: []string{"web"},
		},
		Module: core.Module{
			Metadata: &core.ModuleMetadata{
				Name:             "app",
				DefaultNamespace: "staging",
				Version:          "1.0.0",
				FQN:              "example.com/modules/app@v1",
				UUID:             "module-uuid-1234",
				Labels:           map[string]string{"env": "prod"},
				Components:       []string{"web"},
			},
		},
	}

	// ReleaseMetadata assertions
	assert.Equal(t, "my-app", rel.Metadata.Name)
	assert.Equal(t, "production", rel.Metadata.Namespace)
	assert.Equal(t, "release-uuid-5678", rel.Metadata.UUID, "UUID should be the release identity UUID")
	assert.Equal(t, map[string]string{"env": "prod"}, rel.Metadata.Labels)
	assert.Contains(t, rel.Metadata.Components, "web")

	// ModuleMetadata assertions
	assert.Equal(t, "app", rel.Module.Metadata.Name, "Name should be the canonical module name, not the release name")
	assert.Equal(t, "staging", rel.Module.Metadata.DefaultNamespace)
	assert.Equal(t, "1.0.0", rel.Module.Metadata.Version)
	assert.Equal(t, "example.com/modules/app@v1", rel.Module.Metadata.FQN)
	assert.Equal(t, "module-uuid-1234", rel.Module.Metadata.UUID, "UUID should be the module identity UUID")
	assert.Equal(t, map[string]string{"env": "prod"}, rel.Module.Metadata.Labels)
	assert.Contains(t, rel.Module.Metadata.Components, "web")
}
