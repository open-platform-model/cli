package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/cli/internal/operator/operatortest"
)

// pinRegistry serves three operator module versions: 0.1.0 and 0.2.0 deploy
// beta releases of 1.0, 0.3.0 deploys 1.1.0, 0.4.0 states no operator.
func pinRegistry(t *testing.T) string {
	t.Helper()
	return operatortest.Registry(t,
		operatortest.Version{Module: "0.1.0", Operator: "1.0.0-beta.5"},
		operatortest.Version{Module: "0.2.0", Operator: "1.0.0-beta.6"},
		operatortest.Version{Module: "0.3.0", Operator: "1.1.0"},
		operatortest.Version{Module: "0.4.0", OperatorPackage: "-"},
	)
}

func runTool(t *testing.T, mapping, file string, args ...string) (code int, stdout, stderr string) {
	t.Helper()
	var out, errOut bytes.Buffer
	code = run(context.Background(), args, &out, &errOut, mapping, file)
	return code, out.String(), errOut.String()
}

// "Pin refresh is one reviewable diff": the program writes exactly the
// golden file.
func TestPin_WritesTheGoldenFile(t *testing.T) {
	file := filepath.Join(t.TempDir(), "pin.go")

	code, stdout, stderr := runTool(t, pinRegistry(t), file, "0.2.0")
	require.Equal(t, 0, code, stderr)
	assert.Contains(t, stdout, "0.2.0 (deploys opm-operator v1.0.0-beta.6)")

	got, err := os.ReadFile(file)
	require.NoError(t, err)
	want, err := os.ReadFile(filepath.Join("testdata", "pin.go.golden"))
	require.NoError(t, err)
	assert.Equal(t, string(want), string(got))
}

// "A module without a readable operator version is refused by the pin
// task": non-zero, naming the version, and the pin file unchanged.
func TestPin_UnreadableOperatorVersionLeavesThePin(t *testing.T) {
	file := filepath.Join(t.TempDir(), "pin.go")
	require.NoError(t, os.WriteFile(file, []byte("unchanged\n"), 0o644))

	for _, version := range []string{"0.4.0", "0.9.0"} {
		code, _, stderr := runTool(t, pinRegistry(t), file, version)
		assert.Equal(t, 1, code)
		assert.Contains(t, stderr, version)
		got, err := os.ReadFile(file)
		require.NoError(t, err)
		assert.Equal(t, "unchanged\n", string(got))
	}
}

func TestPin_Check(t *testing.T) {
	reg := pinRegistry(t)
	dir := t.TempDir()
	write := func(module, op string) string {
		sub := filepath.Join(dir, module+op)
		require.NoError(t, os.MkdirAll(sub, 0o755))
		file := filepath.Join(sub, "pin.go")
		data, err := renderPin(module, op)
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(file, data, 0o644))
		return file
	}

	code, stdout, stderr := runTool(t, reg, write("0.2.0", "v1.0.0-beta.6"), "--check")
	assert.Equal(t, 0, code, stderr)
	assert.Contains(t, stdout, "consistent")

	// The recorded operator version is out of step with the module.
	code, _, stderr = runTool(t, reg, write("0.2.0", "v1.0.0-beta.5"), "--check")
	assert.Equal(t, 1, code)
	assert.Contains(t, stderr, "v1.0.0-beta.5")
	assert.Contains(t, stderr, "v1.0.0-beta.6")
	assert.Contains(t, stderr, "task operator:pin VERSION=0.2.0")

	// The pinned module version is not served.
	code, _, stderr = runTool(t, reg, write("0.9.0", "v1.0.0"), "--check")
	assert.Equal(t, 1, code)
	assert.Contains(t, stderr, "pin.go")
	assert.Contains(t, stderr, "opmodel.dev/modules/opm_operator 0.9.0 is not served")

	// An unreadable pin is a failure, not a pass.
	code, _, stderr = runTool(t, reg, filepath.Join(dir, "missing.go"), "--check")
	assert.Equal(t, 1, code)
	assert.Contains(t, stderr, "reading the pin")

	// A registry that cannot be reached is a failure, not a pass.
	code, _, stderr = runTool(t, "opmodel.dev=127.0.0.1:1+insecure", write("0.2.0", "v1.0.0-beta.6"), "--check")
	assert.Equal(t, 1, code)
	assert.Contains(t, stderr, "looking up")
}

func TestPin_Select(t *testing.T) {
	reg := pinRegistry(t)
	file := filepath.Join(t.TempDir(), "pin.go")

	// The newest at or below the bound whose operator the cli can drive.
	code, stdout, stderr := runTool(t, reg, file, "select", "0.3.0", "1.0.0-beta.9")
	assert.Equal(t, 0, code, stderr)
	assert.Equal(t, "0.2.0\n", stdout)

	code, stdout, _ = runTool(t, reg, file, "select", "0.3.0", "1.1.0")
	assert.Equal(t, 0, code)
	assert.Equal(t, "0.3.0\n", stdout)

	code, stdout, _ = runTool(t, reg, file, "select", "0.2.0", "1.1.0")
	assert.Equal(t, 0, code)
	assert.Equal(t, "0.2.0\n", stdout)

	code, stdout, _ = runTool(t, reg, file, "select", "0.3.0", "0.9.0")
	assert.Equal(t, exitNone, code)
	assert.Empty(t, stdout)
}

func TestPin_Usage(t *testing.T) {
	code, _, stderr := runTool(t, "opmodel.dev=127.0.0.1:1+insecure", "unused", "--bogus")
	assert.Equal(t, 1, code)
	assert.Contains(t, stderr, "usage: operator-pin")
}
