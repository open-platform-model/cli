package operator

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestRBACOptions_Validate(t *testing.T) {
	tests := []struct {
		name    string
		opts    RBACOptions
		wantErr string
	}{
		{"disabled with no subject is valid", RBACOptions{}, ""},
		{"enabled with no subject is valid", RBACOptions{Enabled: true}, ""},
		{"enabled with user is valid", RBACOptions{Enabled: true, User: "alice"}, ""},
		{"enabled with group is valid", RBACOptions{Enabled: true, Group: "platform-team"}, ""},
		{"user without rbac is rejected", RBACOptions{User: "alice"}, "--user/--group require --rbac"},
		{"group without rbac is rejected", RBACOptions{Group: "platform-team"}, "--user/--group require --rbac"},
		{"user and group together is rejected", RBACOptions{Enabled: true, User: "alice", Group: "platform-team"}, "mutually exclusive"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.opts.Validate()
			if tt.wantErr == "" {
				assert.NoError(t, err)
				return
			}
			require.Error(t, err)
			assert.ErrorContains(t, err, tt.wantErr)
		})
	}
}

func TestRBACOptions_Objects_DisabledReturnsNone(t *testing.T) {
	assert.Empty(t, RBACOptions{}.Objects())
}

func TestRBACOptions_Objects_EnabledWithoutSubjectIsRoleOnly(t *testing.T) {
	objs := RBACOptions{Enabled: true}.Objects()
	require.Len(t, objs, 1)
	assert.Equal(t, "ClusterRole", objs[0].GetKind())
	assert.Equal(t, "opm-cli-user", objs[0].GetName())
}

func TestRBACOptions_Objects_RoleShape(t *testing.T) {
	objs := RBACOptions{Enabled: true}.Objects()
	require.Len(t, objs, 1)
	role := objs[0]

	rules, found, err := unstructured.NestedSlice(role.Object, "rules")
	require.NoError(t, err)
	require.True(t, found)
	require.Len(t, rules, 3)

	rule := rules[0].(map[string]any)
	assert.Equal(t, []any{"opmodel.dev"}, rule["apiGroups"])
	assert.Equal(t, []any{"moduleinstances"}, rule["resources"])
	assert.Equal(t, []any{"*"}, rule["verbs"])

	statusRule := rules[1].(map[string]any)
	assert.Equal(t, []any{"moduleinstances/status"}, statusRule["resources"])
	assert.Equal(t, []any{"get", "patch", "update"}, statusRule["verbs"])

	platformsRule := rules[2].(map[string]any)
	assert.Equal(t, []any{"platforms"}, platformsRule["resources"])
	assert.Equal(t, []any{"get", "list"}, platformsRule["verbs"])
}

func TestRBACOptions_Objects_UserSubjectAddsBinding(t *testing.T) {
	objs := RBACOptions{Enabled: true, User: "alice"}.Objects()
	require.Len(t, objs, 2)
	assert.Equal(t, "ClusterRole", objs[0].GetKind())

	binding := objs[1]
	assert.Equal(t, "ClusterRoleBinding", binding.GetKind())
	assert.Equal(t, "opm-cli-user", binding.GetName())

	subjects, found, err := unstructured.NestedSlice(binding.Object, "subjects")
	require.NoError(t, err)
	require.True(t, found)
	require.Len(t, subjects, 1)
	subject := subjects[0].(map[string]any)
	assert.Equal(t, "User", subject["kind"])
	assert.Equal(t, "alice", subject["name"])

	roleRef, found, err := unstructured.NestedMap(binding.Object, "roleRef")
	require.NoError(t, err)
	require.True(t, found)
	assert.Equal(t, "ClusterRole", roleRef["kind"])
	assert.Equal(t, "opm-cli-user", roleRef["name"])
}

func TestRBACOptions_Objects_GroupSubjectAddsBinding(t *testing.T) {
	objs := RBACOptions{Enabled: true, Group: "platform-team"}.Objects()
	require.Len(t, objs, 2)

	binding := objs[1]
	subjects, _, err := unstructured.NestedSlice(binding.Object, "subjects")
	require.NoError(t, err)
	subject := subjects[0].(map[string]any)
	assert.Equal(t, "Group", subject["kind"])
	assert.Equal(t, "platform-team", subject["name"])
}
