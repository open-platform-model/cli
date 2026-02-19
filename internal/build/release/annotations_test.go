package release

import (
	"testing"

	"cuelang.org/go/cue/cuecontext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
				metadata: {
					name: "volumes-component"
					annotations: {
						"transformer.opmodel.dev/list-output": "true"
					}
				}
			}`,
			expectedAnnotations: map[string]string{
				"transformer.opmodel.dev/list-output": "true",
			},
		},
		{
			name: "component without annotations",
			cue: `{
				metadata: {
					name: "simple-component"
				}
			}`,
			expectedAnnotations: map[string]string{},
		},
		{
			name: "component with false annotation",
			cue: `{
				metadata: {
					name: "single-output-component"
					annotations: {
						"transformer.opmodel.dev/list-output": "false"
					}
				}
			}`,
			expectedAnnotations: map[string]string{
				"transformer.opmodel.dev/list-output": "false",
			},
		},
		{
			name: "component with string annotation",
			cue: `{
				metadata: {
					name: "annotated-component"
					annotations: {
						"example.com/owner": "platform-team"
					}
				}
			}`,
			expectedAnnotations: map[string]string{
				"example.com/owner": "platform-team",
			},
		},
		{
			name: "component with multiple annotations",
			cue: `{
				metadata: {
					name: "multi-annotated"
					annotations: {
						"transformer.opmodel.dev/list-output": "true"
						"example.com/owner": "platform-team"
					}
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

			comp := extractComponent("test", value)

			assert.NotNil(t, comp.Annotations, "Annotations should not be nil")
			assert.Equal(t, tt.expectedAnnotations, comp.Annotations)
		})
	}
}
