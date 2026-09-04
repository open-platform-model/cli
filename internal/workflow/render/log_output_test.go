package render

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/open-platform-model/library/opm/kernel"

	"github.com/open-platform-model/cli/internal/output"
	"github.com/open-platform-model/cli/pkg/module"
)

func TestWriteTransformerMatches_NoPairs(t *testing.T) {
	result := &Result{Instance: module.InstanceMetadata{Name: "test", Namespace: "default"}}
	assert.NotPanics(t, func() { writeTransformerMatches(result) })
}

func TestWriteVerboseMatchLog_NoPairs(t *testing.T) {
	result := &Result{Instance: module.InstanceMetadata{Name: "test", Namespace: "default"}}
	assert.NotPanics(t, func() { writeVerboseMatchLog(result) })
}

func TestWriteTransformerMatches_PrintsEachPair(t *testing.T) {
	var buf bytes.Buffer
	output.SetupLogging(output.LogConfig{})
	output.SetLogWriter(&buf)

	writeTransformerMatches(&Result{
		Instance: module.InstanceMetadata{Name: "test", Namespace: "default"},
		Pairs: []kernel.RenderPair{
			{Component: "web", Transformer: "opmodel.dev/catalogs/opm@v4#DeploymentTransformer"},
			{Component: "web", Transformer: "opmodel.dev/catalogs/opm@v4#ServiceTransformer"},
		},
	})

	got := buf.String()
	assert.Contains(t, got, "DeploymentTransformer")
	assert.Contains(t, got, "ServiceTransformer")
	assert.Contains(t, got, "web")
}
