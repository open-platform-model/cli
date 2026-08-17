package scaffold

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestParseTemplateRef pins the shortcut grammar: bare words expand into the
// reserved segment, anything with '/' or '.' is a literal path and never
// expands, majors float, exact semvers pin.
func TestParseTemplateRef(t *testing.T) {
	tests := []struct {
		name string
		ref  string
		want TemplateRef
	}{
		{
			name: "bare official word takes the table's default major",
			ref:  "standard",
			want: TemplateRef{Base: "opmodel.dev/templates/standard", Major: "v1", Shortcut: true},
		},
		{
			name: "bare unknown word expands unconstrained — the typo fails at resolution, inside the segment",
			ref:  "standrad",
			want: TemplateRef{Base: "opmodel.dev/templates/standrad", Shortcut: true},
		},
		{
			name: "word with underscores and digits is a word",
			ref:  "my_template2",
			want: TemplateRef{Base: "opmodel.dev/templates/my_template2", Shortcut: true},
		},
		{
			name: "explicit major floats within it",
			ref:  "standard@v2",
			want: TemplateRef{Base: "opmodel.dev/templates/standard", Major: "v2", Shortcut: true},
		},
		{
			name: "exact semver pins the tag",
			ref:  "standard@1.2.3",
			want: TemplateRef{Base: "opmodel.dev/templates/standard", Major: "v1", Exact: "1.2.3", Shortcut: true},
		},
		{
			name: "exact prerelease pins the tag",
			ref:  "minimal@2.0.0-alpha.1",
			want: TemplateRef{Base: "opmodel.dev/templates/minimal", Major: "v2", Exact: "2.0.0-alpha.1", Shortcut: true},
		},
		{
			name: "literal path is never expanded",
			ref:  "example.com/modules/donor@v2",
			want: TemplateRef{Base: "example.com/modules/donor", Major: "v2"},
		},
		{
			name: "literal path without a suffix floats unconstrained",
			ref:  "example.com/modules/donor",
			want: TemplateRef{Base: "example.com/modules/donor"},
		},
		{
			name: "literal path with an exact version",
			ref:  "example.com/modules/donor@2.4.1",
			want: TemplateRef{Base: "example.com/modules/donor", Major: "v2", Exact: "2.4.1"},
		},
		{
			name: "a dotted single segment is a path, not a word",
			ref:  "opmodel.dev",
			want: TemplateRef{Base: "opmodel.dev"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseTemplateRef(tt.ref)
			require.NoError(t, err)
			tt.want.Raw = tt.ref
			assert.Equal(t, tt.want, got)
		})
	}

	invalid := []struct{ name, ref string }{
		{"empty", ""},
		{"empty head", "@v1"},
		{"uppercase is neither word nor path", "Standard"},
		{"hyphenated is neither word nor path", "my-template"},
		{"bad suffix", "standard@latest"},
		{"v-prefixed semver suffix", "standard@v1.2.3"},
	}
	for _, tt := range invalid {
		t.Run("invalid: "+tt.name, func(t *testing.T) {
			_, err := ParseTemplateRef(tt.ref)
			assert.Error(t, err)
		})
	}
}

// TestLookupPath pins how the resolution scope is built from the parse.
func TestLookupPath(t *testing.T) {
	ref, err := ParseTemplateRef("standard")
	require.NoError(t, err)
	assert.Equal(t, "opmodel.dev/templates/standard@v1", ref.LookupPath())

	ref, err = ParseTemplateRef("standrad")
	require.NoError(t, err)
	assert.Equal(t, "opmodel.dev/templates/standrad", ref.LookupPath())

	ref, err = ParseTemplateRef("example.com/modules/donor@2.4.1")
	require.NoError(t, err)
	assert.Equal(t, "example.com/modules/donor@v2", ref.LookupPath())
}

// TestOfficialTableDrivesExpansion asserts the baked table and the shortcut
// grammar share one source: every listed template expands into the reserved
// segment with its listed default major (the `template list` ↔ expansion
// coupling the spec requires).
func TestOfficialTableDrivesExpansion(t *testing.T) {
	require.NotEmpty(t, Official)
	for _, tpl := range Official {
		ref, err := ParseTemplateRef(tpl.Name)
		require.NoError(t, err, tpl.Name)
		assert.True(t, ref.Shortcut)
		assert.Equal(t, Segment+"/"+tpl.Name, ref.Base)
		assert.Equal(t, tpl.DefaultMajor, ref.Major)
	}
	// The default template is an official one.
	assert.NotNil(t, officialByName(DefaultTemplate))
}

func TestValidateNewModulePath(t *testing.T) {
	valid := []string{
		"example.com/modules/my_app@v0",
		"github.com/acme/deep/nested/app2@v12",
		"testing.opmodel.dev/modules/web_app@v1",
	}
	for _, p := range valid {
		assert.NoError(t, ValidateNewModulePath(p), p)
	}

	invalid := []string{
		"",
		"my_app",                        // bare word
		"example.com/modules/my_app",    // no major
		"example.com/modules/my-app@v0", // hyphenated leaf cannot be a package name
		"example.com/modules/2app@v0",   // leaf starts with a digit
		"example.com@v1",                // no name segment
		"example.com/modules/app@1.2.3", // exact version is not a major
	}
	for _, p := range invalid {
		assert.Error(t, ValidateNewModulePath(p), p)
	}
}

func TestLeaf(t *testing.T) {
	assert.Equal(t, "my_app", Leaf("example.com/modules/my_app@v0"))
	assert.Equal(t, "standard", Leaf("opmodel.dev/templates/standard"))
}
