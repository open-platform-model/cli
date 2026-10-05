package query

import (
	"context"
	"io"
	"os"
	"regexp"
	"testing"

	k8sinventory "github.com/open-platform-model/library/opm/k8s/inventory"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/open-platform-model/cli/internal/inventory"
	"github.com/open-platform-model/cli/internal/kubernetes"
	"github.com/open-platform-model/cli/internal/output"
)

// captureStdout returns what fn writes to os.Stdout.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	require.NoError(t, err)
	orig := os.Stdout
	os.Stdout = w
	t.Cleanup(func() { os.Stdout = orig })

	done := make(chan []byte)
	go func() {
		data, _ := io.ReadAll(r)
		done <- data
	}()
	fn()
	require.NoError(t, w.Close())
	os.Stdout = orig
	return string(<-done)
}

func goldenSummaries() []InstanceSummary {
	return []InstanceSummary{
		{Name: "web", Module: "example.com/web@v1", Namespace: "prod", Version: "1.0.0", Status: string(kubernetes.HealthReady),
			ReadyCount: 3, TotalCount: 3, InstanceID: "11111111-1111-1111-1111-111111111111", LastApplied: "2026-10-01T10:00:00Z", Age: "4d", Owner: "cli"},
		{Name: "db", Module: "example.com/db@v1", Namespace: "prod", Version: "2.1.0", Status: string(kubernetes.HealthNotReady),
			ReadyCount: 3, TotalCount: 5, InstanceID: "22222222-2222-2222-2222-222222222222", LastApplied: "2026-10-02T10:00:00Z", Age: "3d", Owner: "operator"},
		{Name: "jobs", Module: "-", Namespace: "prod", Version: "-", Status: string(kubernetes.HealthUnknown),
			ReadyCount: 0, TotalCount: 0, InstanceID: "33333333-3333-3333-3333-333333333333", LastApplied: "", Age: "<unknown>", Owner: "cli"},
	}
}

const goldenListJSON = `[
  {
    "name": "web",
    "module": "example.com/web@v1",
    "namespace": "prod",
    "version": "1.0.0",
    "status": "Ready",
    "readyCount": 3,
    "totalCount": 3,
    "instanceID": "11111111-1111-1111-1111-111111111111",
    "lastApplied": "2026-10-01T10:00:00Z",
    "age": "4d",
    "owner": "cli"
  },
  {
    "name": "db",
    "module": "example.com/db@v1",
    "namespace": "prod",
    "version": "2.1.0",
    "status": "NotReady",
    "readyCount": 3,
    "totalCount": 5,
    "instanceID": "22222222-2222-2222-2222-222222222222",
    "lastApplied": "2026-10-02T10:00:00Z",
    "age": "3d",
    "owner": "operator"
  },
  {
    "name": "jobs",
    "module": "-",
    "namespace": "prod",
    "version": "-",
    "status": "Unknown",
    "readyCount": 0,
    "totalCount": 0,
    "instanceID": "33333333-3333-3333-3333-333333333333",
    "lastApplied": "",
    "age": "\u003cunknown\u003e",
    "owner": "cli"
  }
]
`

const goldenListYAML = `- name: web
  module: example.com/web@v1
  namespace: prod
  version: 1.0.0
  status: Ready
  readyCount: 3
  totalCount: 3
  instanceID: 11111111-1111-1111-1111-111111111111
  lastApplied: "2026-10-01T10:00:00Z"
  age: 4d
  owner: cli
- name: db
  module: example.com/db@v1
  namespace: prod
  version: 2.1.0
  status: NotReady
  readyCount: 3
  totalCount: 5
  instanceID: 22222222-2222-2222-2222-222222222222
  lastApplied: "2026-10-02T10:00:00Z"
  age: 3d
  owner: operator
- name: jobs
  module: '-'
  namespace: prod
  version: '-'
  status: Unknown
  readyCount: 0
  totalCount: 0
  instanceID: 33333333-3333-3333-3333-333333333333
  lastApplied: ""
  age: <unknown>
  owner: cli
`

// The status strings are `opm instance list -o json|yaml` output. These
// literals were generated once from the cli's own evaluator before it moved
// to the library's opm/k8s/health; they must never be regenerated to make a
// test pass.
func TestRenderInstanceListOutput_GoldenJSON(t *testing.T) {
	got := captureStdout(t, func() {
		require.NoError(t, RenderInstanceListOutput(goldenSummaries(), output.FormatJSON, false))
	})
	assert.Equal(t, goldenListJSON, got)
}

