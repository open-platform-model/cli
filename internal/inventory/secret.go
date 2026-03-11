package inventory

import (
	"encoding/json"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	pkgcore "github.com/opmodel/cli/pkg/core"
)

const (
	secretType      = "opmodel.dev/release" //nolint:gosec // not a credential
	secretKeyRecord = "inventory"
)

func SecretName(releaseName, releaseID string) string {
	return fmt.Sprintf("opm.%s.%s", releaseName, releaseID)
}

//nolint:revive // Inventory prefix matches existing internal call sites.
func InventoryLabels(releaseName, releaseNamespace, releaseID string) map[string]string {
	return map[string]string{
		pkgcore.LabelManagedBy:              pkgcore.LabelManagedByValue,
		pkgcore.LabelModuleReleaseName:      releaseName,
		pkgcore.LabelModuleReleaseNamespace: releaseNamespace,
		pkgcore.LabelModuleReleaseUUID:      releaseID,
		pkgcore.LabelComponent:              "inventory",
	}
}

func MarshalToSecret(record *ReleaseInventoryRecord) (*corev1.Secret, error) {
	secretName := SecretName(record.ReleaseMetadata.ReleaseName, record.ReleaseMetadata.ReleaseID)
	payload, err := json.Marshal(record)
	if err != nil {
		return nil, fmt.Errorf("marshaling release inventory record: %w", err)
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: record.ReleaseMetadata.ReleaseNamespace,
			Labels:    InventoryLabels(record.ReleaseMetadata.ReleaseName, record.ReleaseMetadata.ReleaseNamespace, record.ReleaseMetadata.ReleaseID),
		},
		Type: secretType,
		Data: map[string][]byte{
			secretKeyRecord: payload,
		},
	}

	if record.resourceVersion != "" {
		secret.ResourceVersion = record.resourceVersion
	}

	return secret, nil
}

func UnmarshalFromSecret(secret *corev1.Secret) (*ReleaseInventoryRecord, error) {
	payload, ok := secret.Data[secretKeyRecord]
	if !ok {
		if payloadStr, ok := secret.StringData[secretKeyRecord]; ok {
			payload = []byte(payloadStr)
		} else {
			return nil, fmt.Errorf("inventory Secret missing %q key", secretKeyRecord)
		}
	}

	var record ReleaseInventoryRecord
	if err := json.Unmarshal(payload, &record); err != nil {
		return nil, fmt.Errorf("parsing release inventory record: %w", err)
	}
	if record.Inventory.Entries == nil {
		record.Inventory.Entries = []InventoryEntry{}
	}
	record.resourceVersion = secret.ResourceVersion
	return &record, nil
}
