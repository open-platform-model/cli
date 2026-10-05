package kubernetes

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/open-platform-model/library/opm/k8s/health"

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
		AggregateStatus: health.Ready,
		Summary:         statusSummary{Total: 2, Ready: 2},
		Resources: []resourceHealth{
			{Kind: "Deployment", Name: "web", Namespace: "default", Status: health.Ready, Age: "5m"},
			{Kind: "ConfigMap", Name: "config", Namespace: "default", Status: health.Applied, Age: "5m"},
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
			makeResource("ClusterRole"),
			makeResource("ServiceAccount"),
			deploy,
		},
	})
	require.NoError(t, err)

	statuses := map[string]health.Status{}
	for _, r := range res.Resources {
		statuses[r.Kind] = r.Status
	}
	assert.Equal(t, health.Applied, statuses["ClusterRole"])
	assert.Equal(t, health.Applied, statuses["ServiceAccount"])
	assert.Equal(t, health.Ready, statuses["Deployment"])

	// Applied counts as healthy: the instance is Ready and nothing is NotReady.
	assert.Equal(t, health.Ready, res.AggregateStatus)
	assert.Equal(t, 3, res.Summary.Ready)
	assert.Equal(t, 0, res.Summary.NotReady)
}

func TestAggregateStatus_AppliedIsHealthy(t *testing.T) {
	assert.Equal(t, health.Ready, aggregateStatus([]ResourceNode{
		{Kind: "ClusterRole", Status: health.Applied},
		{Kind: "Deployment", Status: health.Ready},
	}))
	assert.Equal(t, health.NotReady, aggregateStatus([]ResourceNode{
		{Kind: "ClusterRole", Status: health.Applied},
		{Kind: "Deployment", Status: health.NotReady},
	}))
}

// A component with no evaluated resource node (depth 0) has nothing to fold
// and is Unknown, whatever its resource count.
func TestAggregateStatus_NothingToFoldIsUnknown(t *testing.T) {
	assert.Equal(t, health.Unknown, aggregateStatus(nil))
}

// These strings are `opm instance status|list|tree -o json|yaml` output.
// They must never change spelling.
func TestHealthStatusStrings(t *testing.T) {
	for want, got := range map[string]health.Status{
		"Ready":    health.Ready,
		"NotReady": health.NotReady,
		"Complete": health.Complete,
		"Unknown":  health.Unknown,
		"Missing":  health.Missing,
		"Applied":  health.Applied,
		"Bound":    health.Bound,
	} {
		assert.Equal(t, want, string(got))
	}
}

// GetInstanceStatus folds every row (live, missing, unreadable) into the
// instance verdict and the summary counts.
func TestGetInstanceStatus_Aggregate(t *testing.T) {
	readyDeploy := func() *unstructured.Unstructured {
		return makeWorkload("Deployment", 1, map[string]interface{}{"replicas": int64(1)}, map[string]interface{}{
			"observedGeneration": int64(1), "replicas": int64(1), "updatedReplicas": int64(1), "availableReplicas": int64(1),
		})
	}
	notReadyDeploy := makeWorkload("Deployment", 1, map[string]interface{}{"replicas": int64(2)}, map[string]interface{}{
		"observedGeneration": int64(1), "replicas": int64(2), "updatedReplicas": int64(1), "availableReplicas": int64(1),
	})
	forbidden := apierrors.NewForbidden(schema.GroupResource{Resource: "secrets"}, "creds", errors.New("denied"))

	tests := []struct {
		name       string
		live       []*unstructured.Unstructured
		missing    []MissingResource
		unreadable []UnreadableResource
		wantAgg    health.Status
		wantRows   []health.Status
		wantSum    statusSummary
	}{
		{
			name:     "all healthy",
			live:     []*unstructured.Unstructured{readyDeploy(), makeResource("ConfigMap")},
			wantAgg:  health.Ready,
			wantRows: []health.Status{health.Ready, health.Applied},
			wantSum:  statusSummary{Total: 2, Ready: 2, NotReady: 0},
		},
		{
			name:     "applied only",
			live:     []*unstructured.Unstructured{makeResource("ConfigMap"), makeResource("ServiceAccount")},
			wantAgg:  health.Ready,
			wantRows: []health.Status{health.Applied, health.Applied},
			wantSum:  statusSummary{Total: 2, Ready: 2, NotReady: 0},
		},
		{
			name:     "one not-ready Deployment",
			live:     []*unstructured.Unstructured{notReadyDeploy, makeResource("ConfigMap")},
			wantAgg:  health.NotReady,
			wantRows: []health.Status{health.NotReady, health.Applied},
			wantSum:  statusSummary{Total: 2, Ready: 1, NotReady: 1},
		},
		{
			name:     "one missing entry",
			live:     []*unstructured.Unstructured{makeResource("ConfigMap")},
			missing:  []MissingResource{{Kind: "Service", Namespace: "default", Name: "web"}},
			wantAgg:  health.NotReady,
			wantRows: []health.Status{health.Applied, health.Missing},
			wantSum:  statusSummary{Total: 2, Ready: 1, NotReady: 1},
		},
		{
			name:       "one unreadable entry",
			live:       []*unstructured.Unstructured{makeResource("ConfigMap")},
			unreadable: []UnreadableResource{{Kind: "Secret", Namespace: "default", Name: "creds", Err: forbidden}},
			wantAgg:    health.NotReady,
			wantRows:   []health.Status{health.Applied, health.Unknown},
			wantSum:    statusSummary{Total: 2, Ready: 1, NotReady: 1},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			res, err := GetInstanceStatus(context.Background(), nil, StatusOptions{
				InstanceName:        "my-app",
				InventoryLive:       tc.live,
				MissingResources:    tc.missing,
				UnreadableResources: tc.unreadable,
			})
			require.NoError(t, err)
			rows := make([]health.Status, len(res.Resources))
			for i, r := range res.Resources {
				rows[i] = r.Status
			}
			assert.Equal(t, tc.wantRows, rows)
			assert.Equal(t, tc.wantAgg, res.AggregateStatus)
			assert.Equal(t, tc.wantSum, res.Summary)
		})
	}
}

