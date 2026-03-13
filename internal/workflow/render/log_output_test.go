package render

import (
	"testing"

	pkgrender "github.com/opmodel/cli/pkg/render"
	"github.com/stretchr/testify/assert"
)

func TestWriteTransformerMatches_NilMatchPlan(t *testing.T) {
	result := &Result{Release: pkgrender.ModuleReleaseMetadata{Name: "test", Namespace: "default"}}
	writeTransformerMatches(result)
}

func TestWriteVerboseMatchLog_NilMatchPlan(t *testing.T) {
	result := &Result{Release: pkgrender.ModuleReleaseMetadata{Name: "test", Namespace: "default"}}
	writeVerboseMatchLog(result)
}

func TestFormatFQNList_Empty(t *testing.T) {
	assert.Equal(t, "", formatFQNList(nil))
	assert.Equal(t, "", formatFQNList([]string{}))
}

func TestFormatFQNList_Single(t *testing.T) {
	result := formatFQNList([]string{"example.com/resources/workload/container@v1"})
	assert.NotEmpty(t, result)
	assert.Contains(t, result, "container")
}
