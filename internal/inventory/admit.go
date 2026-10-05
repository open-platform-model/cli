package inventory

import (
	k8sinventory "github.com/open-platform-model/library/opm/k8s/inventory"
)

// K8sIdentity is an object's group, kind, namespace and name: the fields
// k8sinventory.SameObject compares, as a comparable key. It restates the
// library's identity so the apply guard and the operator install plan can key
// maps by it (the library's own key is unexported). The ownership adoption,
// which moves the apply guard to opm/k8s/ownership, deletes it.
type K8sIdentity struct{ Group, Kind, Namespace, Name string }

// IdentityOf is an entry's K8sIdentity.
func IdentityOf(e k8sinventory.Entry) K8sIdentity {
	return K8sIdentity{Group: e.Group, Kind: e.Kind, Namespace: e.Namespace, Name: e.Name}
}

// AdmitSet is an explicit set of objects the first-install existence check
// lets pass its untracked-object test.
type AdmitSet map[K8sIdentity]struct{}

// Has reports whether the set admits the entry; a nil set admits nothing.
func (s AdmitSet) Has(e k8sinventory.Entry) bool {
	_, ok := s[IdentityOf(e)]
	return ok
}
