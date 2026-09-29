package instinit

import (
	"testing"

	"cuelang.org/go/cue/cuecontext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
			name:   "concrete non-struct rendered verbatim",
			src:    `debugValues: [1, 2]`,
			want:   "[1, 2]",
			source: FromDebugValues,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			pkg := cuecontext.New().CompileString(tc.src)
			require.NoError(t, pkg.Err())

			got, source, err := PickValues(pkg)
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
	values, source, err := PickValues(pkg)
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
