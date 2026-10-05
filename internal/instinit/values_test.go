package instinit

import (
	"context"
	"path/filepath"
	"testing"

	"cuelang.org/go/cue/cuecontext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/library/opm/module"

	"github.com/open-platform-model/cli/internal/config"
)

// TestPickValues_AcquiredModule runs the ladder over a real module acquired
// through the kernel: initValues wins, none of the module's debugValues reach
// values.cue, and the non-concrete initValues render with the default
// resolved, the disjunction kept and the optional field omitted.
func TestPickValues_AcquiredModule(t *testing.T) {
	dir, err := filepath.Abs(filepath.Join("testdata", "initvalues"))
	require.NoError(t, err)
	k := config.NewKernel(config.DefaultRegistry)
	if _, err := k.SchemaCache().Get(); err != nil {
		t.Skipf("core v2 schema unavailable (registry/cache): %v", err)
	}
	mod, err := k.AcquireModuleFromDir(context.Background(), dir)
	require.NoError(t, err)

	values, source, err := PickValues(mod)
	require.NoError(t, err)
	assert.Equal(t, FromInitValues, source)

	in := webAppInput(t)
	in.Values, in.Source = values, source
	files, err := Render(in)
	require.NoError(t, err)
	got := string(files[ValuesFile])
	assert.Equal(t, `// Starting values from the module's initValues. Edit them for this instance.
package instance

values: {
	replicas: 2
	logLevel: "info" | "debug"
}
`, got)
	assert.NotContains(t, got, "debug.example", "no debugValues content reaches the package")
	assert.NotContains(t, got, "8080")
}

func TestPickValues(t *testing.T) {
	for _, tc := range []struct {
		name      string
		src       string
		want      string
		source    ValuesSource
		wantEmpty bool
	}{
		{
			name:   "concrete struct debugValues",
			src:    `debugValues: {image: "nginx:1.27", replicas: 2}`,
			want:   "{\n\timage:    \"nginx:1.27\"\n\treplicas: 2\n}",
			source: FromDebugValues,
		},
		{
			name:   "defaulted field counts as concrete",
			src:    `debugValues: {image: "nginx:1.27", replicas: int | *1}`,
			want:   "{\n\timage:    \"nginx:1.27\"\n\treplicas: 1\n}",
			source: FromDebugValues,
		},
		{
			name:      "top debugValues falls to empty",
			src:       `debugValues: _`,
			want:      "{}",
			source:    FromEmpty,
			wantEmpty: true,
		},
		{
			name:      "partly concrete debugValues falls to empty",
			src:       `debugValues: {image: "nginx:1.27", replicas: int}`,
			want:      "{}",
			source:    FromEmpty,
			wantEmpty: true,
		},
		{
			name:      "undefaulted disjunction falls to empty",
			src:       `debugValues: {mode: "a" | "b"}`,
			want:      "{}",
			source:    FromEmpty,
			wantEmpty: true,
		},
		{
			name:      "empty debugValues keeps its source name",
			src:       `debugValues: {}`,
			want:      "{}",
			source:    FromDebugValues,
			wantEmpty: true,
		},
		{
			name:      "absent debugValues",
			src:       `#config: {replicas: int}`,
			want:      "{}",
			source:    FromEmpty,
			wantEmpty: true,
		},
		{
			name:   "initValues wins over debugValues",
			src:    `initValues: {replicas: 2}, debugValues: {replicas: 7, image: "debug"}`,
			want:   "{\n\treplicas: 2\n}",
			source: FromInitValues,
		},
		{
			name:   "non-concrete initValues rendered: default, disjunction, no optional",
			src:    `initValues: {replicas: *2 | int, logLevel: "info" | "debug", port?: int}`,
			want:   "{\n\treplicas: 2\n\tlogLevel: \"info\" | \"debug\"\n}",
			source: FromInitValues,
		},
		{
			name:   "initValues optional in the schema and unset",
			src:    `#M: {initValues?: _, debugValues?: _}, #M & {debugValues: {replicas: 1}}`,
			want:   "{\n\treplicas: 1\n}",
			source: FromDebugValues,
		},
		{
			name:      "empty initValues keeps its source name",
			src:       `initValues: {}, debugValues: {replicas: 1}`,
			want:      "{}",
			source:    FromInitValues,
			wantEmpty: true,
		},
		{
			name:   "concrete non-struct rendered verbatim",
			src:    `debugValues: [1, 2]`,
			want:   "[1, 2]",
			source: FromDebugValues,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			pkg := cuecontext.New().CompileString(tc.src)
			require.NoError(t, pkg.Err())

			got, source, err := PickValues(&module.Module{Package: pkg})
			require.NoError(t, err)
			assert.Equal(t, tc.source, source)
			assert.Equal(t, tc.want, string(got))
			assert.Equal(t, tc.wantEmpty, IsEmpty(got))
		})
	}
}

func TestPickValues_RendersIntoValuesFile(t *testing.T) {
	pkg := cuecontext.New().CompileString(`debugValues: {image: {repository: "ghcr.io/x", tag: string | *"1.0"}, replicas: 1}`)
	require.NoError(t, pkg.Err())
	values, source, err := PickValues(&module.Module{Package: pkg})
	require.NoError(t, err)

	in := webAppInput(t)
	in.Values, in.Source = values, source
	files, err := Render(in)
	require.NoError(t, err)
	assert.Equal(t, `// Starting values from the module's debugValues, the author's test values.
// Review them before deploying.
package instance

values: {
	image: {
		repository: "ghcr.io/x"
		tag:        "1.0"
	}
	replicas: 1
}
`, string(files[ValuesFile]))
}

func TestIsEmpty(t *testing.T) {
	assert.True(t, IsEmpty([]byte("{}")))
	assert.True(t, IsEmpty([]byte("{\n}\n")))
	assert.False(t, IsEmpty([]byte("{a: 1}")))
	assert.False(t, IsEmpty([]byte("[]")))
}

// TestPickValues_NilModule: a nil module has neither initValues nor
// debugValues, so the ladder walks to empty.
func TestPickValues_NilModule(t *testing.T) {
	values, source, err := PickValues(nil)
	require.NoError(t, err)
	assert.Equal(t, FromEmpty, source)
	assert.True(t, IsEmpty(values))
}
