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
		{"owner controller is controller-owned", &Record{Owner: OwnerController}, ModeControllerOwned},
		{"empty owner on existing CR is controller-owned", &Record{Owner: ""}, ModeControllerOwned},
		{"unknown owner is controller-owned", &Record{Owner: "someone-else"}, ModeControllerOwned},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, ResolveOwnership(tc.rec))
		})
	}
}

func TestDisplayOwner(t *testing.T) {
	assert.Equal(t, "cli", DisplayOwner("cli"))
	assert.Equal(t, "controller", DisplayOwner("controller"))
	assert.Equal(t, "controller", DisplayOwner(""))
}
