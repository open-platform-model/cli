package kubernetes

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsProtectedKind(t *testing.T) {
	tests := []struct {
		name  string
		group string
		kind  string
		want  bool
	}{
		{"core Namespace", "", "Namespace", true},
		{"apiextensions CRD", "apiextensions.k8s.io", "CustomResourceDefinition", true},
		{"Namespace kind in another group", "example.io", "Namespace", false},
		{"CRD kind in the core group", "", "CustomResourceDefinition", false},
		{"apps Deployment", "apps", "Deployment", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, IsProtectedKind(tc.group, tc.kind))
		})
	}
}
