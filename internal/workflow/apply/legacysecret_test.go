package apply

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sfake "k8s.io/client-go/kubernetes/fake"
)

// An instance that an old opm release recorded only in an inventory Secret
// (a, b and c) has no ModuleInstance record, and the cli does not read the
// Secret. Its apply is a first install over resources OPM already manages:
// it warns, applies and records what it renders (a and b), deletes nothing,
// so c stays, and makes no request on Secrets, so the Secret stays too.
func TestExecute_InstanceRecordedOnlyInALegacySecretIsAFirstInstall(t *testing.T) {
	withReleasedCLIVersion(t)
	logBuf := captureLog(t)
	cluster := newApplyCluster(liveManagedConfigMap("a"), liveManagedConfigMap("b"), liveManagedConfigMap("c"))
	clientset := cluster.client.Clientset.(*k8sfake.Clientset)
	// The Secret as the old releases wrote it: name opm.<instance>.<uuid>,
	// the record as JSON under the key "inventory".
	require.NoError(t, clientset.Tracker().Add(&corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "opm.demo.uuid-1",
			Namespace: "default",
			Labels: map[string]string{
				"module-instance.opmodel.dev/uuid": "uuid-1",
				"opmodel.dev/component":            "inventory",
			},
		},
		Data: map[string][]byte{"inventory": []byte(`{"instanceMetadata":{"name":"demo","namespace":"default","uuid":"uuid-1"},` +
			`"inventory":{"revision":4,"count":3,"entries":[` +
			`{"group":"","kind":"ConfigMap","namespace":"default","name":"a","v":"v1"},` +
			`{"group":"","kind":"ConfigMap","namespace":"default","name":"b","v":"v1"},` +
			`{"group":"","kind":"ConfigMap","namespace":"default","name":"c","v":"v1"}]}}`)},
	}))

	require.NoError(t, Execute(context.Background(), cluster.request(Options{WarnUnrecorded: true}, "a", "b")))

	for _, a := range clientset.Actions() {
		assert.NotEqual(t, "secrets", a.GetResource().Resource, "no request on Secrets: %s", a.GetVerb())
	}
	for _, a := range cluster.dyn.Actions() {
		assert.NotEqual(t, "secrets", a.GetResource().Resource, "no request on Secrets through the dynamic client: %s", a.GetVerb())
	}
	assert.Equal(t, []string{
		"patch configmaps a",
		"patch configmaps b",
		"patch moduleinstances demo",
		"patch moduleinstances/status demo",
	}, cluster.writes(), "the rendered resources and the record are written; nothing is deleted or pruned")

	entries, written := cluster.writtenInventory(t)
	require.True(t, written, "the record is written")
	assert.ElementsMatch(t, []string{"a", "b"}, entryNames(entries), "the record holds what this apply rendered")
	assert.Equal(t, 1, cluster.writtenRevision(t), "the record starts at revision 1")

	assert.Contains(t, logBuf.String(), "2 of 2 rendered resource(s) already exist and are managed by OPM", "the apply is not silent")

	_, err := clientset.Tracker().Get(corev1.SchemeGroupVersion.WithResource("secrets"), "default", "opm.demo.uuid-1")
	assert.NoError(t, err, "the Secret is still in the cluster")
}
