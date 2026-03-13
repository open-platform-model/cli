package inventory

import (
	"context"
	"fmt"
	"sort"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/opmodel/cli/internal/kubernetes"
	"github.com/opmodel/cli/internal/output"
	pkgcore "github.com/opmodel/cli/pkg/core"
)

// ListInventories discovers all inventory Secrets in the given namespace.
// Pass namespace="" to list across all namespaces (K8s convention).
//
// Discovery uses the label selector:
//
//	opmodel.dev/component = inventory
//
// The selector intentionally omits app.kubernetes.io/managed-by so that
// inventory Secrets created with any OPM actor value (opm-cli, opm-controller,
// or legacy open-platform-model) are discovered during the transition to
// runtime-owned labels.
//
// Results are sorted alphabetically by ReleaseMetadata.ReleaseName.
// Corrupt Secrets that fail to unmarshal are logged and skipped.
func ListInventories(ctx context.Context, client *kubernetes.Client, namespace string) ([]*ReleaseInventoryRecord, error) {
	labelSelector := fmt.Sprintf("%s=%s",
		pkgcore.LabelComponent, "inventory",
	)

	list, err := client.Clientset.CoreV1().Secrets(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return nil, fmt.Errorf("listing inventory Secrets: %w", err)
	}

	var inventories []*ReleaseInventoryRecord
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
