package kubernetes

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/cli/internal/output"
)

// goldenStatusResult carries every status string `opm instance status` can
// print: the seven constants and a PersistentVolumeClaim's raw phase.
func goldenStatusResult() *StatusResult {
	return &StatusResult{
		InstanceName:    "my-app",
		Version:         "1.2.3",
		Owner:           "cli",
		Namespace:       "prod",
		AggregateStatus: HealthNotReady,
		Summary:         statusSummary{Total: 8, Ready: 4, NotReady: 4},
		Resources: []resourceHealth{
			{Kind: "Deployment", Name: "web", Namespace: "prod", Component: "server", Status: HealthReady, Age: "5m"},
			{Kind: "StatefulSet", Name: "db", Namespace: "prod", Component: "database", Status: HealthNotReady, Age: "5m"},
			{Kind: "Job", Name: "migrate", Namespace: "prod", Component: "database", Status: HealthComplete, Age: "4m"},
			{Kind: "Secret", Name: "creds", Namespace: "prod", Component: "server", Status: HealthUnknown, Age: "<unknown>"},
			{Kind: "Service", Name: "web", Namespace: "prod", Component: "server", Status: HealthMissing, Age: "<unknown>"},
			{Kind: "ConfigMap", Name: "settings", Namespace: "prod", Component: "server", Status: HealthApplied, Age: "5m"},
			{Kind: "PersistentVolumeClaim", Name: "data", Namespace: "prod", Component: "database", Status: HealthBound, Age: "5m"},
			{Kind: "PersistentVolumeClaim", Name: "scratch", Namespace: "prod", Component: "database", Status: HealthStatus("Pending"), Age: "1m"},
		},
	}
}

const goldenStatusJSON = `{
  "instanceName": "my-app",
  "version": "1.2.3",
  "owner": "cli",
  "namespace": "prod",
  "resources": [
    {
      "kind": "Deployment",
      "name": "web",
      "namespace": "prod",
      "component": "server",
      "status": "Ready",
      "age": "5m"
    },
    {
      "kind": "StatefulSet",
      "name": "db",
      "namespace": "prod",
      "component": "database",
      "status": "NotReady",
      "age": "5m"
    },
    {
      "kind": "Job",
      "name": "migrate",
      "namespace": "prod",
      "component": "database",
      "status": "Complete",
      "age": "4m"
    },
    {
      "kind": "Secret",
      "name": "creds",
      "namespace": "prod",
      "component": "server",
      "status": "Unknown",
      "age": "\u003cunknown\u003e"
    },
    {
      "kind": "Service",
      "name": "web",
      "namespace": "prod",
      "component": "server",
      "status": "Missing",
      "age": "\u003cunknown\u003e"
    },
    {
      "kind": "ConfigMap",
      "name": "settings",
      "namespace": "prod",
      "component": "server",
      "status": "Applied",
      "age": "5m"
    },
    {
      "kind": "PersistentVolumeClaim",
      "name": "data",
      "namespace": "prod",
      "component": "database",
      "status": "Bound",
      "age": "5m"
    },
    {
      "kind": "PersistentVolumeClaim",
      "name": "scratch",
      "namespace": "prod",
      "component": "database",
      "status": "Pending",
      "age": "1m"
    }
  ],
  "aggregateStatus": "NotReady",
  "summary": {
    "total": 8,
    "ready": 4,
    "notReady": 4
  }
}`

const goldenStatusYAML = `instanceName: my-app
version: 1.2.3
owner: cli
namespace: prod
resources:
    - kind: Deployment
      name: web
      namespace: prod
      component: server
      status: Ready
      age: 5m
    - kind: StatefulSet
      name: db
      namespace: prod
      component: database
      status: NotReady
      age: 5m
    - kind: Job
      name: migrate
      namespace: prod
      component: database
      status: Complete
      age: 4m
    - kind: Secret
      name: creds
      namespace: prod
      component: server
      status: Unknown
      age: <unknown>
    - kind: Service
      name: web
      namespace: prod
      component: server
      status: Missing
      age: <unknown>
    - kind: ConfigMap
      name: settings
      namespace: prod
      component: server
      status: Applied
      age: 5m
    - kind: PersistentVolumeClaim
      name: data
      namespace: prod
      component: database
      status: Bound
      age: 5m
    - kind: PersistentVolumeClaim
      name: scratch
      namespace: prod
      component: database
      status: Pending
      age: 1m
aggregateStatus: NotReady
summary:
    total: 8
    ready: 4
    notReady: 4
`

// The status strings are `opm instance status -o json|yaml` output. These
// literals were generated once from the cli's own evaluator before it moved
// to the library's opm/k8s/health; they must never be regenerated to make a
// test pass.
func TestFormatStatus_GoldenJSON(t *testing.T) {
	got, err := FormatStatus(goldenStatusResult(), output.FormatJSON)
	require.NoError(t, err)
	assert.Equal(t, goldenStatusJSON, got)
}

func TestFormatStatus_GoldenYAML(t *testing.T) {
	got, err := FormatStatus(goldenStatusResult(), output.FormatYAML)
	require.NoError(t, err)
	assert.Equal(t, goldenStatusYAML, got)
}
