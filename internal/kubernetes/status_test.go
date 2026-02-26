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
		ReleaseName:     "my-app",
		Namespace:       "default",
		AggregateStatus: HealthReady,
		Summary:         statusSummary{Total: 2, Ready: 2},
		Resources: []resourceHealth{
			{Kind: "Deployment", Name: "web", Namespace: "default", Status: HealthReady, Age: "5m"},
			{Kind: "ConfigMap", Name: "config", Namespace: "default", Status: HealthReady, Age: "5m"},
		},
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
		ReleaseName:     "my-app",
		Namespace:       "default",
		AggregateStatus: HealthReady,
		Summary:         statusSummary{Total: 1, Ready: 1},
		Resources: []resourceHealth{
			{Kind: "Deployment", Name: "web", Namespace: "default", Status: HealthReady, Age: "5m"},
		},
	}

	formatted, err := FormatStatus(result, "json")
	require.NoError(t, err)
	assert.Contains(t, formatted, `"kind": "Deployment"`)
	assert.Contains(t, formatted, `"name": "web"`)
	assert.Contains(t, formatted, `"status": "Ready"`)
}

func TestFormatStatus_YAML(t *testing.T) {
	result := &StatusResult{
		ReleaseName:     "my-app",
		Namespace:       "default",
		AggregateStatus: HealthReady,
		Summary:         statusSummary{Total: 1, Ready: 1},
		Resources: []resourceHealth{
			{Kind: "Deployment", Name: "web", Namespace: "default", Status: HealthReady, Age: "5m"},
		},
	}

	formatted, err := FormatStatus(result, "yaml")
	require.NoError(t, err)
	assert.Contains(t, formatted, "kind: Deployment")
	assert.Contains(t, formatted, "name: web")
	assert.Contains(t, formatted, "status: Ready")
}

func TestFormatStatusTable_DefaultColumns(t *testing.T) {
	result := &StatusResult{
		ReleaseName:     "my-app",
		Namespace:       "production",
		AggregateStatus: HealthReady,
		Summary:         statusSummary{Total: 2, Ready: 2},
		Resources: []resourceHealth{
			{Kind: "Deployment", Name: "web", Namespace: "production", Component: "server", Status: HealthReady, Age: "5m"},
			{Kind: "ConfigMap", Name: "config", Namespace: "production", Component: "", Status: HealthReady, Age: "5m"},
		},
	}

	out := FormatStatusTable(result)
	assert.Contains(t, out, "KIND")
	assert.Contains(t, out, "COMPONENT")
	assert.Contains(t, out, "STATUS")
	assert.Contains(t, out, "Deployment")
	assert.Contains(t, out, "server")
	assert.Contains(t, out, "Release:")
	assert.Contains(t, out, "my-app")
	assert.Contains(t, out, "2 total")
}

func TestFormatStatusTable_WideColumns(t *testing.T) {
	result := &StatusResult{
		ReleaseName:     "my-app",
		Namespace:       "production",
		AggregateStatus: HealthNotReady,
		Summary:         statusSummary{Total: 1, Ready: 0, NotReady: 1},
		Resources: []resourceHealth{
			{
				Kind: "Deployment", Name: "web", Namespace: "production", Component: "server",
				Status: HealthNotReady, Age: "5m",
				Wide: &wideInfo{Replicas: "1/3", Image: "nginx:1.25"},
			},
		},
	}

	out, err := FormatStatus(result, "wide")
	require.NoError(t, err)
	assert.Contains(t, out, "REPLICAS")
	assert.Contains(t, out, "IMAGE")
	assert.Contains(t, out, "1/3")
	assert.Contains(t, out, "nginx:1.25")
}

func TestFormatStatusTable_VerboseBlocks(t *testing.T) {
	result := &StatusResult{
		ReleaseName:     "my-app",
		Namespace:       "production",
		AggregateStatus: HealthNotReady,
		Summary:         statusSummary{Total: 1, Ready: 0, NotReady: 1},
		Resources: []resourceHealth{
			{
				Kind: "Deployment", Name: "web", Namespace: "production", Component: "server",
				Status: HealthNotReady, Age: "5m",
				Verbose: &verboseInfo{
					Pods: []podInfo{
						{Name: "web-abc-1", Phase: "Running", Ready: false, Reason: "CrashLoopBackOff", Restarts: 5},
					},
				},
			},
		},
	}

	out := FormatStatusTable(result)
	assert.Contains(t, out, "Deployment/web")
	assert.Contains(t, out, "web-abc-1")
	assert.Contains(t, out, "CrashLoopBackOff")
	assert.Contains(t, out, "5 restarts")
}

func TestFormatStatusTable_NotReadySummary(t *testing.T) {
	result := &StatusResult{
		ReleaseName:     "my-app",
		Namespace:       "production",
		AggregateStatus: HealthNotReady,
		Summary:         statusSummary{Total: 3, Ready: 2, NotReady: 1},
		Resources: []resourceHealth{
			{Kind: "Deployment", Name: "web", Namespace: "production", Status: HealthNotReady, Age: "5m"},
			{Kind: "Service", Name: "svc", Namespace: "production", Status: HealthReady, Age: "5m"},
			{Kind: "ConfigMap", Name: "cfg", Namespace: "production", Status: HealthReady, Age: "5m"},
		},
	}

	out := FormatStatusTable(result)
	assert.Contains(t, out, "3 total")
	assert.Contains(t, out, "2 ready")
	assert.Contains(t, out, "1 not ready")
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
			assert.Equal(t, tc.expected, FormatDuration(d))
		})
	}
}
