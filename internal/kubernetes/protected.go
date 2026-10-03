package kubernetes

// ProtectedKindReason is the reason printed beside every CRD or Namespace
// that prune or delete leaves behind.
const ProtectedKindReason = "CRDs and Namespaces are never deleted"

// IsProtectedKind reports whether the CLI never deletes objects of this kind:
// a Namespace takes everything inside it, and a CustomResourceDefinition takes
// every custom resource of its kind cluster-wide. Prune, the prune preview and
// instance delete all leave such objects behind; there is no override.
//
// The match is on group and kind, so a kind of the same name in another API
// group is not protected by accident.
func IsProtectedKind(group, kind string) bool {
	switch {
	case group == "" && kind == "Namespace":
		return true
	case group == "apiextensions.k8s.io" && kind == "CustomResourceDefinition":
		return true
	}
	return false
}
