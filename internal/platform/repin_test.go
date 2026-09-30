package platform

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/open-platform-model/library/opm/schema"
)

func TestCoreRepinHint(t *testing.T) {
	hint := CoreRepinHint("/work/platform")
	assert.Contains(t, hint, "the platform module at /work/platform")
	assert.Contains(t, hint, "'cue mod get opmodel.dev/core@"+schema.DefaultSchemaVersion()+"' in /work/platform")
}
