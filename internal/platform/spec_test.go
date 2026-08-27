package platform

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/library/opm/helper/synth"

	"github.com/open-platform-model/cli/internal/config"
)

// writePlatformFile writes content as platform.cue in a fresh temp dir and
// returns its path.
func writePlatformFile(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "platform.cue")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	return path
}

func TestDecodeFile_DefaultTemplate(t *testing.T) {
	// The seeded template must decode into a full PlatformInput.
	path := writePlatformFile(t, config.DefaultPlatformTemplate)

	in, err := DecodeFile(path)
	require.NoError(t, err)

	assert.Equal(t, "cluster", in.Name)
	assert.Equal(t, "kubernetes", in.Type)
	require.Len(t, in.Subscriptions, 2)

	opm, ok := in.Subscriptions["opmodel.dev/catalogs/opm@v2"]
	require.True(t, ok)
	assert.Nil(t, opm.Enable, "omitted enable defers to the schema default")
	assert.Equal(t, "2.0.0-alpha.6", opm.Version)

	k8s, ok := in.Subscriptions["opmodel.dev/catalogs/k8s@v1"]
	require.True(t, ok)
	assert.Nil(t, k8s.Enable, "omitted enable defers to the schema default")
	assert.Equal(t, "1.0.0-alpha.1", k8s.Version)
}

func TestDecodeFile_ExplicitEnableAndVersion(t *testing.T) {
	path := writePlatformFile(t, `name: "cluster"
type: "kubernetes"
registry: {
	"opmodel.dev/catalogs/opm@v2": {
		enable:  false
		version: "2.0.0-alpha.3"
	}
}
`)

	in, err := DecodeFile(path)
	require.NoError(t, err)

	sub := in.Subscriptions["opmodel.dev/catalogs/opm@v2"]
	require.NotNil(t, sub.Enable)
	assert.False(t, *sub.Enable)
	assert.Equal(t, "2.0.0-alpha.3", sub.Version)
}

func TestDecodeFile_InvalidFileRejected(t *testing.T) {
	path := writePlatformFile(t, `type: "kubernetes"
`)
	_, err := DecodeFile(path)
	require.Error(t, err, "missing required name must fail schema validation")
}

func TestDecodeFile_FilterShapeRejected(t *testing.T) {
	// The retired filter vocabulary is not accepted in files.
	path := writePlatformFile(t, `name: "cluster"
type: "kubernetes"
registry: {
	"opmodel.dev/catalogs/opm@v2": {
		version: "2.0.0-alpha.3"
		filter: range: ">=1.0.0-0 <2.0.0-0"
	}
}
`)
	_, err := DecodeFile(path)
	require.Error(t, err, "a filter block must fail schema validation")
}

func TestDecodeFile_MissingVersionRejected(t *testing.T) {
	path := writePlatformFile(t, `name: "cluster"
type: "kubernetes"
registry: {
	"opmodel.dev/catalogs/opm@v2": {
		enable: true
	}
}
`)
	_, err := DecodeFile(path)
	require.Error(t, err, "a subscription without version must fail schema validation")
}

func TestDecodeFile_MajorFreeKeyRejected(t *testing.T) {
	path := writePlatformFile(t, `name: "cluster"
type: "kubernetes"
registry: {
	"opmodel.dev/catalogs/opm": {
		version: "2.0.0-alpha.3"
	}
}
`)
	_, err := DecodeFile(path)
	require.Error(t, err, "a registry key without the @vN suffix must fail schema validation")
}

func TestDecodeCRSpec_RoundTripsWireShape(t *testing.T) {
	// The CR spec is the same wire shape the file uses.
	spec := map[string]any{
		"type": "kubernetes",
		"registry": map[string]any{
			"opmodel.dev/catalogs/opm@v2": map[string]any{
				"enable":  true,
				"version": "2.0.0-alpha.3",
			},
		},
	}

	in, err := DecodeCRSpec(spec, "cluster")
	require.NoError(t, err)

	assert.Equal(t, "cluster", in.Name)
	assert.Equal(t, "kubernetes", in.Type)
	sub := in.Subscriptions["opmodel.dev/catalogs/opm@v2"]
	require.NotNil(t, sub.Enable)
	assert.True(t, *sub.Enable)
	assert.Equal(t, "2.0.0-alpha.3", sub.Version)
}

func TestDecodeCRSpec_LegacyFilterCRTolerated(t *testing.T) {
	// A stored CR from before the scalar-version shape carries filter and no
	// version. Decode must succeed (read tolerance is permanent); the empty
	// version passes through to fail only at synthesis.
	spec := map[string]any{
		"type": "kubernetes",
		"registry": map[string]any{
			"opmodel.dev/catalogs/opm": map[string]any{
				"filter": map[string]any{"range": ">=1.0.0-0 <2.0.0-0"},
			},
		},
	}

	in, err := DecodeCRSpec(spec, "cluster")
	require.NoError(t, err, "legacy filter-shaped CRs must decode")

	sub, ok := in.Subscriptions["opmodel.dev/catalogs/opm"]
	require.True(t, ok)
	assert.Empty(t, sub.Version, "missing version decodes empty and fails at synthesis")
}

func TestDecodeCRSpec_MissingType(t *testing.T) {
	_, err := DecodeCRSpec(map[string]any{}, "cluster")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "spec.type")
}

func TestWrapClusterMaterializeError_LegacyCRHint(t *testing.T) {
	// The kernel's missing-version refusal (what an empty Version from a
	// legacy CR fails with at synthesis) gains the legacy-CR hint; the
	// sentinel stays reachable for errors.Is.
	synthErr := fmt.Errorf("synthesizing platform %q: %w", "cluster", synth.ErrSubscriptionMissingVersion)

	wrapped := WrapClusterMaterializeError(synthErr)
	require.Error(t, wrapped)
	assert.ErrorIs(t, wrapped, synth.ErrSubscriptionMissingVersion)
	assert.Contains(t, wrapped.Error(), "predate the scalar-version subscription shape")

	// Any other error passes through unchanged.
	other := errors.New("boom")
	assert.Same(t, other, WrapClusterMaterializeError(other))
	assert.NoError(t, WrapClusterMaterializeError(nil))
}

func TestWireRoundTrip_FileToInputToCRSpec(t *testing.T) {
	// file → input → wire (write-if-absent doc) must preserve the document.
	path := writePlatformFile(t, config.DefaultPlatformTemplate)
	in, err := DecodeFile(path)
	require.NoError(t, err)

	w := wireFromInput(in)
	assert.Equal(t, in.Type, w.Type)
	assert.Len(t, w.Registry, len(in.Subscriptions))
	assert.Equal(t, "2.0.0-alpha.6", w.Registry["opmodel.dev/catalogs/opm@v2"].Version)
	assert.Equal(t, "1.0.0-alpha.1", w.Registry["opmodel.dev/catalogs/k8s@v1"].Version)
}
