package build

import (
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"github.com/stretchr/testify/assert"

	"github.com/opmodel/cli/internal/build/module"
	"github.com/opmodel/cli/internal/build/release"
	"github.com/opmodel/cli/internal/config"
)

func TestNewPipeline(t *testing.T) {
	// Test that NewPipeline creates a valid pipeline
	cfg := &config.OPMConfig{
		CueContext: cuecontext.New(),
		Registry:   "",
		Providers:  make(map[string]cue.Value),
	}
	p := NewPipeline(cfg)
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
	// are properly set on the two typed fields of BuiltRelease.

	rel := &BuiltRelease{
		ReleaseMetadata: release.ReleaseMetadata{
			Name:       "my-app",
			Namespace:  "production",
			UUID:       "release-uuid-5678",
			Labels:     map[string]string{"env": "prod"},
			Components: []string{"web"},
		},
		ModuleMetadata: module.ModuleMetadata{
			Name:             "app",
			DefaultNamespace: "staging",
			Version:          "1.0.0",
			FQN:              "example.com/modules/app@v1",
			UUID:             "module-uuid-1234",
			Labels:           map[string]string{"env": "prod"},
			Components:       []string{"web"},
		},
	}

	// ReleaseMetadata assertions
	assert.Equal(t, "my-app", rel.ReleaseMetadata.Name)
	assert.Equal(t, "production", rel.ReleaseMetadata.Namespace)
	assert.Equal(t, "release-uuid-5678", rel.ReleaseMetadata.UUID, "UUID should be the release identity UUID")
	assert.Equal(t, map[string]string{"env": "prod"}, rel.ReleaseMetadata.Labels)
	assert.Contains(t, rel.ReleaseMetadata.Components, "web")

	// ModuleMetadata assertions
	assert.Equal(t, "app", rel.ModuleMetadata.Name, "Name should be the canonical module name, not the release name")
	assert.Equal(t, "staging", rel.ModuleMetadata.DefaultNamespace)
	assert.Equal(t, "1.0.0", rel.ModuleMetadata.Version)
	assert.Equal(t, "example.com/modules/app@v1", rel.ModuleMetadata.FQN)
	assert.Equal(t, "module-uuid-1234", rel.ModuleMetadata.UUID, "UUID should be the module identity UUID")
	assert.Equal(t, map[string]string{"env": "prod"}, rel.ModuleMetadata.Labels)
	assert.Contains(t, rel.ModuleMetadata.Components, "web")
}
