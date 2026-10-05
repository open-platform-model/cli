package inventory

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/open-platform-model/cli/internal/kubernetes"
)

// --- kindToResource ---

func TestKindToResource(t *testing.T) {
	cases := []struct {
		kind     string
		resource string
	}{
		{"Deployment", "deployments"},
		{"Service", "services"},
		{"ConfigMap", "configmaps"},
		{"Endpoints", "endpoints"},
		{"ClusterRole", "clusterroles"},
	}
	for _, tc := range cases {
		t.Run(tc.kind, func(t *testing.T) {
			assert.Equal(t, tc.resource, kubernetes.KindToResource(tc.kind))
		})
	}
}
