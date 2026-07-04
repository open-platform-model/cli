package operator

import (
	"fmt"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// rbacClusterRoleName is the ClusterRole (and, when a subject is given, the
// ClusterRoleBinding) --rbac emits.
const rbacClusterRoleName = "opm-cli-user"

const (
	rbacAPIVersion    = "rbac.authorization.k8s.io/v1"
	rbacAPIGroupField = "rbac.authorization.k8s.io"
)

// RBACOptions configures the optional opm-cli-user RBAC objects --rbac emits.
type RBACOptions struct {
	// Enabled turns on the opm-cli-user ClusterRole (and, when a subject is
	// given, its ClusterRoleBinding).
	Enabled bool

	// User is the username to bind. Mutually exclusive with Group.
	User string

	// Group is the group name to bind. Mutually exclusive with User.
	Group string
}

// Validate enforces that --user/--group are only used alongside --rbac,
// and that at most one of them is set. Call before any cluster interaction.
func (o RBACOptions) Validate() error {
	if !o.Enabled && (o.User != "" || o.Group != "") {
		return fmt.Errorf("--user/--group require --rbac")
	}
	if o.User != "" && o.Group != "" {
		return fmt.Errorf("--user and --group are mutually exclusive")
	}
	return nil
}

// Objects returns the opm-cli-user ClusterRole and, if a subject (--user or
// --group) is set, its ClusterRoleBinding. Returns nil when RBAC emission is
// disabled.
func (o RBACOptions) Objects() []*unstructured.Unstructured {
	if !o.Enabled {
		return nil
	}

	objs := []*unstructured.Unstructured{clusterRole()}

	switch {
	case o.User != "":
		objs = append(objs, clusterRoleBinding("User", o.User))
	case o.Group != "":
		objs = append(objs, clusterRoleBinding("Group", o.Group))
	}

	return objs
}

// clusterRole builds the opm-cli-user ClusterRole: full verbs on
// moduleinstances, get/patch/update on moduleinstances/status, get/list on
// platforms.
func clusterRole() *unstructured.Unstructured {
	return &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": rbacAPIVersion,
		"kind":       "ClusterRole",
		"metadata": map[string]any{
			"name": rbacClusterRoleName,
		},
		"rules": []any{
			map[string]any{
				"apiGroups": []any{opmodelAPIGroup},
				"resources": []any{moduleInstancesResource},
				"verbs":     []any{"*"},
			},
			map[string]any{
				"apiGroups": []any{opmodelAPIGroup},
				"resources": []any{moduleInstancesResource + "/status"},
				"verbs":     []any{"get", "patch", "update"},
			},
			map[string]any{
				"apiGroups": []any{opmodelAPIGroup},
				"resources": []any{"platforms"},
				"verbs":     []any{"get", "list"},
			},
		},
	}}
}

// clusterRoleBinding builds the ClusterRoleBinding for a single subject
// ("User" or "Group") bound to the opm-cli-user ClusterRole.
func clusterRoleBinding(subjectKind, subjectName string) *unstructured.Unstructured {
	return &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": rbacAPIVersion,
		"kind":       "ClusterRoleBinding",
		"metadata": map[string]any{
			"name": rbacClusterRoleName,
		},
		"subjects": []any{
			map[string]any{
				"kind":     subjectKind,
				"name":     subjectName,
				"apiGroup": rbacAPIGroupField,
			},
		},
		"roleRef": map[string]any{
			"apiGroup": rbacAPIGroupField,
			"kind":     "ClusterRole",
			"name":     rbacClusterRoleName,
		},
	}}
}
