package inventory

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sfake "k8s.io/client-go/kubernetes/fake"

	"github.com/open-platform-model/cli/internal/kubernetes"
	k8sinventory "github.com/open-platform-model/library/opm/k8s/inventory"
	opmlabels "github.com/open-platform-model/library/opm/k8s/labels"
)

// legacySecret builds a Secret in the deleted Secret-backend envelope shape, for
// testing the one-time migration reader.
func legacySecret(name, namespace, instanceName, instanceID string, byLabel bool) *corev1.Secret {
	payload := `{"instanceMetadata":{"name":"` + instanceName + `","namespace":"` + namespace + `","uuid":"` + instanceID + `"},` +
		`"inventory":{"revision":4,"digest":"sha256:legacy","count":2,"entries":[` +
		`{"group":"","kind":"ConfigMap","namespace":"` + namespace + `","name":"cm-a"},` +
		`{"group":"apps","kind":"Deployment","namespace":"` + namespace + `","name":"web","v":"v1","component":"server"}]}}`
	s := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
		Data:       map[string][]byte{legacySecretKeyRecord: []byte(payload)},
	}
	if byLabel {
		s.Labels = map[string]string{
			opmlabels.ModuleInstanceUUID: instanceID,
			opmlabels.Component:          "inventory",
		}
	}
	return s
}

func clientWithSecrets(secrets ...*corev1.Secret) *kubernetes.Client {
	cs := k8sfake.NewClientset()
	for _, s := range secrets {
		_, err := cs.CoreV1().Secrets(s.Namespace).Create(context.Background(), s, metav1.CreateOptions{})
		if err != nil {
			panic(err)
		}
	}
	return &kubernetes.Client{Clientset: cs}
}

func TestFindLegacySecretInventory_ByDirectName(t *testing.T) {
	ctx := context.Background()
	name := LegacySecretName("podinfo", "uuid-1")
	client := clientWithSecrets(legacySecret(name, "demo", "podinfo", "uuid-1", false))

	legacy, err := FindLegacySecretInventory(ctx, client, "podinfo", "demo", "uuid-1")
	require.NoError(t, err)
	require.NotNil(t, legacy)
	assert.Equal(t, "uuid-1", legacy.InstanceUUID)
	assert.Equal(t, 4, legacy.Inventory.Revision)
	assert.Equal(t, name, legacy.SecretName)
	// The whole block is asserted, so a field the hand-written mapping drops
	// fails here.
	assert.Equal(t, Inventory{
		Revision: 4,
		Digest:   "sha256:legacy",
		Count:    2,
		Entries: []k8sinventory.Entry{
			{Kind: "ConfigMap", Namespace: "demo", Name: "cm-a"},
			{Group: "apps", Kind: "Deployment", Namespace: "demo", Name: "web", Version: "v1", Component: "server"},
		},
	}, legacy.Inventory)
}

func TestFindLegacySecretInventory_ByUUIDLabelFallback(t *testing.T) {
	ctx := context.Background()
	// Secret named differently from the deterministic name — found via UUID label.
	client := clientWithSecrets(legacySecret("renamed-secret", "demo", "podinfo", "uuid-2", true))

	legacy, err := FindLegacySecretInventory(ctx, client, "podinfo", "demo", "uuid-2")
	require.NoError(t, err)
	require.NotNil(t, legacy)
	assert.Equal(t, "renamed-secret", legacy.SecretName)
	assert.Equal(t, 4, legacy.Inventory.Revision)
}

func TestFindLegacySecretInventory_NoneReturnsNil(t *testing.T) {
	ctx := context.Background()
	legacy, err := FindLegacySecretInventory(ctx, clientWithSecrets(), "podinfo", "demo", "uuid-3")
	require.NoError(t, err)
	assert.Nil(t, legacy)
}

func TestDeleteLegacySecret_NotFoundIsSuccess(t *testing.T) {
	ctx := context.Background()
	require.NoError(t, DeleteLegacySecret(ctx, clientWithSecrets(), "missing", "demo"))
}

func TestDeleteLegacySecret_RemovesExisting(t *testing.T) {
	ctx := context.Background()
	name := LegacySecretName("podinfo", "uuid-4")
	client := clientWithSecrets(legacySecret(name, "demo", "podinfo", "uuid-4", false))

	require.NoError(t, DeleteLegacySecret(ctx, client, name, "demo"))

	legacy, err := FindLegacySecretInventory(ctx, client, "podinfo", "demo", "uuid-4")
	require.NoError(t, err)
	assert.Nil(t, legacy, "Secret should be gone after delete")
}
