package inventory

import (
	"context"
	"fmt"
	"sort"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/opmodel/cli/internal/core"
	"github.com/opmodel/cli/internal/kubernetes"
	"github.com/opmodel/cli/internal/output"
)

// ListInventories discovers all inventory Secrets in the given namespace.
// Pass namespace="" to list across all namespaces (K8s convention).
//
// Discovery uses label selectors:
//
//	app.kubernetes.io/managed-by = open-platform-model
//	opmodel.dev/component = inventory
//
// Results are sorted alphabetically by ReleaseMetadata.ReleaseName.
// Corrupt Secrets that fail to unmarshal are logged and skipped.
func ListInventories(ctx context.Context, client *kubernetes.Client, namespace string) ([]*InventorySecret, error) {
	labelSelector := fmt.Sprintf("%s=%s,%s=%s",
		core.LabelManagedBy, core.LabelManagedByValue,
		core.LabelComponent, "inventory",
	)

	list, err := client.Clientset.CoreV1().Secrets(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return nil, fmt.Errorf("listing inventory Secrets: %w", err)
	}

	var inventories []*InventorySecret
	for i := range list.Items {
		inv, unmarshalErr := UnmarshalFromSecret(&list.Items[i])
		if unmarshalErr != nil {
			output.Warn("skipping corrupt inventory Secret",
				"name", list.Items[i].Name,
				"namespace", list.Items[i].Namespace,
				"error", unmarshalErr,
			)
			continue
		}
		inventories = append(inventories, inv)
	}

	sort.Slice(inventories, func(i, j int) bool {
		return inventories[i].ReleaseMetadata.ReleaseName < inventories[j].ReleaseMetadata.ReleaseName
	})

	return inventories, nil
}
