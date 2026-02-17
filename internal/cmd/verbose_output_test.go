package cmd

import (
	"bytes"
	"context"
	"path/filepath"
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/opmodel/cli/internal/build"
	"github.com/opmodel/cli/internal/cmdutil"
	"github.com/opmodel/cli/internal/config"
	"github.com/opmodel/cli/internal/output"
)

func TestVerboseOutput_TransformerMatches(t *testing.T) {
	ctx := context.Background()
	cueCtx := cuecontext.New()

	// Create an inline provider with transformers that match the test-module fixture
	providerCUE := cueCtx.CompileString(`{
		version: "1.0.0"
		transformers: {
			DeploymentTransformer: {
				requiredResources: { "opmodel.dev/resources/Container@v0": _ }
				#transform: {
					#component: _
					#context: { name: string, namespace: string, ... }
					output: {
						apiVersion: "apps/v1"
						kind: "Deployment"
						metadata: { name: #context.name, namespace: #context.namespace }
						spec: {}
					}
				}
			}
			ServiceTransformer: {
				requiredResources: { "opmodel.dev/resources/Container@v0": _ }
				requiredTraits: { "opmodel.dev/traits/Expose@v0": _ }
				#transform: {
					#component: _
					#context: { name: string, namespace: string, ... }
					output: {
						apiVersion: "v1"
						kind: "Service"
						metadata: { name: #context.name, namespace: #context.namespace }
						spec: {}
					}
				}
			}
		}
	}`)
	require.NoError(t, providerCUE.Err())

	cfg := &config.OPMConfig{
		CueContext: cueCtx,
		Registry:   "",
		Providers:  map[string]cue.Value{"test": providerCUE},
	}

	// Use the existing test-module fixture
	modulePath, err := filepath.Abs(filepath.Join("..", "build", "testdata", "test-module"))
	require.NoError(t, err)

	// Render the module
	pipeline := build.NewPipeline(cfg)
	result, err := pipeline.Render(ctx, build.RenderOptions{
		ModulePath: modulePath,
		Name:       "test-release",
		Namespace:  "default",
		Provider:   "test",
	})
	require.NoError(t, err)
	require.NotNil(t, result)

	// Verify the MatchPlan has matches
	require.NotEmpty(t, result.MatchPlan.Matches, "should have at least one component with matches")
	require.Contains(t, result.MatchPlan.Matches, "web", "test-module should have a 'web' component")
	require.NotEmpty(t, result.MatchPlan.Matches["web"], "web component should have transformer matches")

	// Verify each match has a reason
	for _, match := range result.MatchPlan.Matches["web"] {
		assert.NotEmpty(t, match.Reason, "each match should have a reason string")
	}

	t.Run("default output shows compact matches", func(t *testing.T) {
		// Setup: info-level logging, capture to buffer
		var buf bytes.Buffer
		output.SetupLogging(output.LogConfig{})
		output.SetLogWriter(&buf)

		// Call the default match writer
		cmdutil.WriteTransformerMatches(result)

		got := buf.String()

		// Should contain the transformer match lines
		assert.Contains(t, got, "▸", "should contain bullet character")
		assert.Contains(t, got, "web", "should contain component name")
		assert.Contains(t, got, "←", "should contain arrow")
		assert.Contains(t, got, "Transformer", "should contain transformer name")

		// Should NOT contain the verbose details
		assert.NotContains(t, got, "Matched:", "default output should not contain match reasons")
		assert.NotContains(t, got, "module", "default output should not contain module metadata header")
	})

	t.Run("verbose output shows reasons and metadata", func(t *testing.T) {
		// Setup: debug-level logging, capture to buffer
		var buf bytes.Buffer
		output.SetupLogging(output.LogConfig{Verbose: true})
		output.SetLogWriter(&buf)

		// Call the verbose match writer
		cmdutil.WriteVerboseMatchLog(result)

		outputStr := buf.String()

		// Should contain module metadata
		assert.Contains(t, outputStr, "module", "verbose should contain module metadata")
		assert.Contains(t, outputStr, "namespace=default", "verbose should show namespace")
		assert.Contains(t, outputStr, "version=1.0.0", "verbose should show version")
		assert.Contains(t, outputStr, "components=web", "verbose should show component list")

		// Should contain transformer matches
		assert.Contains(t, outputStr, "▸", "should contain bullet")
		assert.Contains(t, outputStr, "web", "should contain component")
		assert.Contains(t, outputStr, "←", "should contain arrow")

		// Should contain match reasons
		assert.Contains(t, outputStr, "Matched:", "verbose should contain match reason prefix")
		assert.Contains(t, outputStr, "requiredResources", "verbose should contain resource requirement details")

		// Should contain per-resource validation lines
		assert.Contains(t, outputStr, "r:", "verbose should contain resource lines")
		assert.Contains(t, outputStr, "valid", "verbose should show resource status")
	})
}
