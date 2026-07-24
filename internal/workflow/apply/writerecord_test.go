package apply

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"

	"github.com/open-platform-model/cli/internal/inventory"
	"github.com/open-platform-model/cli/internal/kubernetes"
	"github.com/open-platform-model/cli/internal/output"
	workflowrender "github.com/open-platform-model/cli/internal/workflow/render"
	pkgmodule "github.com/open-platform-model/cli/pkg/module"
)

// clientWithFailingStatusWrite returns a client whose ModuleInstance spec apply
// succeeds but whose status-subresource apply fails, plus a clientset seeded
// with the given legacy Secret. It backs the delete-after-status ordering test.
func clientWithFailingStatusWrite(secret *corev1.Secret) *kubernetes.Client {
	scheme := runtime.NewScheme()
	fake := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(scheme,
		map[schema.GroupVersionResource]string{inventory.ModuleInstanceGVR: "ModuleInstanceList"})

	fake.PrependReactor("patch", inventory.ResourceModuleInstances, func(action k8stesting.Action) (bool, runtime.Object, error) {
		patch, ok := action.(k8stesting.PatchAction)
		if !ok {
			return false, nil, nil
		}
		if patch.GetSubresource() == "status" {
			return true, nil, errors.New("simulated status-subresource write failure")
		}
		// The spec apply succeeds and hands back a generation.
		return true, &unstructured.Unstructured{Object: map[string]any{
			"apiVersion": inventory.APIVersionModuleInstance,
			"kind":       inventory.KindModuleInstance,
			"metadata": map[string]any{
				"name":       patch.GetName(),
				"namespace":  patch.GetNamespace(),
				"generation": int64(1),
			},
		}}, nil
	})

	cs := k8sfake.NewClientset(secret)
	return &kubernetes.Client{Dynamic: fake, Clientset: cs}
}

// WriteInstanceRecord deletes the ported legacy Secret only after the CR status
// write succeeds. When the status write fails, the Secret must survive — it
// stays authoritative for a clean re-run — and no delete may be issued. This is
// the migration's entire safety property; the fake-client reactor is the only
// way to force a real status write to fail.
func TestWriteInstanceRecord_StatusFailureRetainsLegacySecret(t *testing.T) {
	ctx := context.Background()

	const (
		name      = "podinfo"
		namespace = "default"
		instID    = "uuid-1"
	)
	secretName := inventory.LegacySecretName(name, instID)
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: secretName, Namespace: namespace},
		Data:       map[string][]byte{"inventory": []byte(`{"inventory":{"revision":4,"entries":[]}}`)},
	}

	client := clientWithFailingStatusWrite(secret)

	req := Request{
		Result: &workflowrender.Result{
			Instance: pkgmodule.InstanceMetadata{Name: name, Namespace: namespace, UUID: instID},
			Module:   pkgmodule.ModuleMetadata{ModulePath: "opmodel.dev/modules", Name: name, Version: "0.1.0"},
		},
		K8sClient: client,
		Log:       output.InstanceLogger("migrate-fail-test"),
	}
	legacy := &inventory.LegacyInventory{
		SecretName:      secretName,
		SecretNamespace: namespace,
		Inventory:       inventory.Inventory{Revision: 4},
	}
	currentEntries := []inventory.InventoryEntry{{Kind: "ConfigMap", Name: "cm-a", Namespace: namespace}}

	err := WriteInstanceRecord(ctx, req, nil, legacy, currentEntries, "sha256:deadbeef", req.Log)
	require.Error(t, err, "a failed status write must fail the record write")

	// The Secret must still exist — the delete comes only after the status write.
	_, getErr := client.Clientset.CoreV1().Secrets(namespace).Get(ctx, secretName, metav1.GetOptions{})
	require.NoError(t, getErr, "the legacy Secret must survive a failed status write")

	// And no delete may have been issued against any Secret.
	for _, a := range client.Clientset.(*k8sfake.Clientset).Actions() {
		if a.GetVerb() == "delete" && a.GetResource().Resource == "secrets" {
			t.Fatalf("a delete was issued against the legacy Secret before the status write succeeded: %#v", a)
		}
	}
}
