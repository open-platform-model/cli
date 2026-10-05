package inventory

import (
	"testing"

	"github.com/stretchr/testify/assert"

	k8sinventory "github.com/open-platform-model/library/opm/k8s/inventory"
)

// IdentityOf keys exactly the relation the library's SameObject decides.
func TestIdentityOf_AgreesWithSameObject(t *testing.T) {
	base := k8sinventory.Entry{Group: "apps", Kind: "Deployment", Namespace: "ns", Name: "app", Version: "v1", Component: "web"}
	cases := map[string]func(e *k8sinventory.Entry){
		"identical": func(*k8sinventory.Entry) {},
		"group":     func(e *k8sinventory.Entry) { e.Group = "batch" },
		"kind":      func(e *k8sinventory.Entry) { e.Kind = "StatefulSet" },
		"namespace": func(e *k8sinventory.Entry) { e.Namespace = "other" },
		"name":      func(e *k8sinventory.Entry) { e.Name = "other" },
		"version":   func(e *k8sinventory.Entry) { e.Version = "v2" },
		"component": func(e *k8sinventory.Entry) { e.Component = "frontend" },
	}
	for name, change := range cases {
		t.Run(name, func(t *testing.T) {
			other := base
			change(&other)
			assert.Equal(t, k8sinventory.SameObject(base, other), IdentityOf(base) == IdentityOf(other))
		})
	}
}

func TestAdmitSet_Has(t *testing.T) {
	e := k8sinventory.Entry{Kind: "ConfigMap", Namespace: "ns", Name: "cm", Version: "v1"}
	var none AdmitSet
	assert.False(t, none.Has(e), "a nil set admits nothing")

	set := AdmitSet{IdentityOf(e): {}}
	moved := e
	moved.Component = "web"
	moved.Version = "v2"
	assert.True(t, set.Has(moved), "the component and version do not count")
	moved.Name = "other"
	assert.False(t, set.Has(moved))
}
