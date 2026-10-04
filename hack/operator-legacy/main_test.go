package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Only operator releases are a source of the list; an operator module
// release and a malformed tag are skipped.
func TestIsOperatorRelease(t *testing.T) {
	for _, tag := range []string{"v1.0.0-beta.5", "v0.5.0", "v1.0.0-alpha", "v1.0.0"} {
		assert.True(t, IsOperatorRelease(tag), tag)
	}
	for _, tag := range []string{"opm_operator-v0.1.0", "opm_operator-v1.0.0", "1.0.0-beta.5", "v1.0", "v1.0.0-", "release-v1.0.0", ""} {
		assert.False(t, IsOperatorRelease(tag), tag)
	}
}

func TestParseManifest(t *testing.T) {
	objs, err := ParseManifest([]byte(`apiVersion: v1
kind: Namespace
metadata:
  name: opm-operator-system
  labels:
    control-plane: controller-manager
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: opm-operator-controller-manager
  namespace: opm-operator-system
spec:
  selector:
    matchLabels:
      control-plane: controller-manager
  template:
    spec:
      containers: [{name: manager, image: x}]
---
`))
	require.NoError(t, err)
	assert.Equal(t, []Object{
		{Kind: "Namespace", Name: "opm-operator-system", Labels: map[string]string{"control-plane": "controller-manager"}},
		{Group: "apps", Kind: "Deployment", Namespace: "opm-operator-system", Name: "opm-operator-controller-manager",
			Selector: map[string]string{"control-plane": "controller-manager"}},
	}, objs)
}
