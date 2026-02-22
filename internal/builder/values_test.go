package builder

import (
	"path/filepath"
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/opmodel/cli/internal/core/module"
	opmerrors "github.com/opmodel/cli/internal/errors"
)

// testdataPath returns the absolute path to a testdata file.
func testdataPath(t *testing.T, name string) string {
	t.Helper()
	p, err := filepath.Abs(filepath.Join("testdata", name))
	require.NoError(t, err)
	return p
}

// moduleWithValues returns a *module.Module with a pre-compiled Values field.
func moduleWithValues(ctx *cue.Context, valCUE string) *module.Module {
	v := ctx.CompileString(valCUE)
	return &module.Module{Values: v}
}

// moduleWithNoValues returns a *module.Module with an absent Values field.
func moduleWithNoValues() *module.Module {
	return &module.Module{} // Values is zero cue.Value — not Exists()
}

func TestSelectValues(t *testing.T) {
	ctx := cuecontext.New()

	t.Run("no files, module has values returns mod.Values", func(t *testing.T) {
		mod := moduleWithValues(ctx, `{image: "nginx:1.0"}`)

		got, err := selectValues(ctx, mod, nil)
		require.NoError(t, err)
		assert.True(t, got.Exists())

		// Should be the same structural value as mod.Values.
		img, err := got.LookupPath(cue.ParsePath("image")).String()
		require.NoError(t, err)
		assert.Equal(t, "nginx:1.0", img)
	})

	t.Run("no files, module has no values returns ValidationError", func(t *testing.T) {
		mod := moduleWithNoValues()

		_, err := selectValues(ctx, mod, nil)
		require.Error(t, err)

		var valErr *opmerrors.ValidationError
		require.ErrorAs(t, err, &valErr)
		assert.Contains(t, valErr.Message, "values.cue")
	})

	t.Run("one file provided extracts values field", func(t *testing.T) {
		mod := moduleWithNoValues()
		file := testdataPath(t, "values-a.cue")

		got, err := selectValues(ctx, mod, []string{file})
		require.NoError(t, err)
		assert.True(t, got.Exists())

		img, err := got.LookupPath(cue.ParsePath("image")).String()
		require.NoError(t, err)
		assert.Equal(t, "nginx:1.0", img)

		replicas, err := got.LookupPath(cue.ParsePath("replicas")).Int64()
		require.NoError(t, err)
		assert.Equal(t, int64(1), replicas)
	})

	t.Run("two files unified, result contains fields from both", func(t *testing.T) {
		mod := moduleWithNoValues()
		fileA := testdataPath(t, "values-a.cue")
		fileB := testdataPath(t, "values-b.cue")

		// values-a: image=nginx:1.0, replicas=1
		// values-b: image=nginx:1.0, replicas=1, port=8080 (adds a new field)
		// Unify produces the conjunction of both — all fields present.
		got, err := selectValues(ctx, mod, []string{fileA, fileB})
		require.NoError(t, err)
		assert.True(t, got.Exists())

		img, err := got.LookupPath(cue.ParsePath("image")).String()
		require.NoError(t, err)
		assert.Equal(t, "nginx:1.0", img)

		port, err := got.LookupPath(cue.ParsePath("port")).Int64()
		require.NoError(t, err)
		assert.Equal(t, int64(8080), port)
	})

	t.Run("file with no values field returns ValidationError", func(t *testing.T) {
		mod := moduleWithNoValues()
		file := testdataPath(t, "values-no-field.cue")

		_, err := selectValues(ctx, mod, []string{file})
		require.Error(t, err)

		var valErr *opmerrors.ValidationError
		require.ErrorAs(t, err, &valErr)
		assert.Contains(t, valErr.Message, "'values'")
	})
}
