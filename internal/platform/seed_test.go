package platform

import (
	"testing"

	"cuelang.org/go/cue/cuecontext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/library/opm/kernel"
)

// The values below mimic the shape of an acquired platform: metadata, type
// and #registry entries whose enable and version are concrete, as they are
// once the imported catalog fills them in a real build.

func TestSpecFromPlatform_DecodesDerivedEntries(t *testing.T) {
	k := kernel.New()
	v := cuecontext.New().CompileString(`{
		kind: "Platform"
		metadata: name: "cluster"
		type: "kubernetes"
		#registry: {
			"opmodel.dev/catalogs/opm@v4": {enable: bool | *true, version: "4.0.1"}
			"opmodel.dev/catalogs/k8s@v1": {enable: false, version: "1.0.0-alpha.2"}
		}
	}`)
	require.NoError(t, v.Err())
	p, err := k.NewPlatformFromValue(v)
	require.NoError(t, err)

	s, err := SpecFromPlatform(p)
	require.NoError(t, err)

	assert.Equal(t, "cluster", s.Name)
	assert.Equal(t, "kubernetes", s.Type)
	assert.Empty(t, s.SkewPolicy)
	assert.Equal(t, []Entry{
		{Path: "opmodel.dev/catalogs/k8s@v1", Version: "1.0.0-alpha.2", Enable: false},
		{Path: "opmodel.dev/catalogs/opm@v4", Version: "4.0.1", Enable: true},
	}, s.Entries, "one entry per #registry key, sorted, with the derived version and the defaulted enable")
}

func TestSpecFromPlatform_EmptyRegistry(t *testing.T) {
	k := kernel.New()
	v := cuecontext.New().CompileString(`{
		kind: "Platform"
		metadata: name: "cluster"
		type: "kubernetes"
		#registry: {}
	}`)
	require.NoError(t, v.Err())
	p, err := k.NewPlatformFromValue(v)
	require.NoError(t, err)

	s, err := SpecFromPlatform(p)
	require.NoError(t, err)
	assert.Empty(t, s.Entries)
}

func TestSpecFromPlatform_NonConcreteVersionRefused(t *testing.T) {
	// A version the build did not derive (no imported catalog) must never
	// seed an empty pin.
	k := kernel.New()
	v := cuecontext.New().CompileString(`{
		kind: "Platform"
		metadata: name: "cluster"
		type: "kubernetes"
		#registry: "opmodel.dev/catalogs/opm@v4": {enable: true, version: string}
	}`)
	require.NoError(t, v.Err())
	p, err := k.NewPlatformFromValue(v)
	require.NoError(t, err)

	_, err = SpecFromPlatform(p)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "opmodel.dev/catalogs/opm@v4")
}

func TestSpecFromPlatform_NilPlatform(t *testing.T) {
	_, err := SpecFromPlatform(nil)
	require.Error(t, err)
}
