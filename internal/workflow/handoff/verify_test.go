package handoff

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	liberrors "github.com/open-platform-model/library/opm/errors"
)

func TestAcquireFailureError_IdentityMismatch(t *testing.T) {
	// The fetch succeeded but the artifact lies about its identity: the
	// message must name declared vs fetched, not claim the module is
	// unpublished.
	idErr := &liberrors.IdentityError{
		Artifact: "module",
		Field:    "version",
		Declared: "v0.1.2",
		Fetched:  "v0.1.4",
	}
	wrapped := fmt.Errorf("acquiring module: %w", idErr)

	err := acquireFailureError("opmodel.dev/modules/test/podinfo@v0", "v0.1.4", wrapped)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "identity mismatch")
	assert.Contains(t, err.Error(), `declares version "v0.1.2"`)
	assert.Contains(t, err.Error(), `fetched by "v0.1.4"`)
	assert.NotContains(t, err.Error(), "publish the module (")
	assert.ErrorAs(t, err, new(*liberrors.IdentityError))
}

func TestAcquireFailureError_NotFoundKeepsPublishHint(t *testing.T) {
	err := acquireFailureError("opmodel.dev/modules/test/podinfo@v0", "v0.9.9",
		fmt.Errorf("module not found"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "publish the module")
	assert.NotContains(t, err.Error(), "identity mismatch")
}
