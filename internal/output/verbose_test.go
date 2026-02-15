package output

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWriteVerboseHuman_ResourceLines(t *testing.T) {
	result := &verboseResult{
		Module: verboseModule{
			Name:       "test-module",
			Namespace:  "default",
			Version:    "v1.0.0",
			Components: []string{"app"},
		},
		MatchPlan: verboseMatchPlan{
			Matches: map[string][]verboseMatch{
				"app": {
					{
						Transformer: "test/transformer",
						Reason:      "matched",
					},
				},
			},
		},
		Resources: []verboseResource{
			{
				Kind:        "Deployment",
				Name:        "test-app",
				Namespace:   "default",
				Component:   "app",
				Transformer: "test/transformer",
			},
			{
				Kind:        "Service",
				Name:        "test-svc",
				Namespace:   "default",
				Component:   "app",
				Transformer: "test/transformer",
			},
		},
	}

	var buf bytes.Buffer
	err := writeVerboseHuman(result, &buf)
	assert.NoError(t, err)

	output := buf.String()

	// Verify output contains the "Generated Resources" section
	assert.Contains(t, output, "Generated Resources:")

	// Verify resources are rendered with FormatResourceLine format (r: prefix)
	assert.Contains(t, output, "r:Deployment/default/test-app")
	assert.Contains(t, output, "r:Service/default/test-svc")

	// Verify resources have "valid" status
	assert.Contains(t, output, "valid")
}

func TestWriteVerboseHuman_ClusterScopedResource(t *testing.T) {
	result := &verboseResult{
		Module: verboseModule{
			Name:       "test-module",
			Namespace:  "default",
			Version:    "v1.0.0",
			Components: []string{"rbac"},
		},
		MatchPlan: verboseMatchPlan{
			Matches: map[string][]verboseMatch{
				"rbac": {
					{
						Transformer: "test/transformer",
						Reason:      "matched",
					},
				},
			},
		},
		Resources: []verboseResource{
			{
				Kind:        "ClusterRole",
				Name:        "admin",
				Namespace:   "", // cluster-scoped
				Component:   "rbac",
				Transformer: "test/transformer",
			},
		},
	}

	var buf bytes.Buffer
	err := writeVerboseHuman(result, &buf)
	assert.NoError(t, err)

	output := buf.String()

	// Verify cluster-scoped resource is rendered without namespace
	assert.Contains(t, output, "r:ClusterRole/admin")

	// Should NOT contain the old (cluster-scoped) format
	assert.False(t, strings.Contains(output, "(cluster-scoped)"))
}
