package operator

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func deploymentWithImage(image string) *unstructured.Unstructured {
	objs := moduleObjects(renderOpts{})
	dep := objs[len(objs)-1]
	_ = unstructured.SetNestedSlice(dep.Object, []any{map[string]any{"name": "manager", "image": image}},
		"spec", "template", "spec", "containers")
	return dep
}

func TestRenderedOperatorImage(t *testing.T) {
	cases := []struct {
		image, tag, err string
	}{
		{"ghcr.io/open-platform-model/opm-operator:v1.0.0-beta.7@sha256:7871a5dd", "v1.0.0-beta.7", ""},
		{"ghcr.io/open-platform-model/opm-operator:v1.0.0", "v1.0.0", ""},
		{"localhost:5000/opm-operator:v1.2.3", "v1.2.3", ""},
		{"localhost:5000/opm-operator@sha256:abcd", "", "carries no tag"},
		{"opm-operator", "", "carries no tag"},
	}
	for _, c := range cases {
		tag, err := RenderedOperatorImage([]*unstructured.Unstructured{deploymentWithImage(c.image)})
		if c.err != "" {
			require.Error(t, err, c.image)
			assert.Contains(t, err.Error(), c.err, c.image)
			continue
		}
		require.NoError(t, err, c.image)
		assert.Equal(t, c.tag, tag, c.image)
	}

	_, err := RenderedOperatorImage(moduleObjects(renderOpts{noDeployment: true}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no Deployment opm-operator-system/opm-operator-controller-manager")

	other := deploymentWithImage("x:v1")
	_ = unstructured.SetNestedSlice(other.Object, []any{map[string]any{"name": "sidecar", "image": "x:v1"}},
		"spec", "template", "spec", "containers")
	_, err = RenderedOperatorImage([]*unstructured.Unstructured{other})
	require.Error(t, err)
	assert.Contains(t, err.Error(), `no container "manager"`)
}

// "Install refuses a target version the CLI cannot drive or that disagrees
// with itself", each scenario.
func TestCheckTarget(t *testing.T) {
	target := Target{ModuleVersion: "v0.1.0", OperatorVersion: "v1.0.0-beta.7"}

	require.NoError(t, CheckTarget(target, moduleObjects(renderOpts{}), "v1.0.0-beta.9"))
	require.NoError(t, CheckTarget(target, moduleObjects(renderOpts{}), "1.0.4"), "patch and prerelease never refuse")

	refusals := []struct {
		name   string
		target Target
		objs   []*unstructured.Unstructured
		cli    string
		want   []string
	}{
		{
			"operator newer than the CLI",
			Target{ModuleVersion: "v0.4.0", OperatorVersion: "v1.1.0"},
			moduleObjects(renderOpts{imageTag: "v1.1.0"}), "v1.0.0-beta.9",
			[]string{"opm_operator 0.4.0", "v1.1.0", "upgrade the CLI"},
		},
		{
			"rendered image disagrees with the stated operator version",
			Target{ModuleVersion: "v0.2.0", OperatorVersion: "v1.0.0-beta.6"},
			moduleObjects(renderOpts{imageTag: "v1.0.0-beta.5"}), "v1.0.0-beta.9",
			[]string{"opm_operator 0.2.0", "v1.0.0-beta.6", "v1.0.0-beta.5"},
		},
		{
			"no controller Deployment",
			target, moduleObjects(renderOpts{noDeployment: true}), "v1.0.0-beta.9",
			[]string{"opm_operator 0.1.0", "no Deployment"},
		},
		{
			"rendered CRD below the floor",
			target, moduleObjects(renderOpts{noCRDFloor: true}), "v1.0.0-beta.9",
			[]string{"opm_operator 0.1.0", "spec.owner and status.inventory"},
		},
	}
	for _, r := range refusals {
		err := CheckTarget(r.target, r.objs, r.cli)
		var te *TargetError
		require.True(t, errors.As(err, &te), "%s: got %v", r.name, err)
		assert.True(t, IsRefusal(err), r.name)
		for _, w := range r.want {
			assert.Contains(t, err.Error(), w, r.name)
		}
	}

	// A dev CLI skips V1 with a warning, but the image and floor rules hold.
	require.NoError(t, CheckTarget(Target{ModuleVersion: "v0.4.0", OperatorVersion: "v1.1.0"},
		moduleObjects(renderOpts{imageTag: "v1.1.0"}), "dev"))
	require.Error(t, CheckTarget(target, moduleObjects(renderOpts{noCRDFloor: true}), "dev"))
}
