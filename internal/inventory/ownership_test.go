package inventory

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResolveOwnership(t *testing.T) {
	cases := []struct {
		name string
		rec  *Record
		want OwnershipMode
	}{
		{"nil record (no CR) is CLI-executor", nil, ModeCLIExecutor},
		{"owner cli is CLI-executor", &Record{Owner: OwnerCLI}, ModeCLIExecutor},
		{"owner operator is operator-owned", &Record{Owner: OwnerOperator}, ModeOperatorOwned},
		{"empty owner on existing CR is operator-owned", &Record{Owner: ""}, ModeOperatorOwned},
		{"unknown owner is operator-owned", &Record{Owner: "someone-else"}, ModeOperatorOwned},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, ResolveOwnership(tc.rec))
		})
	}
}

func TestDisplayOwner(t *testing.T) {
	assert.Equal(t, "cli", DisplayOwner("cli"))
	assert.Equal(t, "operator", DisplayOwner("operator"))
	assert.Equal(t, "operator", DisplayOwner(""))
}

func TestOperatorOwnedErrors(t *testing.T) {
	applyErr := OperatorOwnedApplyError("demo", "apps")
	assert.Contains(t, applyErr.Error(), "operator-managed")
	assert.Contains(t, applyErr.Error(), "demo")

	deleteErr := OperatorOwnedDeleteError("demo", "apps")
	assert.Contains(t, deleteErr.Error(), "operator-managed")
	assert.Contains(t, deleteErr.Error(), "kubectl delete moduleinstance demo")
}
