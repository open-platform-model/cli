package kubernetes

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// --- 7.6: Tests for output format selection ---

func TestFormatStatus_Table(t *testing.T) {
	result := &StatusResult{
		InstanceName:    "my-app",
		Owner:           "cli",
		Namespace:       "default",
		AggregateStatus: HealthReady,
		Summary:         statusSummary{Total: 2, Ready: 2},
		Resources: []resourceHealth{
			{Kind: "Deployment", Name: "web", Namespace: "default", Status: HealthReady, Age: "5m"},
			{Kind: "ConfigMap", Name: "config", Namespace: "default", Status: HealthApplied, Age: "5m"},
		},
	}

	formatted, err := FormatStatus(result, "table")
	require.NoError(t, err)
	assert.Contains(t, formatted, "KIND")
	assert.Contains(t, formatted, "NAME")
	assert.Contains(t, formatted, "Deployment")
	assert.Contains(t, formatted, "web")
	assert.Contains(t, formatted, "ConfigMap")
	assert.Contains(t, formatted, "Applied")
}

func TestGetInstanceStatus_PassiveKindsReportApplied(t *testing.T) {
	deploy := makeWorkload("Deployment", 1, map[string]interface{}{"replicas": int64(1)}, map[string]interface{}{
		"observedGeneration": int64(1), "replicas": int64(1), "updatedReplicas": int64(1), "availableReplicas": int64(1),
	})
	res, err := GetInstanceStatus(context.Background(), nil, StatusOptions{
		InstanceName: "my-app",
		InventoryLive: []*unstructured.Unstructured{
			makeResource("ClusterRole", nil),
			makeResource("ServiceAccount", nil),
			deploy,
		},
	})
	require.NoError(t, err)

	statuses := map[string]HealthStatus{}
	for _, r := range res.Resources {
		statuses[r.Kind] = r.Status
	}
	assert.Equal(t, HealthApplied, statuses["ClusterRole"])
	assert.Equal(t, HealthApplied, statuses["ServiceAccount"])
	assert.Equal(t, HealthReady, statuses["Deployment"])

	// Applied counts as healthy: the instance is Ready and nothing is NotReady.
	assert.Equal(t, HealthReady, res.AggregateStatus)
	assert.Equal(t, 3, res.Summary.Ready)
	assert.Equal(t, 0, res.Summary.NotReady)
}

func TestAggregateStatus_AppliedIsHealthy(t *testing.T) {
	assert.Equal(t, HealthReady, aggregateStatus([]ResourceNode{
		{Kind: "ClusterRole", Status: HealthApplied},
		{Kind: "Deployment", Status: HealthReady},
	}, 2))
	assert.Equal(t, HealthNotReady, aggregateStatus([]ResourceNode{
		{Kind: "ClusterRole", Status: HealthApplied},
		{Kind: "Deployment", Status: HealthNotReady},
	}, 2))
}

func TestFormatStatus_JSON(t *testing.T) {
	result := &StatusResult{
		InstanceName:    "my-app",
		Owner:           "cli",
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
		InstanceName:    "my-app",
		Owner:           "cli",
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
		InstanceName:    "my-app",
		Owner:           "cli",
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
	assert.Contains(t, out, "Instance:")
	assert.Contains(t, out, "Owner:")
	assert.Contains(t, out, "my-app")
	assert.Contains(t, out, "2 total")
}

func TestFormatStatusTable_WideColumns(t *testing.T) {
	result := &StatusResult{
		InstanceName:    "my-app",
		Owner:           "cli",
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
		InstanceName:    "my-app",
		Owner:           "cli",
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
		InstanceName:    "my-app",
		Owner:           "cli",
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

func TestFormatStatusHeader_OperatorWarning(t *testing.T) {
	result := &StatusResult{
		InstanceName:    "my-app",
		Owner:           "operator",
		Namespace:       "production",
		AggregateStatus: HealthReady,
		Summary:         statusSummary{Total: 1, Ready: 1},
	}

	out := FormatStatusTable(result)
	assert.Contains(t, out, "Owner:")
	assert.Contains(t, out, "operator-managed instance")
}

func TestFormatStatusHeader_CLIOwnedNoWarning(t *testing.T) {
	result := &StatusResult{
		InstanceName:    "my-app",
		Owner:           "cli",
		Namespace:       "production",
		AggregateStatus: HealthReady,
		Summary:         statusSummary{Total: 1, Ready: 1},
	}

	out := FormatStatusTable(result)
	assert.NotContains(t, out, "operator-managed instance")
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

// A tracked resource discovery could not read is an Unknown row: not healthy,
// so the instance is NotReady and the summary counts it as not ready.
func TestGetInstanceStatus_UnreadableIsUnknown(t *testing.T) {
	forbidden := apierrors.NewForbidden(schema.GroupResource{Resource: "secrets"}, "creds", errors.New("denied"))
	unreadable := []UnreadableResource{{Kind: "Secret", Namespace: "default", Name: "creds", Err: forbidden}}

	res, err := GetInstanceStatus(context.Background(), nil, StatusOptions{
		InstanceName:        "my-app",
		InventoryLive:       []*unstructured.Unstructured{makeResource("ConfigMap", nil)},
		UnreadableResources: unreadable,
	})
	require.NoError(t, err)
	require.Len(t, res.Resources, 2)
	row := res.Resources[1]
	assert.Equal(t, "Secret", row.Kind)
	assert.Equal(t, "creds", row.Name)
	assert.Equal(t, "default", row.Namespace)
	assert.Equal(t, HealthUnknown, row.Status)
	assert.Equal(t, "<unknown>", row.Age)
	assert.Equal(t, HealthNotReady, res.AggregateStatus)
	assert.Equal(t, statusSummary{Total: 2, Ready: 1, NotReady: 1}, res.Summary)

	// Only unreadable entries: still a status, not "no resources found".
	res, err = GetInstanceStatus(context.Background(), nil, StatusOptions{InstanceName: "my-app", UnreadableResources: unreadable})
	require.NoError(t, err)
	assert.Equal(t, HealthNotReady, res.AggregateStatus)
	assert.Equal(t, statusSummary{Total: 1, NotReady: 1}, res.Summary)
}
