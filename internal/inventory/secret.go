package inventory

import (
	"encoding/json"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	pkgcore "github.com/open-platform-model/cli/pkg/core"
)

const (
	secretType      = "opmodel.dev/instance" //nolint:gosec // not a credential
	secretKeyRecord = "inventory"
)

func SecretName(instanceName, instanceID string) string {
	return fmt.Sprintf("opm.%s.%s", instanceName, instanceID)
}

//nolint:revive // Inventory prefix matches existing internal call sites.
func InventoryLabels(instanceName, instanceNamespace, instanceID string) map[string]string {
	return map[string]string{
		pkgcore.LabelManagedBy:               pkgcore.LabelManagedByValue,
		pkgcore.LabelModuleInstanceName:      instanceName,
		pkgcore.LabelModuleInstanceNamespace: instanceNamespace,
		pkgcore.LabelModuleInstanceUUID:      instanceID,
		pkgcore.LabelComponent:               "inventory",
	}
}

func MarshalToSecret(record *InstanceInventoryRecord) (*corev1.Secret, error) {
	secretName := SecretName(record.InstanceMetadata.InstanceName, record.InstanceMetadata.InstanceID)
	payload, err := json.Marshal(record)
	if err != nil {
		return nil, fmt.Errorf("marshaling instance inventory record: %w", err)
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: record.InstanceMetadata.InstanceNamespace,
			Labels:    InventoryLabels(record.InstanceMetadata.InstanceName, record.InstanceMetadata.InstanceNamespace, record.InstanceMetadata.InstanceID),
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

func UnmarshalFromSecret(secret *corev1.Secret) (*InstanceInventoryRecord, error) {
	payload, ok := secret.Data[secretKeyRecord]
	if !ok {
		if payloadStr, ok := secret.StringData[secretKeyRecord]; ok {
			payload = []byte(payloadStr)
		} else {
			return nil, fmt.Errorf("inventory Secret missing %q key", secretKeyRecord)
		}
	}

	var record InstanceInventoryRecord
	if err := json.Unmarshal(payload, &record); err != nil {
		return nil, fmt.Errorf("parsing instance inventory record: %w", err)
	}
	// Fail loud on a zero-value instance metadata: a pre-migration Secret carrying
	// the old `releaseMetadata` JSON key (enhancement 0002 D8/D9) unmarshals cleanly
	// into an empty InstanceMetadata, which would otherwise surface as a phantom
	// blank-name record in `instance list` or poison a `GetInventory` read.
	if record.InstanceMetadata.InstanceName == "" {
		return nil, fmt.Errorf("inventory Secret %q has empty instance metadata (possibly a pre-migration record)", secret.Name)
	}
	if record.Inventory.Entries == nil {
		record.Inventory.Entries = []InventoryEntry{}
	}
	record.resourceVersion = secret.ResourceVersion
	return &record, nil
}
