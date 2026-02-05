package build

import (
	"testing"

	"cuelang.org/go/cue/cuecontext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReleaseBuilder_Build(t *testing.T) {
	ctx := cuecontext.New()
	builder := NewReleaseBuilder(ctx, "")

	tests := []struct {
		name          string
		cue           string
		opts          ReleaseOptions
		expectedCount int
		expectError   bool
		errorContains string
	}{
		{
			name: "successful build with #config pattern",
			cue: `{
				#config: {
					image: string
					replicas: int
				}
				values: {
					image: "nginx:1.25"
					replicas: 2
				}
				#components: {
					web: {
						spec: {
							container: {
								image: #config.image
							}
							replicas: #config.replicas
						}
					}
				}
			}`,
			opts: ReleaseOptions{
				Name:      "test-release",
				Namespace: "default",
			},
			expectedCount: 1,
		},
		{
			name: "missing values field",
			cue: `{
				#config: { image: string }
				#components: { web: { spec: { image: #config.image } } }
			}`,
			opts: ReleaseOptions{
				Name:      "test-release",
				Namespace: "default",
			},
			expectError:   true,
			errorContains: "missing 'values' field",
		},
		{
			name: "non-concrete values",
			cue: `{
				#config: { image: string }
				values: { image: string }  // Not concrete!
				#components: { web: { spec: { image: #config.image } } }
			}`,
			opts: ReleaseOptions{
				Name:      "test-release",
				Namespace: "default",
			},
			expectError:   true,
			errorContains: "non-concrete values",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			moduleValue := ctx.CompileString(tt.cue)
			require.NoError(t, moduleValue.Err())

			release, err := builder.Build(moduleValue, tt.opts)

			if tt.expectError {
				require.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, release)
			assert.Len(t, release.Components, tt.expectedCount)
			assert.Equal(t, tt.opts.Name, release.Metadata.Name)
			assert.Equal(t, tt.opts.Namespace, release.Metadata.Namespace)
		})
	}
}

func TestReleaseBuilder_ExtractMetadata(t *testing.T) {
	ctx := cuecontext.New()
	builder := NewReleaseBuilder(ctx, "")

	moduleValue := ctx.CompileString(`{
		metadata: {
			name: "test-module"
			version: "1.0.0"
			fqn: "example.com/test@v0#test-module"
			labels: { "env": "prod" }
		}
		#config: { image: string }
		values: { image: "nginx:1.25" }
		#components: {}
	}`)
	require.NoError(t, moduleValue.Err())

	release, err := builder.Build(moduleValue, ReleaseOptions{
		Name:      "my-release",
		Namespace: "production",
	})
	require.NoError(t, err)

	assert.Equal(t, "my-release", release.Metadata.Name)
	assert.Equal(t, "production", release.Metadata.Namespace)
	assert.Equal(t, "1.0.0", release.Metadata.Version)
}
