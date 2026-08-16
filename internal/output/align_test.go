package output

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAlignColumns(t *testing.T) {
	tests := []struct {
		name string
		rows [][]string
		want string
	}{
		{
			name: "two-value disagreement",
			rows: [][]string{
				{"declared", "opmodel.dev/catalogs/opm", "src/identity/identity.cue:22"},
				{"cue.mod", "opmodel.dev/catalogs/opm-x", "src/cue.mod/module.cue:1"},
			},
			want: "  declared  opmodel.dev/catalogs/opm    src/identity/identity.cue:22\n" +
				"  cue.mod   opmodel.dev/catalogs/opm-x  src/cue.mod/module.cue:1",
		},
		{
			name: "ragged rows do not pad the last cell",
			rows: [][]string{
				{"--version", "2.4.1"},
				{"declared", "2.4.0", "identity/identity.cue:18"},
			},
			want: "  --version  2.4.1\n" +
				"  declared   2.4.0  identity/identity.cue:18",
		},
		{
			name: "single row",
			rows: [][]string{{"Version", "identity/identity.cue:9", "declared, never filled"}},
			want: "  Version  identity/identity.cue:9  declared, never filled",
		},
		{
			name: "empty",
			rows: nil,
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, AlignColumns("  ", tt.rows))
		})
	}
}
