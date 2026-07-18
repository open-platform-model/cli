package platform

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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

	opm, ok := in.Subscriptions["opmodel.dev/catalogs/opm"]
	require.True(t, ok)
	assert.Nil(t, opm.Enable, "omitted enable defers to the schema default")
	require.NotNil(t, opm.Filter)
	assert.Equal(t, ">=1.0.0-0 <2.0.0-0", opm.Filter.Range)

	k8s, ok := in.Subscriptions["opmodel.dev/catalogs/kubernetes"]
	require.True(t, ok)
	require.NotNil(t, k8s.Filter)
	assert.Equal(t, ">=1.1.0-0 <2.0.0-0", k8s.Filter.Range)
}

func TestDecodeFile_ExplicitEnableAndLists(t *testing.T) {
	path := writePlatformFile(t, `name: "cluster"
type: "kubernetes"
registry: {
	"opmodel.dev/catalogs/opm": {
		enable: false
		filter: {
			allow: ["1.2.3"]
			deny: ["1.2.4"]
		}
	}
}
`)

	in, err := DecodeFile(path)
	require.NoError(t, err)

	sub := in.Subscriptions["opmodel.dev/catalogs/opm"]
	require.NotNil(t, sub.Enable)
	assert.False(t, *sub.Enable)
	require.NotNil(t, sub.Filter)
	assert.Equal(t, []string{"1.2.3"}, sub.Filter.Allow)
	assert.Equal(t, []string{"1.2.4"}, sub.Filter.Deny)
	assert.Empty(t, sub.Filter.Range)
}

func TestDecodeFile_InvalidFileRejected(t *testing.T) {
	path := writePlatformFile(t, `type: "kubernetes"
`)
	_, err := DecodeFile(path)
	require.Error(t, err, "missing required name must fail schema validation")
}

func TestDecodeCRSpec_RoundTripsWireShape(t *testing.T) {
	// The CR spec is the same wire shape the file uses.
	spec := map[string]any{
		"type": "kubernetes",
		"registry": map[string]any{
			"opmodel.dev/catalogs/opm": map[string]any{
				"enable": true,
				"filter": map[string]any{
					"range": ">=1.0.0-0 <2.0.0-0",
				},
			},
		},
	}

	in, err := DecodeCRSpec(spec, "cluster")
	require.NoError(t, err)

	assert.Equal(t, "cluster", in.Name)
	assert.Equal(t, "kubernetes", in.Type)
	sub := in.Subscriptions["opmodel.dev/catalogs/opm"]
	require.NotNil(t, sub.Enable)
	assert.True(t, *sub.Enable)
	require.NotNil(t, sub.Filter)
	assert.Equal(t, ">=1.0.0-0 <2.0.0-0", sub.Filter.Range)
}

func TestDecodeCRSpec_MissingType(t *testing.T) {
	_, err := DecodeCRSpec(map[string]any{}, "cluster")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "spec.type")
}

func TestWireRoundTrip_FileToInputToCRSpec(t *testing.T) {
	// file → input → wire (write-if-absent doc) must preserve the document.
	path := writePlatformFile(t, config.DefaultPlatformTemplate)
	in, err := DecodeFile(path)
	require.NoError(t, err)

	w := wireFromInput(in)
	assert.Equal(t, in.Type, w.Type)
	assert.Len(t, w.Registry, len(in.Subscriptions))
	assert.Equal(t, ">=1.0.0-0 <2.0.0-0", w.Registry["opmodel.dev/catalogs/opm"].Filter.Range)
}
