package mod

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/opmodel/cli/internal/cmdutil"
	"github.com/opmodel/cli/internal/core"
	"github.com/opmodel/cli/internal/core/module"
	"github.com/opmodel/cli/internal/core/modulerelease"
	"github.com/opmodel/cli/internal/core/transformer"
	"github.com/opmodel/cli/internal/output"
	"github.com/opmodel/cli/internal/pipeline"
)

// buildTestResult constructs a minimal *pipeline.RenderResult suitable for
// testing output formatting without requiring a registry or real module render.
func buildTestResult() *pipeline.RenderResult {
	return &pipeline.RenderResult{
		Release: modulerelease.ReleaseMetadata{
			Name:       "test-release",
			Namespace:  "default",
			Components: []string{"web"},
		},
		Module: module.ModuleMetadata{
			Version: "1.0.0",
		},
		MatchPlan: transformer.MatchPlan{
			Matches: map[string][]transformer.TransformerMatchOld{
				"web": {
					{
						TransformerFQN: "test#DeploymentTransformer",
						Reason:         "Matched: requiredResources[opmodel.dev/resources/Container@v0]",
					},
					{
						TransformerFQN: "test#ServiceTransformer",
						Reason:         "Matched: requiredResources[opmodel.dev/resources/Container@v0], requiredTraits[opmodel.dev/traits/Expose@v0]",
					},
				},
			},
			Unmatched: nil,
		},
		Resources: []*core.Resource{
			{
				Object: &unstructured.Unstructured{Object: map[string]interface{}{
					"apiVersion": "apps/v1",
					"kind":       "Deployment",
					"metadata":   map[string]interface{}{"name": "test-release", "namespace": "default"},
				}},
				Component:   "web",
				Transformer: "test#DeploymentTransformer",
			},
			{
				Object: &unstructured.Unstructured{Object: map[string]interface{}{
					"apiVersion": "v1",
					"kind":       "Service",
					"metadata":   map[string]interface{}{"name": "test-release", "namespace": "default"},
				}},
				Component:   "web",
				Transformer: "test#ServiceTransformer",
			},
		},
		Errors:   []error{},
		Warnings: []string{},
	}
}

func TestVerboseOutput_TransformerMatches(t *testing.T) {
	result := buildTestResult()

	// Verify the MatchPlan has the expected structure.
	assert.NotEmpty(t, result.MatchPlan.Matches, "should have at least one component with matches")
	assert.Contains(t, result.MatchPlan.Matches, "web", "should have a 'web' component")
	assert.NotEmpty(t, result.MatchPlan.Matches["web"], "web component should have transformer matches")

	// Verify each match has a reason.
	for _, match := range result.MatchPlan.Matches["web"] {
		assert.NotEmpty(t, match.Reason, "each match should have a reason string")
	}

	t.Run("default output shows compact matches", func(t *testing.T) {
		var buf bytes.Buffer
		output.SetupLogging(output.LogConfig{})
		output.SetLogWriter(&buf)

		cmdutil.WriteTransformerMatches(result)

		got := buf.String()

		assert.Contains(t, got, "▸", "should contain bullet character")
		assert.Contains(t, got, "web", "should contain component name")
		assert.Contains(t, got, "←", "should contain arrow")
		assert.Contains(t, got, "Transformer", "should contain transformer name")

		assert.NotContains(t, got, "Matched:", "default output should not contain match reasons")
		assert.NotContains(t, got, "module", "default output should not contain module metadata header")
	})

	t.Run("verbose output shows reasons and metadata", func(t *testing.T) {
		var buf bytes.Buffer
		output.SetupLogging(output.LogConfig{Verbose: true})
		output.SetLogWriter(&buf)

		cmdutil.WriteVerboseMatchLog(result)

		outputStr := buf.String()

		assert.Contains(t, outputStr, "release", "verbose should contain release metadata")
		assert.Contains(t, outputStr, "namespace=default", "verbose should show namespace")
		assert.Contains(t, outputStr, "version=1.0.0", "verbose should show version")
		assert.Contains(t, outputStr, "components=web", "verbose should show component list")

		assert.Contains(t, outputStr, "▸", "should contain bullet")
		assert.Contains(t, outputStr, "web", "should contain component")
		assert.Contains(t, outputStr, "←", "should contain arrow")

		assert.Contains(t, outputStr, "Matched:", "verbose should contain match reason prefix")
		assert.Contains(t, outputStr, "requiredResources", "verbose should contain resource requirement details")

		assert.Contains(t, outputStr, "r:", "verbose should contain resource lines")
		assert.Contains(t, outputStr, "valid", "verbose should show resource status")
	})
}
