package render

import (
	"bytes"
	"io"
	"os"
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"github.com/opmodel/cli/internal/output"
	pkgrender "github.com/opmodel/cli/pkg/render"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPrintValidationError_UsesGroupedFormatting(t *testing.T) {
	ctx := cuecontext.New()
	schema := ctx.CompileString(`close({
		media?: [Name=string]: {
			mountPath: string
			type:      "pvc" | *"emptyDir"
			size:      string
		}
	})`, cue.Filename("module.cue"))
	values := ctx.CompileString(`{
		test: "test"
		media: {
			test: "test"
		}
	}`, cue.Filename("values.cue"))

	_, cfgErr := pkgrender.ValidateConfig(schema, []cue.Value{values}, "module", "demo")
	require.NotNil(t, cfgErr)

	var logBuf bytes.Buffer
	output.SetupLogging(output.LogConfig{})
	output.SetLogWriter(&logBuf)

	oldStderr := os.Stderr
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stderr = w
	defer func() { os.Stderr = oldStderr }()

	printValidationError("render failed", cfgErr)
	require.NoError(t, w.Close())

	details, err := io.ReadAll(r)
	require.NoError(t, err)
	require.NoError(t, r.Close())

	assert.Contains(t, logBuf.String(), "render failed: 2 issues")
	assert.Contains(t, string(details), "field not allowed")
	assert.Contains(t, string(details), "values.test")
	assert.Contains(t, string(details), "> values.cue:2:3")
	assert.Contains(t, string(details), "> values.cue:4:10")
	assert.Contains(t, string(details), "conflicting values \"test\"")
	assert.NotContains(t, logBuf.String(), "values do not satisfy #config")
}