func TestFormatStatus_JSON(t *testing.T) {
	result := &StatusResult{
		InstanceName:    "my-app",
		Owner:           "cli",
		Namespace:       "default",
		AggregateStatus: health.Ready,
		Summary:         statusSummary{Total: 1, Ready: 1},
		Resources: []resourceHealth{
			{Kind: "Deployment", Name: "web", Namespace: "default", Status: health.Ready, Age: "5m"},
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
		AggregateStatus: health.Ready,
		Summary:         statusSummary{Total: 1, Ready: 1},
		Resources: []resourceHealth{
			{Kind: "Deployment", Name: "web", Namespace: "default", Status: health.Ready, Age: "5m"},
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
		AggregateStatus: health.Ready,
		Summary:         statusSummary{Total: 2, Ready: 2},
		Resources: []resourceHealth{
			{Kind: "Deployment", Name: "web", Namespace: "production", Component: "server", Status: health.Ready, Age: "5m"},
			{Kind: "ConfigMap", Name: "config", Namespace: "production", Component: "", Status: health.Ready, Age: "5m"},
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
		AggregateStatus: health.NotReady,
		Summary:         statusSummary{Total: 1, Ready: 0, NotReady: 1},
		Resources: []resourceHealth{
			{
				Kind: "Deployment", Name: "web", Namespace: "production", Component: "server",
				Status: health.NotReady, Age: "5m",
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
		AggregateStatus: health.NotReady,
		Summary:         statusSummary{Total: 1, Ready: 0, NotReady: 1},
		Resources: []resourceHealth{
			{
				Kind: "Deployment", Name: "web", Namespace: "production", Component: "server",
				Status: health.NotReady, Age: "5m",
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
		AggregateStatus: health.NotReady,
		Summary:         statusSummary{Total: 3, Ready: 2, NotReady: 1},
		Resources: []resourceHealth{
			{Kind: "Deployment", Name: "web", Namespace: "production", Status: health.NotReady, Age: "5m"},
			{Kind: "Service", Name: "svc", Namespace: "production", Status: health.Ready, Age: "5m"},
			{Kind: "ConfigMap", Name: "cfg", Namespace: "production", Status: health.Ready, Age: "5m"},
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
		AggregateStatus: health.Ready,
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
		AggregateStatus: health.Ready,
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
		InventoryLive:       []*unstructured.Unstructured{makeResource("ConfigMap")},
		UnreadableResources: unreadable,
	})
	require.NoError(t, err)
	require.Len(t, res.Resources, 2)
	row := res.Resources[1]
	assert.Equal(t, "Secret", row.Kind)
	assert.Equal(t, "creds", row.Name)
	assert.Equal(t, "default", row.Namespace)
	assert.Equal(t, health.Unknown, row.Status)
	assert.Equal(t, "<unknown>", row.Age)
	assert.Equal(t, health.NotReady, res.AggregateStatus)
	assert.Equal(t, statusSummary{Total: 2, Ready: 1, NotReady: 1}, res.Summary)

	// Only unreadable entries: still a status, not "no resources found".
	res, err = GetInstanceStatus(context.Background(), nil, StatusOptions{InstanceName: "my-app", UnreadableResources: unreadable})
	require.NoError(t, err)
	assert.Equal(t, health.NotReady, res.AggregateStatus)
	assert.Equal(t, statusSummary{Total: 1, NotReady: 1}, res.Summary)
}