func TestRenderInstanceListOutput_GoldenYAML(t *testing.T) {
	got := captureStdout(t, func() {
		require.NoError(t, RenderInstanceListOutput(goldenSummaries(), output.FormatYAML, false))
	})
	assert.Equal(t, goldenListYAML, got)
}

var ansiEscape = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func namedObject(apiVersion, kind, name string) *unstructured.Unstructured {
	return &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": apiVersion, "kind": kind,
		"metadata": map[string]any{"name": name, "namespace": "apps"},
	}}
}

// rolledOutDeployment is a Deployment whose rollout has finished when
// rolledOut, and one still behind (Available, but updatedReplicas below
// spec.replicas) when not.
func rolledOutDeployment(name string, rolledOut bool) *unstructured.Unstructured {
	updated := int64(2)
	if !rolledOut {
		updated = 1
	}
	d := namedObject("apps/v1", "Deployment", name)
	d.Object["metadata"].(map[string]any)["generation"] = int64(1)
	d.Object["spec"] = map[string]any{"replicas": int64(2)}
	d.Object["status"] = map[string]any{
		"observedGeneration": int64(1), "replicas": int64(2),
		"updatedReplicas": updated, "availableReplicas": int64(2),
		"conditions": []any{map[string]any{"type": "Available", "status": "True"}},
	}
	return d
}

func entryOf(group, kind, name string) k8sinventory.Entry {
	return k8sinventory.Entry{Group: group, Kind: kind, Namespace: "apps", Name: name, Version: "v1"}
}

// opm instance list folds each instance's statuses into one verdict with a
// ready count and a total. Missing and unreadable entries count toward the
// total and never toward ready.
func TestEvaluateInstanceHealth_Aggregate(t *testing.T) {
	tests := []struct {
		name       string
		objs       []*unstructured.Unstructured
		entries    []k8sinventory.Entry
		forbidCMs  bool
		wantStatus string
		wantReady  int
		wantTotal  int
		wantColumn string
	}{
		{
			name:       "all healthy",
			objs:       []*unstructured.Unstructured{rolledOutDeployment("web", true), namedObject("v1", "ServiceAccount", "runner")},
			entries:    []k8sinventory.Entry{entryOf("apps", "Deployment", "web"), entryOf("", "ServiceAccount", "runner")},
			wantStatus: "Ready", wantReady: 2, wantTotal: 2,
			wantColumn: "Ready (2/2)",
		},
		{
			name:       "one not-ready Deployment",
			objs:       []*unstructured.Unstructured{rolledOutDeployment("web", false), namedObject("v1", "ServiceAccount", "runner")},
			entries:    []k8sinventory.Entry{entryOf("apps", "Deployment", "web"), entryOf("", "ServiceAccount", "runner")},
			wantStatus: "NotReady", wantReady: 1, wantTotal: 2,
			wantColumn: "NotReady (1/2)",
		},
		{
			name: "missing and unreadable count against the aggregate",
			objs: []*unstructured.Unstructured{
				namedObject("v1", "ServiceAccount", "runner"),
				namedObject("v1", "Secret", "creds"),
				namedObject("v1", "Service", "web"),
				namedObject("v1", "ConfigMap", "settings"),
			},
			entries: []k8sinventory.Entry{
				entryOf("", "ServiceAccount", "runner"),
				entryOf("", "Secret", "creds"),
				entryOf("", "Service", "web"),
				entryOf("rbac.authorization.k8s.io", "Role", "gone"),
				entryOf("", "ConfigMap", "settings"),
			},
			forbidCMs:  true,
			wantStatus: "NotReady", wantReady: 3, wantTotal: 5,
			wantColumn: "NotReady (3/5)",
		},
		{
			name:       "empty inventory",
			wantStatus: "Unknown", wantReady: 0, wantTotal: 0,
			wantColumn: "Unknown (0/0)",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			client := makeCRClient(tc.objs...)
			if tc.forbidCMs {
				forbidConfigMapGets(t, client)
			}
			captureLog(t)
			inv := &inventory.Record{Name: "demo", Namespace: "apps", Inventory: inventory.Inventory{Entries: tc.entries}}

			summaries := EvaluateInstanceHealth(context.Background(), client, []*inventory.Record{inv}, 1, false)
			require.Len(t, summaries, 1)
			s := summaries[0]
			assert.Equal(t, tc.wantStatus, s.Status)
			assert.Equal(t, tc.wantReady, s.ReadyCount)
			assert.Equal(t, tc.wantTotal, s.TotalCount)
			assert.Equal(t, tc.wantColumn, ansiEscape.ReplaceAllString(formatStatusColumn(s.Status, s.ReadyCount, s.TotalCount), ""))
		})
	}
}
