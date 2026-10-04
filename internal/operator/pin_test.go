package operator

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Until the embedded manifest is retired, the operator version the pinned
// module deploys must be the release whose manifest is embedded, so the docs
// pins and the release-pin check stay true.
func TestPin_OperatorVersionMatchesTheEmbeddedManifest(t *testing.T) {
	assert.Equal(t, PinnedOperatorVersion, pinnedModuleOperatorVersion)
}
