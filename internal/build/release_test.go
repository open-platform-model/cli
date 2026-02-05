package build

import (
	"testing"

	"cuelang.org/go/cue/cuecontext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsModuleRelease(t *testing.T) {
	ctx := cuecontext.New()
	loader := NewModuleLoader()

	tests := []struct {
		name     string
		cue      string
		expected bool
	}{
		{
			name: "module with #components is not release",
			cue: `{
				#components: {
					web: { name: "web" }
				}
			}`,
			expected: false,
		},
		{
			name: "release with concrete components",
			cue: `{
				components: {
					web: { name: "web" }
				}
			}`,
			expected: true,
		},
		{
			name: "empty module is not release",
			cue:      `{}`,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value := ctx.CompileString(tt.cue)
			require.NoError(t, value.Err())

			result := loader.isModuleRelease(value)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractComponentsFromModule(t *testing.T) {
	ctx := cuecontext.New()
	loader := NewModuleLoader()

	tests := []struct {
		name          string
		cue           string
		isRelease     bool
		expectedCount int
		expectError   bool
	}{
		{
			name: "extract from #components",
			cue: `{
				#components: {
					web: { name: "web" }
					api: { name: "api" }
				}
			}`,
			isRelease:     false,
			expectedCount: 2,
		},
		{
			name: "extract from components",
			cue: `{
				components: {
					web: { name: "web" }
				}
			}`,
			isRelease:     true,
			expectedCount: 1,
		},
		{
			name:          "empty module",
			cue:           `{}`,
			isRelease:     false,
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value := ctx.CompileString(tt.cue)
			require.NoError(t, value.Err())

			components, err := loader.extractComponentsFromModule(value, tt.isRelease)
			if tt.expectError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Len(t, components, tt.expectedCount)
		})
	}
}

func TestResolveNamespace(t *testing.T) {
	loader := NewModuleLoader()

	tests := []struct {
		name             string
		flagValue        string
		defaultNamespace string
		expected         string
	}{
		{
			name:             "flag takes precedence",
			flagValue:        "production",
			defaultNamespace: "dev",
			expected:         "production",
		},
		{
			name:             "defaultNamespace when no flag",
			flagValue:        "",
			defaultNamespace: "dev",
			expected:         "dev",
		},
		{
			name:             "empty when neither set",
			flagValue:        "",
			defaultNamespace: "",
			expected:         "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := loader.resolveNamespace(tt.flagValue, tt.defaultNamespace)
			assert.Equal(t, tt.expected, result)
		})
	}
}
