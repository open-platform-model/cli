package build

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/opmodel/cli/internal/identity"
)

// TestOPMNamespaceUUIDCorrect verifies the OPM namespace UUID is correctly computed
// from the DNS namespace + "opmodel.dev".
func TestOPMNamespaceUUIDCorrect(t *testing.T) {
	// The expected UUID was computed as:
	// uuid.NewSHA1(uuid.NameSpaceDNS, []byte("opmodel.dev")).String()
	// This ensures the constant in the identity package is correct.
	expected := "c1cbe76d-5687-5a47-bfe6-83b081b15413"
	assert.Equal(t, expected, identity.OPMNamespaceUUID,
		"OPM namespace UUID doesn't match expected value")
}
