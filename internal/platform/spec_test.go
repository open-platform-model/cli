package platform

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDecodeCRSpec_DecodesWireShape(t *testing.T) {
	spec := map[string]any{
		"type": "kubernetes",
		"registry": map[string]any{
			"opmodel.dev/catalogs/opm@v4": map[string]any{
				"enable":  true,
				"version": "4.0.1",
			},
			"opmodel.dev/catalogs/k8s@v1": map[string]any{
				"enable":  false,
				"version": "1.0.0-alpha.2",
			},
		},
		"skewPolicy": "Refuse",
	}

	s, err := DecodeCRSpec(spec, "cluster")
	require.NoError(t, err)

	assert.Equal(t, "cluster", s.Name)
	assert.Equal(t, "kubernetes", s.Type)
	assert.Equal(t, "Refuse", s.SkewPolicy)
	require.Equal(t, []Entry{
		{Path: "opmodel.dev/catalogs/k8s@v1", Version: "1.0.0-alpha.2", Enable: false},
		{Path: "opmodel.dev/catalogs/opm@v4", Version: "4.0.1", Enable: true},
	}, s.Entries, "entries are sorted by path")
}

func TestDecodeCRSpec_OmittedEnableIsTrue(t *testing.T) {
	s, err := DecodeCRSpec(map[string]any{
		"type": "kubernetes",
		"registry": map[string]any{
			"opmodel.dev/catalogs/opm@v4": map[string]any{"version": "4.0.1"},
		},
	}, "cluster")
	require.NoError(t, err)
	require.Len(t, s.Entries, 1)
	assert.True(t, s.Entries[0].Enable, "a nil enable resolves to the schema default")
	assert.Empty(t, s.SkewPolicy)
}

func TestDecodeCRSpec_LegacyFilterCRTolerated(t *testing.T) {
	// A stored CR from before the scalar-version shape carries filter and no
	// version. Decode must succeed (read tolerance is permanent); the empty
	// version passes through to fail only at generation.
	spec := map[string]any{
		"type": "kubernetes",
		"registry": map[string]any{
			"opmodel.dev/catalogs/opm": map[string]any{
				"filter": map[string]any{"range": ">=1.0.0-0 <2.0.0-0"},
			},
		},
	}

	s, err := DecodeCRSpec(spec, "cluster")
	require.NoError(t, err, "legacy filter-shaped CRs must decode")
	require.Len(t, s.Entries, 1)
	assert.Equal(t, "opmodel.dev/catalogs/opm", s.Entries[0].Path)
	assert.Empty(t, s.Entries[0].Version, "missing version decodes empty and fails at generation")
}

func TestDecodeCRSpec_MissingType(t *testing.T) {
	_, err := DecodeCRSpec(map[string]any{}, "cluster")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "spec.type")
}

func TestWireRoundTrip_SpecToWireToSpec(t *testing.T) {
	// spec → wire (write-if-absent doc) → spec must preserve the document,
	// with every entry's enable stated explicitly on the wire.
	in := Spec{
		Name: "cluster",
		Type: "kubernetes",
		Entries: []Entry{
			{Path: "opmodel.dev/catalogs/k8s@v1", Version: "1.0.0-alpha.2", Enable: false},
			{Path: "opmodel.dev/catalogs/opm@v4", Version: "4.0.1", Enable: true},
		},
	}

	w := wireFromSpec(in)
	assert.Equal(t, in.Type, w.Type)
	require.Len(t, w.Registry, 2)
	require.NotNil(t, w.Registry["opmodel.dev/catalogs/k8s@v1"].Enable)
	assert.False(t, *w.Registry["opmodel.dev/catalogs/k8s@v1"].Enable)
	assert.Equal(t, "4.0.1", w.Registry["opmodel.dev/catalogs/opm@v4"].Version)
	assert.Empty(t, w.SkewPolicy, "a seed never writes a skew policy")

	assert.Equal(t, in, w.toSpec("cluster"))
}
