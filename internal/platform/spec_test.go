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

// crDoc is a Platform document as the cluster getter returns it.
func crDoc(spec, status map[string]any, generation int64) *ClusterPlatform {
	return &ClusterPlatform{Name: "cluster", Generation: generation, Spec: spec, Status: status}
}

func specWithOneSubscription() map[string]any {
	return map[string]any{
		"type": "kubernetes",
		"registry": map[string]any{
			"opmodel.dev/catalogs/opm@v4": map[string]any{"version": "4.4.0"},
		},
	}
}

func TestDecodeCR_EffectiveRegistryCarriesBothSources(t *testing.T) {
	spec, eff, err := DecodeCR(crDoc(specWithOneSubscription(), map[string]any{
		"observedGeneration": int64(7),
		"packageIdentity":    "gen-7-3f9a1c2b",
		"operatorVersion":    "v1.0.0-alpha.20",
		"registry": []any{
			map[string]any{
				"catalog": "opmodel.dev/catalogs/k8up@v1",
				"version": "1.2.0",
				"enabled": true,
				"source":  EntrySourceRegistration,
			},
			map[string]any{
				"catalog": "opmodel.dev/catalogs/opm@v4",
				"version": "4.4.0",
				"enabled": true,
				"source":  EntrySourceSubscription,
			},
		},
		"conditions": []any{
			map[string]any{"type": "Ready", "status": "True", "reason": "Generated"},
		},
	}, 7))
	require.NoError(t, err)

	// The spec half is untouched: the authored subscription only.
	require.Len(t, spec.Entries, 1)
	assert.Equal(t, "opmodel.dev/catalogs/opm@v4", spec.Entries[0].Path)

	require.NotNil(t, eff)
	require.Equal(t, []Entry{
		{Path: "opmodel.dev/catalogs/k8up@v1", Version: "1.2.0", Enable: true},
		{Path: "opmodel.dev/catalogs/opm@v4", Version: "4.4.0", Enable: true},
	}, eff.Entries, "effective entries are sorted by path")
	assert.Equal(t, map[string]string{
		"opmodel.dev/catalogs/k8up@v1": EntrySourceRegistration,
		"opmodel.dev/catalogs/opm@v4":  EntrySourceSubscription,
	}, eff.Sources)
	assert.Equal(t, "gen-7-3f9a1c2b", eff.PackageIdentity)
	assert.Equal(t, int64(7), eff.ObservedGeneration)
	assert.Equal(t, "v1.0.0-alpha.20", eff.OperatorVersion)
	require.NotNil(t, eff.Ready)
	assert.Equal(t, "True", eff.Ready.Status)
	assert.Equal(t, "Generated", eff.Ready.Reason)
}

// A disabled catalog is recorded with enabled omitted (the operator's
// omitzero), so the decode default must be false — the opposite of
// spec.registry's enable.
func TestDecodeCR_OmittedEnabledIsFalse(t *testing.T) {
	_, eff, err := DecodeCR(crDoc(specWithOneSubscription(), map[string]any{
		"registry": []any{
			map[string]any{
				"catalog": "opmodel.dev/catalogs/k8s@v1",
				"version": "1.0.0-alpha.3",
				"source":  EntrySourceSubscription,
			},
		},
	}, 3))
	require.NoError(t, err)
	require.NotNil(t, eff)
	require.Len(t, eff.Entries, 1)
	assert.False(t, eff.Entries[0].Enable, "an omitted enabled means a disabled catalog")
}

func TestDecodeCR_NoStatusMeansNoEffective(t *testing.T) {
	spec, eff, err := DecodeCR(crDoc(specWithOneSubscription(), nil, 1))
	require.NoError(t, err)
	assert.Nil(t, eff, "a Platform no operator reconciled has no effective registry")
	assert.Equal(t, "kubernetes", spec.Type)
}

// A status without a registry still carries what the operator recorded, so
// the operator version and the observed generation stay reportable.
func TestDecodeCR_StatusWithoutRegistry(t *testing.T) {
	_, eff, err := DecodeCR(crDoc(specWithOneSubscription(), map[string]any{
		"observedGeneration": int64(2),
		"operatorVersion":    "v1.0.0-alpha.20",
	}, 2))
	require.NoError(t, err)
	require.NotNil(t, eff)
	assert.Empty(t, eff.Entries, "no recorded registry: resolution falls back to the spec")
	assert.Equal(t, "v1.0.0-alpha.20", eff.OperatorVersion)
}

func TestDecodeCR_ReadyFalseIsDecoded(t *testing.T) {
	_, eff, err := DecodeCR(crDoc(specWithOneSubscription(), map[string]any{
		"registry": []any{
			map[string]any{
				"catalog": "opmodel.dev/catalogs/opm@v4",
				"version": "4.4.0",
				"enabled": true,
				"source":  EntrySourceSubscription,
			},
		},
		"conditions": []any{
			map[string]any{"type": "ContractsFulfilled", "status": "True", "reason": "ContractsFulfilled"},
			map[string]any{"type": "Ready", "status": "False", "reason": "OverSubscribedContracts"},
		},
	}, 4))
	require.NoError(t, err)
	require.NotNil(t, eff)
	require.NotNil(t, eff.Ready)
	assert.Equal(t, "False", eff.Ready.Status)
	assert.Equal(t, "OverSubscribedContracts", eff.Ready.Reason)
}

// The operator never records a row without a version, so one means the
// status was written by hand; generating from it would drop the pin.
func TestDecodeCR_StatusEntryWithoutVersionRefused(t *testing.T) {
	_, _, err := DecodeCR(crDoc(specWithOneSubscription(), map[string]any{
		"registry": []any{
			map[string]any{
				"catalog": "opmodel.dev/catalogs/opm@v4",
				"source":  EntrySourceSubscription,
			},
		},
	}, 1))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "opmodel.dev/catalogs/opm@v4")
	assert.Contains(t, err.Error(), "no version")
}

func TestDecodeCR_SpecFailureSurfaces(t *testing.T) {
	_, _, err := DecodeCR(crDoc(map[string]any{}, nil, 1))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "spec.type")
}
