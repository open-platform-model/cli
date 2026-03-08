package mod

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/opmodel/cli/internal/cmdutil"
	"github.com/opmodel/cli/internal/output"
	"github.com/opmodel/cli/pkg/engine"
	pkgmodule "github.com/opmodel/cli/pkg/module"
	"github.com/opmodel/cli/pkg/modulerelease"
)

// buildTestResult constructs a minimal *cmdutil.RenderResult suitable for
// testing output formatting without requiring a registry or real module render.
func buildTestResult() *cmdutil.RenderResult {
	return &cmdutil.RenderResult{
		Release: modulerelease.ReleaseMetadata{
			Name:      "test-release",
			Namespace: "default",
		},
		Module: pkgmodule.ModuleMetadata{
			Version: "1.0.0",
		},
		Components: []engine.ComponentSummary{
			{
				Name:         "web",
				Labels:       map[string]string{"core.opmodel.dev/workload-type": "stateless"},
				ResourceFQNs: []string{"opmodel.dev/resources/workload/container@v1"},
				TraitFQNs:    []string{"opmodel.dev/traits/network/expose@v1"},
			},
		},
		MatchPlan: &engine.MatchPlan{
			Matches: map[string]map[string]engine.MatchResult{
				"web": {
					"test#DeploymentTransformer": {Matched: true},
					"test#ServiceTransformer":    {Matched: true},
				},
			},
			Unmatched: nil,
		},
		Resources: []*unstructured.Unstructured{
			{Object: map[string]interface{}{
				"apiVersion": "apps/v1",
				"kind":       "Deployment",
				"metadata":   map[string]interface{}{"name": "test-release", "namespace": "default"},
			}},
			{Object: map[string]interface{}{
				"apiVersion": "v1",
				"kind":       "Service",
				"metadata":   map[string]interface{}{"name": "test-release", "namespace": "default"},
			}},
		},
		Warnings: []string{},
	}
}

func TestVerboseOutput_TransformerMatches(t *testing.T) {
	result := buildTestResult()

	// Verify the MatchPlan has the expected structure.
	assert.NotEmpty(t, result.MatchPlan.Matches, "should have at least one match")

	// Verify matched pairs are correctly reported.
	pairs := result.MatchPlan.MatchedPairs()
	assert.Len(t, pairs, 2, "should have 2 matched pairs")
	for _, p := range pairs {
		assert.Equal(t, "web", p.ComponentName, "each pair should have component name")
		assert.NotEmpty(t, p.TransformerFQN, "each pair should have transformer FQN")
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

		assert.NotContains(t, got, "module", "default output should not contain module metadata header")
	})

	t.Run("verbose output shows component details and metadata", func(t *testing.T) {
		var buf bytes.Buffer
		output.SetupLogging(output.LogConfig{Verbose: true})
		output.SetLogWriter(&buf)

		cmdutil.WriteVerboseMatchLog(result)

		outputStr := buf.String()

		assert.Contains(t, outputStr, "release", "verbose should contain release metadata")
		assert.Contains(t, outputStr, "namespace=default", "verbose should show namespace")
		assert.Contains(t, outputStr, "version=1.0.0", "verbose should show version")
		assert.Contains(t, outputStr, "component: web", "verbose should show component name")
		assert.Contains(t, outputStr, "container", "verbose should show component resources")
		assert.Contains(t, outputStr, "expose", "verbose should show component traits")

		assert.Contains(t, outputStr, "▸", "should contain bullet")
		assert.Contains(t, outputStr, "web", "should contain component")
		assert.Contains(t, outputStr, "←", "should contain arrow")

		assert.Contains(t, outputStr, "r:", "verbose should contain resource lines")
		assert.Contains(t, outputStr, "valid", "verbose should show resource status")
	})
}
