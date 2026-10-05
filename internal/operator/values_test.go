package operator

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeValues(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "values.cue")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
	return path
}

// decoded returns the values the kernel source carries.
func decoded(t *testing.T, v *Values) map[string]any {
	t.Helper()
	var wrapper struct {
		Values map[string]any `json:"values"`
	}
	require.NoError(t, json.Unmarshal(v.Source.Data, &wrapper))
	return wrapper.Values
}

func TestMergeValues(t *testing.T) {
	recorded := map[string]any{
		"registry": "opmodel.dev=mirror.example",
		"replicas": int64(1),
		"resources": map[string]any{
			"limits": map[string]any{"memory": "4Gi", "cpu": int64(2)},
		},
		"extraArgs": []any{"--a"},
	}

	t.Run("a recorded value survives reinstall", func(t *testing.T) {
		v, err := MergeValues(ValuesInput{Recorded: recorded})
		require.NoError(t, err)
		assert.True(t, v.FromRecord)
		got := decoded(t, v)
		assert.Equal(t, "opmodel.dev=mirror.example", got["registry"])
		assert.Contains(t, v.Source.Origin, "values recorded on opm-operator-system/opm-operator")
	})

	t.Run("changing one value keeps the others", func(t *testing.T) {
		file := writeValues(t, "package x\n\nvalues: {\n\treplicas: 2\n\tresources: limits: memory: \"2Gi\"\n\textraArgs: [\"--b\"]\n}\n")
		v, err := MergeValues(ValuesInput{Recorded: recorded, Files: []string{file}})
		require.NoError(t, err)
		got := decoded(t, v)
		assert.Equal(t, float64(2), got["replicas"])
		assert.Equal(t, "opmodel.dev=mirror.example", got["registry"])
		limits := got["resources"].(map[string]any)["limits"].(map[string]any)
		assert.Equal(t, "2Gi", limits["memory"], "a scalar in a later source replaces")
		assert.Equal(t, float64(2), limits["cpu"], "maps merge: the recorded cpu limit stays")
		assert.Equal(t, []any{"--b"}, got["extraArgs"], "a list in a later source replaces")
		assert.Contains(t, v.Source.Origin, file)
	})

	t.Run("files layer in order, a file without a values field is the payload", func(t *testing.T) {
		first := writeValues(t, "replicas: 3\n")
		second := writeValues(t, "values: replicas: 4\n")
		v, err := MergeValues(ValuesInput{Files: []string{first, second}})
		require.NoError(t, err)
		assert.False(t, v.FromRecord)
		assert.Equal(t, float64(4), decoded(t, v)["replicas"])
	})

	t.Run("reset drops the recorded values", func(t *testing.T) {
		v, err := MergeValues(ValuesInput{Recorded: recorded, Reset: true})
		require.NoError(t, err)
		assert.False(t, v.FromRecord)
		assert.Empty(t, decoded(t, v))
		assert.Equal(t, `{"values":{}}`, string(v.Source.Data))
	})

	t.Run("a values file that is not concrete is refused naming it", func(t *testing.T) {
		file := writeValues(t, "values: replicas: int\n")
		_, err := MergeValues(ValuesInput{Files: []string{file}})
		require.Error(t, err)
		assert.Contains(t, err.Error(), file)
		assert.Contains(t, err.Error(), "not concrete")
	})
}
