package release_test

import (
	"testing"

	"cuelang.org/go/cue/cuecontext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/opmodel/cli/internal/core"
)

func TestExtractComponent_WithAnnotations(t *testing.T) {
	ctx := cuecontext.New()

	tests := []struct {
		name                string
		cue                 string
		expectedAnnotations map[string]string
	}{
		{
			name: "component with list-output annotation",
			cue: `{
				comp: {
					metadata: {
						name: "volumes-component"
						annotations: {
							"transformer.opmodel.dev/list-output": "true"
						}
					}
					#resources: { r: {} }
				}
			}`,
			expectedAnnotations: map[string]string{
				"transformer.opmodel.dev/list-output": "true",
			},
		},
		{
			name: "component without annotations",
			cue: `{
				comp: {
					metadata: {
						name: "simple-component"
					}
					#resources: { r: {} }
				}
			}`,
			expectedAnnotations: map[string]string{},
		},
		{
			name: "component with false annotation",
			cue: `{
				comp: {
					metadata: {
						name: "single-output-component"
						annotations: {
							"transformer.opmodel.dev/list-output": "false"
						}
					}
					#resources: { r: {} }
				}
			}`,
			expectedAnnotations: map[string]string{
				"transformer.opmodel.dev/list-output": "false",
			},
		},
		{
			name: "component with string annotation",
			cue: `{
				comp: {
					metadata: {
						name: "annotated-component"
						annotations: {
							"example.com/owner": "platform-team"
						}
					}
					#resources: { r: {} }
				}
			}`,
			expectedAnnotations: map[string]string{
				"example.com/owner": "platform-team",
			},
		},
		{
			name: "component with multiple annotations",
			cue: `{
				comp: {
					metadata: {
						name: "multi-annotated"
						annotations: {
							"transformer.opmodel.dev/list-output": "true"
							"example.com/owner": "platform-team"
						}
					}
					#resources: { r: {} }
				}
			}`,
			expectedAnnotations: map[string]string{
				"transformer.opmodel.dev/list-output": "true",
				"example.com/owner":                   "platform-team",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value := ctx.CompileString(tt.cue)
			require.NoError(t, value.Err())

			components, err := core.ExtractComponents(value)
			require.NoError(t, err)
			require.Len(t, components, 1)

			comp := components["comp"]
			require.NotNil(t, comp)
			require.NotNil(t, comp.Metadata)
			assert.NotNil(t, comp.Metadata.Annotations, "Annotations should not be nil")
			assert.Equal(t, tt.expectedAnnotations, comp.Metadata.Annotations)
		})
	}
}
