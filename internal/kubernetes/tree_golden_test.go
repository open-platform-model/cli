package kubernetes

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// goldenTreeResult carries one component per aggregate verdict, a workload
// chain down to a pod with its raw phase, and a bound PVC.
func goldenTreeResult() *TreeResult {
	return &TreeResult{
		Instance: InstanceInfo{Name: "my-app", Namespace: "prod", Module: "example.com/app@v1", Version: "1.2.3"},
		Components: []Component{
			{
				Name:          "server",
				ResourceCount: 1,
				Status:        HealthReady,
				Resources: []ResourceNode{
					{
						Kind: "Deployment", Name: "web", Namespace: "prod", Status: HealthReady, Replicas: "1/1",
						Children: []ResourceNode{
							{
								Kind: "ReplicaSet", Name: "web-abc", Namespace: "prod", Status: HealthReady, Replicas: "1 pods",
								Children: []ResourceNode{
									{Kind: "Pod", Name: "web-abc-xyz", Namespace: "prod", Status: HealthStatus("Running"), Ready: true},
								},
							},
						},
					},
				},
			},
			{
				Name:          "database",
				ResourceCount: 2,
				Status:        HealthNotReady,
				Resources: []ResourceNode{
					{Kind: "StatefulSet", Name: "db", Namespace: "prod", Status: HealthNotReady, Replicas: "0/1"},
					{Kind: "PersistentVolumeClaim", Name: "data", Namespace: "prod", Status: HealthBound, Replicas: "10Gi"},
				},
			},
			{
				Name:          "jobs",
				ResourceCount: 1,
				Status:        HealthUnknown,
			},
		},
	}
}

const goldenTreeJSON = `{
  "instance": {
    "name": "my-app",
    "namespace": "prod",
    "module": "example.com/app@v1",
    "version": "1.2.3"
  },
  "components": [
    {
      "name": "server",
      "resourceCount": 1,
      "status": "Ready",
      "resources": [
        {
          "kind": "Deployment",
          "name": "web",
          "namespace": "prod",
          "status": "Ready",
          "replicas": "1/1",
          "children": [
            {
              "kind": "ReplicaSet",
              "name": "web-abc",
              "namespace": "prod",
              "status": "Ready",
              "replicas": "1 pods",
              "children": [
                {
                  "kind": "Pod",
                  "name": "web-abc-xyz",
                  "namespace": "prod",
                  "status": "Running",
                  "ready": true
                }
              ]
            }
          ]
        }
      ]
    },
    {
      "name": "database",
      "resourceCount": 2,
      "status": "NotReady",
      "resources": [
        {
          "kind": "StatefulSet",
          "name": "db",
          "namespace": "prod",
          "status": "NotReady",
          "replicas": "0/1"
        },
        {
          "kind": "PersistentVolumeClaim",
          "name": "data",
          "namespace": "prod",
          "status": "Bound",
          "replicas": "10Gi"
        }
      ]
    },
    {
      "name": "jobs",
      "resourceCount": 1,
      "status": "Unknown"
    }
  ]
}`

const goldenTreeYAML = `instance:
    name: my-app
    namespace: prod
    module: example.com/app@v1
    version: 1.2.3
components:
    - name: server
      resourceCount: 1
      status: Ready
      resources:
        - kind: Deployment
          name: web
          namespace: prod
          status: Ready
          replicas: 1/1
          children:
            - kind: ReplicaSet
              name: web-abc
              namespace: prod
              status: Ready
              replicas: 1 pods
              children:
                - kind: Pod
                  name: web-abc-xyz
                  namespace: prod
                  status: Running
                  ready: true
    - name: database
      resourceCount: 2
      status: NotReady
      resources:
        - kind: StatefulSet
          name: db
          namespace: prod
          status: NotReady
          replicas: 0/1
        - kind: PersistentVolumeClaim
          name: data
          namespace: prod
          status: Bound
          replicas: 10Gi
    - name: jobs
      resourceCount: 1
      status: Unknown
`

// The status strings are `opm instance tree -o json|yaml` output. These
// literals were generated once from the cli's own evaluator before it moved
// to the library's opm/k8s/health; they must never be regenerated to make a
// test pass.
func TestFormatTree_GoldenJSON(t *testing.T) {
	got, err := FormatTreeJSON(goldenTreeResult())
	require.NoError(t, err)
	assert.Equal(t, goldenTreeJSON, got)
}

func TestFormatTree_GoldenYAML(t *testing.T) {
	got, err := FormatTreeYAML(goldenTreeResult())
	require.NoError(t, err)
	assert.Equal(t, goldenTreeYAML, got)
}
