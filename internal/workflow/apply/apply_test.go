package apply

import (
	"testing"

	k8sinventory "github.com/open-platform-model/library/opm/k8s/inventory"

	"github.com/open-platform-model/cli/internal/inventory"
	"github.com/open-platform-model/cli/internal/kubernetes"
	"github.com/open-platform-model/cli/internal/output"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestCurrentInventoryEntries(t *testing.T) {
	resources := []*unstructured.Unstructured{{Object: map[string]any{"apiVersion": "v1", "kind": "ConfigMap", "metadata": map[string]any{"name": "demo", "namespace": "apps"}}}}
	entries := CurrentInventoryEntries(resources)
	require.Len(t, entries, 1)
	assert.Equal(t, "ConfigMap", entries[0].Kind)
	assert.Equal(t, "demo", entries[0].Name)
	assert.Equal(t, "apps", entries[0].Namespace)
}

func TestPreviousEntries_FromCRRecord(t *testing.T) {
	prev := &inventory.Record{Inventory: inventory.Inventory{Entries: []k8sinventory.Entry{{Kind: "Service", Name: "web"}}}}
	entries := previousEntries(prev)
	require.Len(t, entries, 1)
	assert.Equal(t, "Service", entries[0].Kind)
	assert.Equal(t, "web", entries[0].Name)
}

func TestPreviousEntries_NoRecord(t *testing.T) {
	assert.Empty(t, previousEntries(nil))
}

func TestNextRevision(t *testing.T) {
	assert.Equal(t, 1, nextRevision(nil))
	assert.Equal(t, 3, nextRevision(&inventory.Record{Inventory: inventory.Inventory{Revision: 2}}))
}

func TestGuardEmptyRender(t *testing.T) {
	instanceLog := output.InstanceLogger("test")
	err := GuardEmptyRender(0, []k8sinventory.Entry{{Kind: "ConfigMap"}}, false, instanceLog)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "render produced 0 resources")
}

func TestSourceDigest(t *testing.T) {
	// Deterministic and reference-derived.
	a := sourceDigest("opmodel.dev/modules/podinfo@v0", "0.1.0")
	b := sourceDigest("opmodel.dev/modules/podinfo@v0", "0.1.0")
	assert.Equal(t, a, b)
	assert.Contains(t, a, "sha256:")
	// Different reference → different digest.
	assert.NotEqual(t, a, sourceDigest("opmodel.dev/modules/podinfo@v0", "0.2.0"))
	// Empty reference → empty digest.
	assert.Equal(t, "", sourceDigest("", ""))
}

// TestSourceDigestGoldenPeer pins the exact bytes of the digest for a fixed
// canonical reference. The operator's ModuleSourceDigest
// (opm-operator/internal/status) is the byte-identical peer computing
// sha256(path + "@" + version) over the same pair read from the same CR; the
// operator carries the mirror of this golden. A change that moves this value
// breaks cross-actor digest comparability — fix the code, not the golden,
// unless both sides move together.
func TestSourceDigestGoldenPeer(t *testing.T) {
	got := sourceDigest("opmodel.dev/modules/podinfo@v0", "v0.1.4")
	assert.Equal(t, "sha256:abb734bab3f4ff7665b0f5da96352c68f0b539905501916df3b1a1c6a87e910d", got)
}

func TestFormatApplySummary(t *testing.T) {
	summary := FormatApplySummary(&kubernetes.ApplyResult{Applied: 5, Created: 2, Configured: 1, Unchanged: 2})
	assert.Equal(t, "applied 5 resources successfully (2 created, 1 configured, 2 unchanged)", summary)
}

// Ownership refusal is unit-tested at the resolver (inventory.ResolveOwnership)
// and exercised end-to-end with a live CRD in the e2e gate tests; the full
// gate+ownership Execute path needs a seeded CRD/Platform/CR fixture that the
// e2e suite provides.
