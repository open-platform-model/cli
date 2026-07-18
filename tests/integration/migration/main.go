//go:build ignore

// Integration test for the one-time Secret→CR inventory migration (enhancement
// 0006 C1, D6/D8). Exercises the real migration helpers against a live cluster:
// a legacy inventory Secret is ported to a ModuleInstance CR, a stale entry is
// pruned, the Secret is deleted only after the CR status write, and
// status/list read CRs only (an unmigrated Secret is invisible).
//
// Requires a running kind cluster at context "kind-opm-dev" with the
// ModuleInstance CRD installed (opm operator install --crds-only).
// Run with: go run tests/integration/migration/main.go
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/open-platform-model/cli/internal/inventory"
	"github.com/open-platform-model/cli/internal/kubernetes"
	pkgcore "github.com/open-platform-model/cli/pkg/core"
)

const (
	clusterContext = "kind-opm-dev"
	namespace      = "default"

	instanceName = "opm-migrate-test"
	instanceID   = "c3d4e5f6-3333-4444-5555-ccddeeff0011"
	modulePath   = "opmodel.dev/modules/opm_migrate_test@v0"

	otherName = "opm-migrate-unmigrated"
	otherID   = "d4e5f6a7-4444-5555-6666-ddeeff002233"
)

var configMapGVR = schema.GroupVersionResource{Group: "", Version: "v1", Resource: "configmaps"}

func main() {
	ctx := context.Background()
	fmt.Println("=== OPM Secret→CR Migration Integration Test ===")

	client, err := kubernetes.NewClient(kubernetes.ClientOptions{Context: clusterContext})
	check("creating Kubernetes client", err)
	cleanup(ctx, client)

	// 1. Seed the legacy world: apply cm-a, cm-b, cm-stale and a legacy Secret
	//    tracking all three (as if deployed by an older CLI).
	step(1, "Seed legacy Secret inventory tracking [cm-a, cm-b, cm-stale]")
	seeded := buildResources("cm-a", "cm-b", "cm-stale")
	_, err = kubernetes.Apply(ctx, client, seeded, instanceName, kubernetes.ApplyOptions{})
	check("applying seeded resources", err)
	createLegacySecret(ctx, client, entriesOf(seeded), 4)
	fmt.Println("   OK: legacy Secret written (revision 4)")

	// 2. Migrating apply: current render is [cm-a, cm-b]; cm-stale is pruned.
	step(2, "Migrating apply — cm-stale pruned, CR ported, Secret deleted after status write")
	current := buildResources("cm-a", "cm-b")
	_, err = kubernetes.Apply(ctx, client, current, instanceName, kubernetes.ApplyOptions{})
	check("applying current resources", err)

	legacy, err := inventory.FindLegacySecretInventory(ctx, client, instanceName, namespace, instanceID)
	check("finding legacy Secret", err)
	if legacy == nil {
		failf("expected to find the legacy Secret for migration")
	}

	stale := inventory.ComputeStaleSet(legacy.Inventory.Entries, entriesOf(current))
	if len(stale) != 1 || stale[0].Name != "cm-stale" {
		failf("expected stale set [cm-stale], got %v", stale)
	}
	check("pruning cm-stale", inventory.PruneStaleResources(ctx, client, stale))
	waitForConfigMap(ctx, client, "cm-stale", false)
	fmt.Println("   OK: cm-stale pruned")

	// Write the CR (spec + status) continuing the legacy revision, then delete
	// the Secret only after the status write succeeds.
	check("writing CR spec", inventory.ApplySpec(ctx, client, inventory.SpecInput{
		Name: instanceName, Namespace: namespace, Owner: inventory.OwnerCLI,
		ModulePath: modulePath, ModuleVersion: "0.1.0",
	}))
	check("writing CR status", inventory.ApplyStatus(ctx, client, inventory.StatusInput{
		Name: instanceName, Namespace: namespace, InstanceUUID: instanceID,
		Inventory: inventory.Inventory{
			Revision: legacy.Inventory.Revision + 1,
			Digest:   inventory.ComputeDigest(entriesOf(current)),
			Count:    2,
			Entries:  entriesOf(current),
		},
	}))
	check("deleting migrated Secret", inventory.DeleteLegacySecret(ctx, client, legacy.SecretName, legacy.SecretNamespace))

	// 3. Assert the CR is ported and the Secret is gone.
	step(3, "Assert CR ported (revision continued) and Secret deleted")
	rec, err := inventory.GetRecord(ctx, client, instanceName, namespace)
	check("reading migrated CR", err)
	if rec == nil {
		failf("expected a ModuleInstance CR after migration")
	}
	if rec.Inventory.Revision != 5 {
		failf("expected continued revision 5 (legacy 4 + 1), got %d", rec.Inventory.Revision)
	}
	if rec.Owner != inventory.OwnerCLI {
		failf("expected spec.owner cli, got %q", rec.Owner)
	}
	if rec.InstanceUUID != instanceID {
		failf("expected instanceUUID %q, got %q", instanceID, rec.InstanceUUID)
	}
	if _, getErr := client.Clientset.CoreV1().Secrets(namespace).Get(ctx, inventory.LegacySecretName(instanceName, instanceID), metav1.GetOptions{}); !apierrors.IsNotFound(getErr) {
		failf("expected the legacy Secret to be deleted, got err=%v", getErr)
	}
	fmt.Println("   OK: CR at revision 5, spec.owner cli; legacy Secret deleted")

	// 4. status/list read CRs only — a Secret-only instance is invisible.
	step(4, "Unmigrated Secret is invisible to status/list")
	createLegacySecretFor(ctx, client, otherName, otherID, 1)
	other, err := inventory.GetRecord(ctx, client, otherName, namespace)
	check("reading unmigrated instance", err)
	if other != nil {
		failf("expected GetRecord to not see a Secret-only instance")
	}
	records, err := inventory.ListRecords(ctx, client, namespace)
	check("listing records", err)
	for _, r := range records {
		if r.Name == otherName {
			failf("ListRecords must not surface the Secret-only instance %q", otherName)
		}
	}
	fmt.Println("   OK: Secret-only instance is not found by GetRecord/ListRecords")

	cleanup(ctx, client)
	fmt.Println("\n=== ALL SCENARIOS PASSED ===")
}

