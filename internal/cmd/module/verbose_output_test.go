package modulecmd

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/open-platform-model/library/opm/kernel"

	"github.com/open-platform-model/cli/internal/output"
	"github.com/open-platform-model/cli/internal/workflow/render"
	"github.com/open-platform-model/cli/pkg/module"
)

// buildTestResult constructs a minimal *render.Result suitable for
// testing output formatting without requiring a registry or real module render.
func buildTestResult() *render.Result {
	return &render.Result{
		Instance: module.InstanceMetadata{
			Name:      "test-instance",
			Namespace: "default",
		},
		Module: module.ModuleMetadata{
			Version: "1.0.0",
		},
		Pairs: []kernel.RenderPair{
			{Component: "web", Transformer: "test#DeploymentTransformer"},
			{Component: "web", Transformer: "test#ServiceTransformer"},
		},
		Resources: []*unstructured.Unstructured{
			{Object: map[string]interface{}{
				"apiVersion": "apps/v1",
				"kind":       "Deployment",
				"metadata":   map[string]interface{}{"name": "test-instance", "namespace": "default"},
			}},
			{Object: map[string]interface{}{
				"apiVersion": "v1",
				"kind":       "Service",
				"metadata":   map[string]interface{}{"name": "test-instance", "namespace": "default"},
			}},
		},
		Warnings: []string{},
	}
}

func TestVerboseOutput_TransformerMatches(t *testing.T) {
	result := buildTestResult()

	// Verify the pair set has the expected structure.
	assert.Len(t, result.Pairs, 2, "should have 2 matched pairs")
	for _, p := range result.Pairs {
		assert.Equal(t, "web", p.Component, "each pair should have component name")
		assert.NotEmpty(t, p.Transformer, "each pair should have transformer FQN")
	}

	t.Run("default output shows compact matches", func(t *testing.T) {
		var buf bytes.Buffer
		output.SetupLogging(output.LogConfig{})
		output.SetLogWriter(&buf)

		render.WriteTransformerMatchesForTest(result)

		got := buf.String()

		assert.Contains(t, got, "▸", "should contain bullet character")
		assert.Contains(t, got, "web", "should contain component name")
		assert.Contains(t, got, "←", "should contain arrow")
		assert.Contains(t, got, "Transformer", "should contain transformer name")

		assert.NotContains(t, got, "module", "default output should not contain module metadata header")
	})

	t.Run("verbose output shows pairs, metadata and resources", func(t *testing.T) {
		var buf bytes.Buffer
		output.SetupLogging(output.LogConfig{Verbose: true})
		output.SetLogWriter(&buf)

		render.WriteVerboseMatchLogForTest(result)

		outputStr := buf.String()

		assert.Contains(t, outputStr, "instance", "verbose should contain instance metadata")
		assert.Contains(t, outputStr, "namespace=default", "verbose should show namespace")
		assert.Contains(t, outputStr, "version=1.0.0", "verbose should show version")

		assert.Contains(t, outputStr, "▸", "should contain bullet")
		assert.Contains(t, outputStr, "web", "should contain component")
		assert.Contains(t, outputStr, "←", "should contain arrow")
		assert.Contains(t, outputStr, "DeploymentTransformer")
		assert.Contains(t, outputStr, "ServiceTransformer")

		assert.Contains(t, outputStr, "r:", "verbose should contain resource lines")
		assert.Contains(t, outputStr, "valid", "verbose should show resource status")
	})
}
