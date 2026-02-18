package inventory

import (
	"encoding/json"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// secretType is the Kubernetes Secret type for inventory Secrets.
	// This is not a credential — it is a K8s Secret type identifier string.
	secretType = "opmodel.dev/release" //nolint:gosec // not a credential

	// secretKeyMetadata is the stringData key for InventoryMetadata JSON.
	secretKeyMetadata = "metadata"

	// secretKeyIndex is the stringData key for the ordered change ID list.
	secretKeyIndex = "index"

	// secretKeyPrefix is the prefix for per-change-entry keys.
	secretKeyPrefix = "change-sha1-"
)

// SecretName returns the Kubernetes Secret name for an inventory Secret.
// Format: opm.<releaseName>.<releaseID>
func SecretName(releaseName, releaseID string) string {
	return fmt.Sprintf("opm.%s.%s", releaseName, releaseID)
}

// InventoryLabels returns the labels to apply to an inventory Secret.
//
// These labels enable discovery, filtering, and inventory exclusion from
// workload resource queries.
//
// moduleName is the canonical module name (e.g. "minecraft").
// releaseName is the release name supplied by the user (e.g. "mc").
// Both are stored so that inventory Secrets can be found by either.
//
//nolint:revive // Inventory prefix is intentional for cross-package clarity
func InventoryLabels(moduleName, releaseName, releaseNamespace, releaseID string) map[string]string {
	return map[string]string{
		"app.kubernetes.io/managed-by":    "open-platform-model",
		"module.opmodel.dev/name":         moduleName,
		"module-release.opmodel.dev/name": releaseName,
		"module.opmodel.dev/namespace":    releaseNamespace,
		"module-release.opmodel.dev/uuid": releaseID,
		"opmodel.dev/component":           "inventory",
	}
}

// MarshalToSecret serializes an InventorySecret to a typed corev1.Secret.
// The Secret uses stringData for writing (Kubernetes will base64-encode it).
// Existing resourceVersion is included to support updates.
func MarshalToSecret(inv *InventorySecret) (*corev1.Secret, error) {
	secretName := SecretName(inv.Metadata.ReleaseName, inv.Metadata.ReleaseID)

	// Serialize metadata
	metaBytes, err := json.Marshal(inv.Metadata)
	if err != nil {
		return nil, fmt.Errorf("marshaling inventory metadata: %w", err)
	}

	// Serialize index
	indexBytes, err := json.Marshal(inv.Index)
	if err != nil {
		return nil, fmt.Errorf("marshaling inventory index: %w", err)
	}

	stringData := map[string]string{
		secretKeyMetadata: string(metaBytes),
		secretKeyIndex:    string(indexBytes),
	}

	// Serialize each change entry
	for id, entry := range inv.Changes {
		entryBytes, err := json.Marshal(entry)
		if err != nil {
			return nil, fmt.Errorf("marshaling change entry %q: %w", id, err)
		}
		stringData[id] = string(entryBytes)
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: inv.Metadata.Namespace,
			Labels:    InventoryLabels(inv.Metadata.Name, inv.Metadata.ReleaseName, inv.Metadata.Namespace, inv.Metadata.ReleaseID),
		},
		Type:       secretType,
		StringData: stringData,
	}

	// Preserve resourceVersion for updates (optimistic concurrency)
	if inv.resourceVersion != "" {
		secret.ResourceVersion = inv.resourceVersion
	}

	return secret, nil
}

// UnmarshalFromSecret deserializes a corev1.Secret into an InventorySecret.
// Handles both stringData (write path) and data (base64-encoded, Kubernetes GET response).
// The resourceVersion from the Secret metadata is preserved for optimistic concurrency.
func UnmarshalFromSecret(secret *corev1.Secret) (*InventorySecret, error) {
	// Build a unified string map from both data and stringData.
	// data (base64) is what Kubernetes returns on GET; stringData is our write path.
	values := make(map[string]string)

	for k, v := range secret.Data {
		values[k] = string(v)
	}
	// stringData takes precedence if both are present (shouldn't happen in practice)
	for k, v := range secret.StringData {
		values[k] = v
	}

	// Parse metadata
	metaJSON, ok := values[secretKeyMetadata]
	if !ok {
		return nil, fmt.Errorf("inventory Secret missing %q key", secretKeyMetadata)
	}
	var metadata InventoryMetadata
	if err := json.Unmarshal([]byte(metaJSON), &metadata); err != nil {
		return nil, fmt.Errorf("parsing inventory metadata: %w", err)
	}

	// Parse index
	indexJSON, ok := values[secretKeyIndex]
	if !ok {
		return nil, fmt.Errorf("inventory Secret missing %q key", secretKeyIndex)
	}
	var index []string
	if err := json.Unmarshal([]byte(indexJSON), &index); err != nil {
		return nil, fmt.Errorf("parsing inventory index: %w", err)
	}
	if index == nil {
		index = []string{}
	}

	// Parse change entries
	changes := make(map[string]*ChangeEntry)
	for k, v := range values {
		if !strings.HasPrefix(k, secretKeyPrefix) {
			continue
		}
		var entry ChangeEntry
		if err := json.Unmarshal([]byte(v), &entry); err != nil {
			return nil, fmt.Errorf("parsing change entry %q: %w", k, err)
		}
		changes[k] = &entry
	}

	inv := &InventorySecret{
		Metadata:        metadata,
		Index:           index,
		Changes:         changes,
		resourceVersion: secret.ResourceVersion,
	}

	return inv, nil
}
