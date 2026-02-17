package kubernetes

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- 7.6: Tests for output format selection ---

func TestFormatStatus_Table(t *testing.T) {
	result := &StatusResult{
		Resources: []resourceHealth{
			{Kind: "Deployment", Name: "web", Namespace: "default", Status: healthReady, Age: "5m"},
			{Kind: "ConfigMap", Name: "config", Namespace: "default", Status: healthReady, Age: "5m"},
		},
		AggregateStatus: healthReady,
	}

	formatted, err := FormatStatus(result, "table")
	require.NoError(t, err)
	assert.Contains(t, formatted, "KIND")
	assert.Contains(t, formatted, "NAME")
	assert.Contains(t, formatted, "Deployment")
	assert.Contains(t, formatted, "web")
	assert.Contains(t, formatted, "ConfigMap")
}

func TestFormatStatus_JSON(t *testing.T) {
	result := &StatusResult{
		Resources: []resourceHealth{
			{Kind: "Deployment", Name: "web", Namespace: "default", Status: healthReady, Age: "5m"},
		},
		AggregateStatus: healthReady,
	}

	formatted, err := FormatStatus(result, "json")
	require.NoError(t, err)
	assert.Contains(t, formatted, `"kind": "Deployment"`)
	assert.Contains(t, formatted, `"name": "web"`)
	assert.Contains(t, formatted, `"status": "Ready"`)
}

func TestFormatStatus_YAML(t *testing.T) {
	result := &StatusResult{
		Resources: []resourceHealth{
			{Kind: "Deployment", Name: "web", Namespace: "default", Status: healthReady, Age: "5m"},
		},
		AggregateStatus: healthReady,
	}

	formatted, err := FormatStatus(result, "yaml")
	require.NoError(t, err)
	assert.Contains(t, formatted, "kind: Deployment")
	assert.Contains(t, formatted, "name: web")
	assert.Contains(t, formatted, "status: Ready")
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name     string
		seconds  int
		expected string
	}{
		{"30 seconds", 30, "30s"},
		{"5 minutes", 300, "5m"},
		{"2 hours", 7200, "2h"},
		{"1 day", 86400, "1d"},
		{"3 days", 259200, "3d"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			d := (time.Duration(tc.seconds) * time.Second)
			assert.Equal(t, tc.expected, formatDuration(d))
		})
	}
}