func entriesOf(resources []*unstructured.Unstructured) []inventory.InventoryEntry {
	entries := make([]inventory.InventoryEntry, len(resources))
	for i, r := range resources {
		entries[i] = inventory.NewEntryFromResource(r)
	}
	return entries
}

// createLegacySecret writes a Secret in the deleted Secret-backend envelope
// shape for the primary test instance.
func createLegacySecret(ctx context.Context, client *kubernetes.Client, entries []inventory.InventoryEntry, revision int) {
	createLegacySecretEntries(ctx, client, instanceName, instanceID, entries, revision)
}

func createLegacySecretFor(ctx context.Context, client *kubernetes.Client, name, id string, revision int) {
	createLegacySecretEntries(ctx, client, name, id, entriesOf(buildResources("cm-x")), revision)
}

func createLegacySecretEntries(ctx context.Context, client *kubernetes.Client, name, id string, entries []inventory.InventoryEntry, revision int) {
	payload := legacyRecordJSON(name, namespace, id, entries, revision)
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      inventory.LegacySecretName(name, id),
			Namespace: namespace,
			Labels: map[string]string{
				pkgcore.LabelManagedBy:          pkgcore.LabelManagedByValue,
				pkgcore.LabelModuleInstanceName: name,
				pkgcore.LabelModuleInstanceUUID: id,
				pkgcore.LabelComponent:          "inventory",
			},
		},
		Data: map[string][]byte{"inventory": []byte(payload)},
	}
	_, err := client.Clientset.CoreV1().Secrets(namespace).Create(ctx, secret, metav1.CreateOptions{})
	if apierrors.IsAlreadyExists(err) {
		_, err = client.Clientset.CoreV1().Secrets(namespace).Update(ctx, secret, metav1.UpdateOptions{})
	}
	check("creating legacy Secret", err)
}

func legacyRecordJSON(name, ns, id string, entries []inventory.InventoryEntry, revision int) string {
	entriesJSON := ""
	for i, e := range entries {
		if i > 0 {
			entriesJSON += ","
		}
		entriesJSON += fmt.Sprintf(`{"group":%q,"kind":%q,"namespace":%q,"name":%q,"v":%q}`,
			e.Group, e.Kind, e.Namespace, e.Name, e.Version)
	}
	return fmt.Sprintf(
		`{"instanceMetadata":{"name":%q,"namespace":%q,"uuid":%q},"inventory":{"revision":%d,"digest":%q,"count":%d,"entries":[%s]}}`,
		name, ns, id, revision, inventory.ComputeDigest(entries), len(entries), entriesJSON,
	)
}

func buildResources(names ...string) []*unstructured.Unstructured {
	res := make([]*unstructured.Unstructured, len(names))
	for i, name := range names {
		res[i] = &unstructured.Unstructured{Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "ConfigMap",
			"metadata": map[string]interface{}{
				"name":      name,
				"namespace": namespace,
				"labels": map[string]interface{}{
					pkgcore.LabelManagedBy:             pkgcore.LabelManagedByValue,
					"module-instance.opmodel.dev/name": instanceName,
					"module-instance.opmodel.dev/uuid": instanceID,
				},
			},
			"data": map[string]interface{}{"key": fmt.Sprintf("value-%s", name)},
		}}
	}
	return res
}

func cleanup(ctx context.Context, client *kubernetes.Client) {
	for _, n := range []string{"cm-a", "cm-b", "cm-stale", "cm-x"} {
		_ = client.ResourceClient(configMapGVR, namespace).Delete(ctx, n, metav1.DeleteOptions{})
	}
	_ = inventory.DeleteCR(ctx, client, instanceName, namespace)
	for _, entry := range []struct{ name, id string }{{instanceName, instanceID}, {otherName, otherID}} {
		_ = client.Clientset.CoreV1().Secrets(namespace).Delete(ctx, inventory.LegacySecretName(entry.name, entry.id), metav1.DeleteOptions{})
	}
}

func waitForConfigMap(ctx context.Context, client *kubernetes.Client, name string, wantPresent bool) {
	deadline := time.Now().Add(15 * time.Second)
	for {
		_, err := client.ResourceClient(configMapGVR, namespace).Get(ctx, name, metav1.GetOptions{})
		if wantPresent && err == nil {
			return
		}
		if !wantPresent && apierrors.IsNotFound(err) {
			return
		}
		if time.Now().After(deadline) {
			failf("timed out waiting for ConfigMap/%s present=%v", name, wantPresent)
		}
		time.Sleep(250 * time.Millisecond)
	}
}

func step(n int, desc string) { fmt.Printf("\n--- Step %d: %s\n", n, desc) }

func check(label string, err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "FAIL: %s: %v\n", label, err)
		os.Exit(1)
	}
}

func failf(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "FAIL: "+format+"\n", args...)
	os.Exit(1)
}
