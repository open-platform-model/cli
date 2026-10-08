package kubernetes

import "github.com/open-platform-model/library/opm/k8s/ownership"

// ProtectedKindReason is the reason printed beside every CRD or Namespace
// that prune or delete leaves behind.
const ProtectedKindReason = "CRDs and Namespaces are never deleted"

// The API group and kind of a CustomResourceDefinition.
const (
	groupAPIExtensions           = "apiextensions.k8s.io"
	kindCustomResourceDefinition = "CustomResourceDefinition"
)

// IsProtectedKind reports whether the CLI never deletes objects of this kind:
// a Namespace takes everything inside it, and a CustomResourceDefinition takes
// every custom resource of its kind cluster-wide. Prune, the prune preview and
// instance delete all leave such objects behind; there is no override.
//
// The match is on group and kind, so a kind of the same name in another API
// group is not protected by accident. The rule is the library's
// ownership.SafetyExcluded, shared with the operator.
func IsProtectedKind(group, kind string) bool {
	return ownership.SafetyExcluded(group, kind)
}

// KindPersistentVolumeClaim is the kind of the claims delete and prune keep.
const KindPersistentVolumeClaim = "PersistentVolumeClaim"

// IsDataClaim reports whether the object is a PersistentVolumeClaim of the
// core API group. Deleting one deletes the data on its volume under the usual
// reclaim policy, so instance delete and prune keep it unless the user passes
// --delete-data. Unlike a protected kind (IsProtectedKind) it has that
// override, so the two tests stay apart.
//
// The match is on group and kind, as in IsProtectedKind.
func IsDataClaim(group, kind string) bool {
	return group == "" && kind == KindPersistentVolumeClaim
}
