package inventory

import (
	"encoding/json"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	pkgcore "github.com/opmodel/cli/pkg/core"
)

const (
	secretType = "opmodel.dev/release" //nolint:gosec // not a credential

	secretKeyReleaseMetadata = "releaseMetadata"
	secretKeyModuleMetadata  = "moduleMetadata"
	secretKeyIndex           = "index"
	secretKeyPrefix          = "change-sha1-"
)

func SecretName(releaseName, releaseID string) string {
	return fmt.Sprintf("opm.%s.%s", releaseName, releaseID)
}

//nolint:revive // Inventory prefix is intentional for compatibility and clarity
func InventoryLabels(releaseName, releaseNamespace, releaseID string) map[string]string {
	return map[string]string{
		pkgcore.LabelManagedBy:              pkgcore.LabelManagedByValue,
		pkgcore.LabelModuleReleaseName:      releaseName,
		pkgcore.LabelModuleReleaseNamespace: releaseNamespace,
		pkgcore.LabelModuleReleaseUUID:      releaseID,
		pkgcore.LabelComponent:              "inventory",
	}
}

func MarshalToSecret(inv *InventorySecret) (*corev1.Secret, error) {
	secretName := SecretName(inv.ReleaseMetadata.ReleaseName, inv.ReleaseMetadata.ReleaseID)

	releaseMetaBytes, err := json.Marshal(inv.ReleaseMetadata)
	if err != nil {
		return nil, fmt.Errorf("marshaling release metadata: %w", err)
	}

	moduleMetaBytes, err := json.Marshal(inv.ModuleMetadata)
	if err != nil {
		return nil, fmt.Errorf("marshaling module metadata: %w", err)
	}

	indexBytes, err := json.Marshal(inv.Index)
	if err != nil {
		return nil, fmt.Errorf("marshaling inventory index: %w", err)
	}

	stringData := map[string]string{
		secretKeyReleaseMetadata: string(releaseMetaBytes),
		secretKeyModuleMetadata:  string(moduleMetaBytes),
		secretKeyIndex:           string(indexBytes),
	}

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

	if inv.resourceVersion != "" {
		secret.ResourceVersion = inv.resourceVersion
	}

	return secret, nil
}

func UnmarshalFromSecret(secret *corev1.Secret) (*InventorySecret, error) {
	values := make(map[string]string)

	for k, v := range secret.Data {
		values[k] = string(v)
	}
	for k, v := range secret.StringData {
		values[k] = v
	}

	releaseMetaJSON, ok := values[secretKeyReleaseMetadata]
	if !ok {
		return nil, fmt.Errorf("inventory Secret missing %q key", secretKeyReleaseMetadata)
	}
	var releaseMeta ReleaseMetadata
	if err := json.Unmarshal([]byte(releaseMetaJSON), &releaseMeta); err != nil {
		return nil, fmt.Errorf("parsing release metadata: %w", err)
	}

	var moduleMeta ModuleMetadata
	if moduleMetaJSON, exists := values[secretKeyModuleMetadata]; exists {
		if err := json.Unmarshal([]byte(moduleMetaJSON), &moduleMeta); err != nil {
			return nil, fmt.Errorf("parsing module metadata: %w", err)
		}
	}

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
