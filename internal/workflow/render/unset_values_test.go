package render

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/cli/internal/config"
	opmexit "github.com/open-platform-model/cli/internal/exit"
)

// TestFromModule_UnsetRequiredValuesAreNamed holds what a render prints when
// the values leave required #config values unset: one finding for each unset
// value at values.<field>, whether a component reads it (note) or not
// (other), and no finding at the place inside a component that reads it. The
// refusal is the kernel's; the instance commands reach the same kernel check
// through AcquireInstanceFromDir and print it the same way.
func TestFromModule_UnsetRequiredValuesAreNamed(t *testing.T) {
	const registry = "opmodel.dev=ghcr.io/open-platform-model,registry.cue.works"
	if _, err := config.NewKernel(registry).SchemaCache().Get(); err != nil {
		t.Skipf("core v2 schema unavailable (registry/cache): %v", err)
	}
	fixture, err := filepath.Abs(filepath.Join("..", "..", "..", "tests", "fixtures", "valid", "simple-module"))
	require.NoError(t, err)
	dir := filepath.Join(t.TempDir(), "unset")
	require.NoError(t, os.CopyFS(dir, os.DirFS(fixture)))
	modFile := filepath.Join(dir, "module.cue")
	src, err := os.ReadFile(modFile)
	require.NoError(t, err)
	head, _, found := strings.Cut(string(src), "// Configuration schema with defaults.")
	require.True(t, found, "the fixture's #config comment moved")
	body := head + `#config: {
	replicas: *1 | int
	note:     string
	other:    int
}

debugValues: {}

#components: foo: metadata: {name: "foo", annotations: note: #config.note}
`
	require.NoError(t, os.WriteFile(modFile, []byte(body), 0o600))
	logBuf := captureRenderLog(t)

	// The grouped findings go to the process's standard error.
	stderr := os.Stderr
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stderr = w
	_, err = FromModule(context.Background(), ModuleOpts{
		ModulePath: dir,
		Config:     &config.GlobalConfig{ConfigPath: filepath.Join(t.TempDir(), "config.cue"), Registry: registry},
		K8sConfig:  &config.ResolvedKubernetesConfig{},
	})
	os.Stderr = stderr
	require.NoError(t, w.Close())
	findings, readErr := io.ReadAll(r)
	require.NoError(t, readErr)

	var exitErr *opmexit.ExitError
	require.ErrorAs(t, err, &exitErr)
	assert.Equal(t, opmexit.ExitValidationError, exitErr.Code)
	assert.Contains(t, logBuf.String(), "render failed: 2 issues")
	printed := string(findings)
	assert.Contains(t, printed, "values.note")
	assert.Contains(t, printed, "values.other")
	assert.Contains(t, printed, "incomplete value string")
	assert.Contains(t, printed, "incomplete value int")
	assert.NotContains(t, printed, "components.foo", "the place that reads the value is not a finding")
	assert.NotContains(t, printed, "values.replicas", "a defaulted value is not a finding")
}
