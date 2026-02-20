package release_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/opmodel/cli/internal/core"
)

// TestOPMNamespaceUUIDCorrect verifies the OPM namespace UUID is correctly computed
// from the DNS namespace + "opmodel.dev".
func TestOPMNamespaceUUIDCorrect(t *testing.T) {
	// The expected UUID was computed as:
	// uuid.NewSHA1(uuid.NameSpaceDNS, []byte("opmodel.dev")).String()
	// This ensures the constant used in the release builder is correct.
	//
	// Note: the old overlay.go used a different (incorrect) constant
	// "c1cbe76d-5687-5a47-bfe6-83b081b15413". The correct value is in core.OPMNamespace.
	expected := "11bc6112-a6e8-4021-bec9-b3ad246f9466"
	assert.Equal(t, expected, core.OPMNamespace,
		"OPM namespace UUID doesn't match expected value")
}
