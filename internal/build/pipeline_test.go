package build

import (
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"github.com/stretchr/testify/assert"

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
	// are properly propagated from release.Metadata into the two projection types.

	rel := &BuiltRelease{
		Components: map[string]*LoadedComponent{
			"web": {Name: "web"},
		},
		Metadata: release.Metadata{
			Name:            "my-app",
			Namespace:       "production",
			Version:         "1.0.0",
			FQN:             "example.com/modules/app@v1",
			Labels:          map[string]string{"env": "prod"},
			Identity:        "module-uuid-1234",
			ReleaseIdentity: "release-uuid-5678",
		},
	}

	relMeta := rel.ToReleaseMetadata()
	modMeta := rel.ToModuleMetadata("app", "staging") // "app" = canonical module name

	// ReleaseMetadata assertions
	assert.Equal(t, "my-app", relMeta.Name)
	assert.Equal(t, "production", relMeta.Namespace)
	assert.Equal(t, "release-uuid-5678", relMeta.UUID, "UUID should be the release identity UUID")
	assert.Equal(t, map[string]string{"env": "prod"}, relMeta.Labels)
	assert.Contains(t, relMeta.Components, "web")

	// ModuleMetadata assertions
	assert.Equal(t, "app", modMeta.Name, "Name should be the canonical module name, not the release name")
	assert.Equal(t, "staging", modMeta.DefaultNamespace)
	assert.Equal(t, "1.0.0", modMeta.Version)
	assert.Equal(t, "example.com/modules/app@v1", modMeta.FQN)
	assert.Equal(t, "module-uuid-1234", modMeta.UUID, "UUID should be the module identity UUID")
	assert.Equal(t, map[string]string{"env": "prod"}, modMeta.Labels)
	assert.Contains(t, modMeta.Components, "web")
}
