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

	// secretKeyReleaseMetadata is the stringData key for ReleaseMetadata JSON.
	secretKeyReleaseMetadata = "releaseMetadata"

	// secretKeyModuleMetadata is the stringData key for ModuleMetadata JSON.
	secretKeyModuleMetadata = "moduleMetadata"

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
// releaseName is the release name supplied by the user (e.g. "mc").
// Module identity is carried in data.moduleMetadata instead of labels.
//
//nolint:revive // Inventory prefix is intentional for cross-package clarity
func InventoryLabels(releaseName, releaseNamespace, releaseID string) map[string]string {
	return map[string]string{
		"app.kubernetes.io/managed-by":         "open-platform-model",
		"module-release.opmodel.dev/name":      releaseName,
		"module-release.opmodel.dev/namespace": releaseNamespace,
		"module-release.opmodel.dev/uuid":      releaseID,
		"opmodel.dev/component":                "inventory",
	}
}

// MarshalToSecret serializes an InventorySecret to a typed corev1.Secret.
// The Secret uses stringData for writing (Kubernetes will base64-encode it).
// Existing resourceVersion is included to support updates.
func MarshalToSecret(inv *InventorySecret) (*corev1.Secret, error) {
	secretName := SecretName(inv.ReleaseMetadata.ReleaseName, inv.ReleaseMetadata.ReleaseID)

	// Serialize release metadata
	releaseMetaBytes, err := json.Marshal(inv.ReleaseMetadata)
	if err != nil {
		return nil, fmt.Errorf("marshaling release metadata: %w", err)
	}

	// Serialize module metadata
	moduleMetaBytes, err := json.Marshal(inv.ModuleMetadata)
	if err != nil {
		return nil, fmt.Errorf("marshaling module metadata: %w", err)
	}

	// Serialize index
	indexBytes, err := json.Marshal(inv.Index)
	if err != nil {
		return nil, fmt.Errorf("marshaling inventory index: %w", err)
	}

	stringData := map[string]string{
		secretKeyReleaseMetadata: string(releaseMetaBytes),
		secretKeyModuleMetadata:  string(moduleMetaBytes),
		secretKeyIndex:           string(indexBytes),
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
			Namespace: inv.ReleaseMetadata.ReleaseNamespace,
			Labels:    InventoryLabels(inv.ReleaseMetadata.ReleaseName, inv.ReleaseMetadata.ReleaseNamespace, inv.ReleaseMetadata.ReleaseID),
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
// The moduleMetadata key is optional — if absent, ModuleMetadata is left as zero value.
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

	// Parse release metadata (required)
	releaseMetaJSON, ok := values[secretKeyReleaseMetadata]
	if !ok {
		return nil, fmt.Errorf("inventory Secret missing %q key", secretKeyReleaseMetadata)
	}
	var releaseMeta ReleaseMetadata
	if err := json.Unmarshal([]byte(releaseMetaJSON), &releaseMeta); err != nil {
		return nil, fmt.Errorf("parsing release metadata: %w", err)
	}

	// Parse module metadata (optional — zero value if absent)
	var moduleMeta ModuleMetadata
	if moduleMetaJSON, exists := values[secretKeyModuleMetadata]; exists {
		if err := json.Unmarshal([]byte(moduleMetaJSON), &moduleMeta); err != nil {
			return nil, fmt.Errorf("parsing module metadata: %w", err)
		}
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
		ReleaseMetadata: releaseMeta,
		ModuleMetadata:  moduleMeta,
		Index:           index,
		Changes:         changes,
		resourceVersion: secret.ResourceVersion,
	}

	return inv, nil
}
