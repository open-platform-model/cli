package inventory

import (
	"context"
	"encoding/json"
	"fmt"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/open-platform-model/cli/internal/kubernetes"
	"github.com/open-platform-model/cli/internal/output"
	pkgcore "github.com/open-platform-model/cli/pkg/core"
	pkginventory "github.com/open-platform-model/cli/pkg/inventory"
)

// This file is the only surviving reader of the deprecated inventory Secret
// backend. It exists solely to power the one-time Secret→CR migration on apply
// (enhancement 0006 D8/D14); no other command reads Secrets.

const legacySecretKeyRecord = "inventory"

// LegacyInventory is a legacy inventory Secret decoded for migration.
type LegacyInventory struct {
	// InstanceUUID is the record's instance identity UUID.
	InstanceUUID string
	// Inventory is the ported inventory block (entries/revision/digest/count).
	Inventory pkginventory.Inventory
	// SecretName and SecretNamespace locate the Secret to delete after the CR
	// status write succeeds.
	SecretName      string
	SecretNamespace string
}

// LegacySecretName reconstructs the deterministic legacy inventory Secret name.
func LegacySecretName(instanceName, instanceID string) string {
	return fmt.Sprintf("opm.%s.%s", instanceName, instanceID)
}

// legacyRecord mirrors the JSON envelope the Secret backend persisted. Only the
// fields the migration needs are decoded.
type legacyRecord struct {
	InstanceMetadata struct {
		Name      string `json:"name"`
		Namespace string `json:"namespace"`
		UUID      string `json:"uuid"`
	} `json:"instanceMetadata"`
	Inventory pkginventory.Inventory `json:"inventory"`
}

// FindLegacySecretInventory locates a legacy inventory Secret for migration
// using the Secret-era lookup: a direct GET by name, then a UUID-label
// fallback. Returns (nil, nil) when no legacy Secret exists.
func FindLegacySecretInventory(ctx context.Context, client *kubernetes.Client, instanceName, namespace, instanceID string) (*LegacyInventory, error) {
	name := LegacySecretName(instanceName, instanceID)

	secret, err := client.Clientset.CoreV1().Secrets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err == nil {
		return decodeLegacySecret(secret.Name, secret.Namespace, secret.Data[legacySecretKeyRecord])
	}
	if !apierrors.IsNotFound(err) {
		return nil, fmt.Errorf("getting legacy inventory Secret %q: %w", name, err)
	}

	// Fallback: UUID-label lookup (handles renamed / legacy Secrets).
	labelSelector := fmt.Sprintf("%s=%s,%s=%s",
		pkgcore.LabelModuleInstanceUUID, instanceID,
		pkgcore.LabelComponent, "inventory",
	)
	list, err := client.Clientset.CoreV1().Secrets(namespace).List(ctx, metav1.ListOptions{LabelSelector: labelSelector})
	if err != nil {
		return nil, fmt.Errorf("listing legacy inventory Secrets by label: %w", err)
	}
	if len(list.Items) == 0 {
		return nil, nil
	}
	item := list.Items[0]
	return decodeLegacySecret(item.Name, item.Namespace, item.Data[legacySecretKeyRecord])
}

func decodeLegacySecret(name, namespace string, payload []byte) (*LegacyInventory, error) {
	if len(payload) == 0 {
		return nil, fmt.Errorf("legacy inventory Secret %q missing %q key", name, legacySecretKeyRecord)
	}
	var rec legacyRecord
	if err := json.Unmarshal(payload, &rec); err != nil {
		return nil, fmt.Errorf("parsing legacy inventory Secret %q: %w", name, err)
	}
	if rec.Inventory.Entries == nil {
		rec.Inventory.Entries = []pkginventory.InventoryEntry{}
	}
	return &LegacyInventory{
		InstanceUUID:    rec.InstanceMetadata.UUID,
		Inventory:       rec.Inventory,
		SecretName:      name,
		SecretNamespace: namespace,
	}, nil
}

// DeleteLegacySecret removes a migrated legacy Secret. NotFound is success.
func DeleteLegacySecret(ctx context.Context, client *kubernetes.Client, name, namespace string) error {
	err := client.Clientset.CoreV1().Secrets(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("deleting legacy inventory Secret %q: %w", name, err)
	}
	output.Debug("deleted migrated legacy inventory Secret", "name", name, "namespace", namespace)
	return nil
}
