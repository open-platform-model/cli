package release

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestOPMNamespaceUUIDCorrect verifies the OPM namespace UUID is correctly computed
// from the DNS namespace + "opmodel.dev".
func TestOPMNamespaceUUIDCorrect(t *testing.T) {
	// The expected UUID was computed as:
	// uuid.NewSHA1(uuid.NameSpaceDNS, []byte("opmodel.dev")).String()
	// This ensures the constant used in the release builder is correct.
	expected := "c1cbe76d-5687-5a47-bfe6-83b081b15413"
	assert.Equal(t, expected, opmNamespaceUUID,
		"OPM namespace UUID doesn't match expected value")
}
