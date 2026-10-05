package apply

import (
	"context"
	"encoding/json"
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
	k8sinventory "github.com/open-platform-model/library/opm/k8s/inventory"
	"github.com/open-platform-model/library/opm/kernel"
	"github.com/open-platform-model/library/opm/module"
)

// clientWithFailingStatusWrite returns a client whose ModuleInstance spec apply
// succeeds but whose status-subresource apply fails, plus a clientset seeded
// with the given legacy Secret. It backs the delete-after-status ordering test.
// onSpecPatch, when non-nil, receives the spec apply's patch payload.
func clientWithFailingStatusWrite(secret *corev1.Secret, onSpecPatch func([]byte)) *kubernetes.Client {
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
		if onSpecPatch != nil {
			onSpecPatch(patch.GetPatch())
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

	var specPatch []byte
	client := clientWithFailingStatusWrite(secret, func(p []byte) { specPatch = p })

	req := Request{
		Result: &workflowrender.Result{
			Instance: module.InstanceMetadata{Name: name, Namespace: namespace, UUID: instID},
			Module:   module.ModuleMetadata{ModulePath: "opmodel.dev/modules/" + name + "@v0", Name: name, Version: "0.1.0"},
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

	// The spec write that preceded the failure carried the canonical module
	// reference: the registry path verbatim and the v-prefixed version the
	// operator resolves without normalising.
	require.NotNil(t, specPatch, "the spec apply must have been issued")
	var applied struct {
		Spec struct {
			Module struct {
				Path    string `json:"path"`
				Version string `json:"version"`
			} `json:"module"`
		} `json:"spec"`
	}
	require.NoError(t, json.Unmarshal(specPatch, &applied))
	require.Equal(t, "opmodel.dev/modules/podinfo@v0", applied.Spec.Module.Path)
	require.Equal(t, "v0.1.0", applied.Spec.Module.Version)

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

// The spec write records the render's skipped demands as the
// skipped-contracts annotation, "<component>=<fqn>" pairs; a render that
// skipped nothing writes no annotation.
func TestWriteInstanceRecord_RecordsSkippedContracts(t *testing.T) {
	annotationsWritten := func(t *testing.T, skipped []kernel.SkippedDemand) map[string]string {
		t.Helper()
		var specPatch []byte
		client := clientWithFailingStatusWrite(&corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "unrelated", Namespace: "default"}},
			func(p []byte) { specPatch = p })
		req := Request{
			Result: &workflowrender.Result{
				Instance: module.InstanceMetadata{Name: "hello", Namespace: "default", UUID: "uuid-1"},
				Module:   module.ModuleMetadata{ModulePath: "example.com/modules/hello@v0", Name: "hello", Version: "0.1.0"},
				Skipped:  skipped,
			},
			K8sClient: client,
			Log:       output.InstanceLogger("skipped-test"),
		}
		// The status write fails by design of the fake; the spec write
		// before it is what this test reads.
		_ = WriteInstanceRecord(context.Background(), req, nil, nil, nil, "sha256:x", req.Log)
		require.NotNil(t, specPatch, "the spec apply must have been issued")
		var applied struct {
			Metadata struct {
				Annotations map[string]string `json:"annotations"`
			} `json:"metadata"`
		}
		require.NoError(t, json.Unmarshal(specPatch, &applied))
		return applied.Metadata.Annotations
	}

	got := annotationsWritten(t, []kernel.SkippedDemand{
		{Component: "db", FQN: "opmodel.dev/catalogs/opm/traits/backup@v1alpha1", Kind: "trait"},
	})
	require.Equal(t, map[string]string{
		inventory.AnnotationSkippedContracts: "db=opmodel.dev/catalogs/opm/traits/backup@v1alpha1",
	}, got)

	require.Empty(t, annotationsWritten(t, nil), "nothing skipped: no annotation, so SSA removes a prior one")
}

// clientCapturingStatusWrite returns a client whose spec and status applies
// both succeed, handing each status-subresource patch to onStatusPatch.
func clientCapturingStatusWrite(onStatusPatch func([]byte)) *kubernetes.Client {
	scheme := runtime.NewScheme()
	fake := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(scheme,
		map[schema.GroupVersionResource]string{inventory.ModuleInstanceGVR: "ModuleInstanceList"})
	fake.PrependReactor("patch", inventory.ResourceModuleInstances, func(action k8stesting.Action) (bool, runtime.Object, error) {
		patch, ok := action.(k8stesting.PatchAction)
		if !ok {
			return false, nil, nil
		}
		if patch.GetSubresource() == "status" {
			onStatusPatch(patch.GetPatch())
		}
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
	return &kubernetes.Client{Dynamic: fake, Clientset: k8sfake.NewClientset()}
}

// retiredInventoryDigest is the value the CLI's own inventory digest (the
// sorted entries' JSON, hashed) gave for the entries of
// TestWriteInstanceRecord_StoresTheLibraryInventoryDigest. The stored digest
// changes once, in the release that first records the library's encoding, on
// purpose (0012:D7:R4); this literal keeps that change visible.
const retiredInventoryDigest = "sha256:ff47fc33e679f4ab80a04244a33d8ae6887140412f0c0709506a03b5354a7ca4"

// The status write records the library's inventory digest of the entries it
// writes, not a digest of the CLI's own.
func TestWriteInstanceRecord_StoresTheLibraryInventoryDigest(t *testing.T) {
	const namespace = "demo"
	entries := []inventory.InventoryEntry{
		{Kind: "Namespace", Name: namespace, Version: "v1"},
		{Group: "apps", Kind: "Deployment", Namespace: namespace, Name: "web", Version: "v1", Component: "web"},
		{Kind: "ConfigMap", Namespace: namespace, Name: "settings", Version: "v1"},
	}

	var statusPatch []byte
	client := clientCapturingStatusWrite(func(p []byte) { statusPatch = p })
	req := Request{
		Result: &workflowrender.Result{
			Instance: module.InstanceMetadata{Name: "web", Namespace: namespace, UUID: "uuid-1"},
			Module:   module.ModuleMetadata{ModulePath: "example.com/modules/web@v0", Name: "web", Version: "0.1.0"},
		},
		K8sClient: client,
		Log:       output.InstanceLogger("digest-test"),
	}
	require.NoError(t, WriteInstanceRecord(context.Background(), req, nil, nil, entries, "sha256:render", req.Log))
	require.NotNil(t, statusPatch, "the status apply must have been issued")

	var written struct {
		Status struct {
			Inventory struct {
				Digest string `json:"digest"`
			} `json:"inventory"`
		} `json:"status"`
	}
	require.NoError(t, json.Unmarshal(statusPatch, &written))

	libEntries := make([]k8sinventory.Entry, 0, len(entries))
	for _, e := range entries {
		libEntries = append(libEntries, k8sinventory.Entry(e))
	}
	require.Equal(t, k8sinventory.Digest(libEntries), written.Status.Inventory.Digest)
	require.NotEqual(t, retiredInventoryDigest, written.Status.Inventory.Digest)
}
