package inventory

import (
	"context"
	"fmt"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/opmodel/cli/internal/kubernetes"
	"github.com/opmodel/cli/internal/output"
)

// GetInventory reads the inventory Secret for a release.
//
// Discovery strategy:
//  1. Primary: direct GET by constructed name (opm.<releaseName>.<releaseID>)
//  2. Fallback: list Secrets with label module-release.opmodel.dev/uuid=<releaseID>
//
// Returns (nil, nil) on first-time apply when no inventory exists.
func GetInventory(ctx context.Context, client *kubernetes.Client, releaseName, namespace, releaseID string) (*InventorySecret, error) {
	secretName := SecretName(releaseName, releaseID)

	// Primary: direct GET by name
	secret, err := client.Clientset.CoreV1().Secrets(namespace).Get(ctx, secretName, metav1.GetOptions{})
	if err == nil {
		inv, unmarshalErr := UnmarshalFromSecret(secret)
		if unmarshalErr != nil {
			return nil, fmt.Errorf("parsing inventory Secret %q: %w", secretName, unmarshalErr)
		}
		return inv, nil
	}

	if !apierrors.IsNotFound(err) {
		return nil, fmt.Errorf("getting inventory Secret %q: %w", secretName, err)
	}

	// Fallback: label-based lookup (handles renamed Secrets or legacy inventory)
	output.Debug("inventory Secret not found by name, falling back to label lookup",
		"name", secretName, "releaseID", releaseID)

	labelSelector := fmt.Sprintf("%s=%s,%s=%s",
		kubernetes.LabelReleaseUUID, releaseID,
		kubernetes.LabelComponent, "inventory",
	)
	list, err := client.Clientset.CoreV1().Secrets(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return nil, fmt.Errorf("listing inventory Secrets by label: %w", err)
	}

	if len(list.Items) == 0 {
		output.Debug("no inventory found, treating as first-time apply", "releaseID", releaseID)
		return nil, nil
	}

	// Use the first match (there should only be one)
	inv, err := UnmarshalFromSecret(&list.Items[0])
	if err != nil {
		return nil, fmt.Errorf("parsing inventory Secret from label lookup: %w", err)
	}
	return inv, nil
}

// WriteInventory creates or updates the inventory Secret with full PUT semantics.
//
// If the Secret has no resourceVersion (new inventory), it is created.
// If the Secret has a resourceVersion (from a previous GetInventory), it is updated
// with optimistic concurrency — a concurrent write will cause a conflict error.
//
// moduleName and moduleUUID are the canonical module name and identity UUID.
// They are only used when constructing a new InventorySecret (inv.ReleaseMetadata is zero),
// and are ignored on updates where metadata is preserved from the previous Secret.
func WriteInventory(ctx context.Context, client *kubernetes.Client, inv *InventorySecret, moduleName, moduleUUID string) error {
	// On first write (no resourceVersion), populate ModuleMetadata from caller-supplied values.
	// On updates, ModuleMetadata is already populated from UnmarshalFromSecret — preserve it.
	if inv.ResourceVersion() == "" && inv.ModuleMetadata.Name == "" {
		inv.ModuleMetadata = ModuleMetadata{
			Kind:       "Module",
			APIVersion: "core.opmodel.dev/v1alpha1",
			Name:       moduleName,
			UUID:       moduleUUID,
		}
	}

	secret, err := MarshalToSecret(inv)
	if err != nil {
		return fmt.Errorf("serializing inventory: %w", err)
	}

	if inv.ResourceVersion() == "" {
		// Create: new inventory Secret
		_, err = client.Clientset.CoreV1().Secrets(secret.Namespace).Create(ctx, secret, metav1.CreateOptions{})
		if err != nil {
			return fmt.Errorf("creating inventory Secret %q: %w", secret.Name, err)
		}
		output.Debug("created inventory Secret", "name", secret.Name, "namespace", secret.Namespace)
		return nil
	}

	// Update: replace with optimistic concurrency via resourceVersion
	_, err = client.Clientset.CoreV1().Secrets(secret.Namespace).Update(ctx, secret, metav1.UpdateOptions{})
	if err != nil {
		if apierrors.IsConflict(err) {
			return fmt.Errorf("inventory Secret %q was modified concurrently — retry the apply: %w", secret.Name, err)
		}
		return fmt.Errorf("updating inventory Secret %q: %w", secret.Name, err)
	}

	output.Debug("updated inventory Secret", "name", secret.Name, "namespace", secret.Namespace)
	return nil
}

// FindInventoryByReleaseName finds the inventory Secret for a release using only
// the release name and namespace, without requiring a release ID.
//
// Uses label selector: module-release.opmodel.dev/name=<releaseName> + opmodel.dev/component=inventory
//
// Returns (nil, nil) if no inventory Secret exists for the release.
// Returns an error if multiple inventory Secrets are found (unexpected state).
func FindInventoryByReleaseName(ctx context.Context, client *kubernetes.Client, releaseName, namespace string) (*InventorySecret, error) {
	labelSelector := fmt.Sprintf("%s=%s,%s=%s",
		kubernetes.LabelReleaseName, releaseName,
		kubernetes.LabelComponent, "inventory",
	)
	list, err := client.Clientset.CoreV1().Secrets(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return nil, fmt.Errorf("listing inventory Secrets for release %q: %w", releaseName, err)
	}

	if len(list.Items) == 0 {
		output.Debug("no inventory Secret found for release", "releaseName", releaseName, "namespace", namespace)
		return nil, nil
	}

	// Use the first match; warn if multiple exist (shouldn't happen in practice)
	if len(list.Items) > 1 {
		output.Debug("multiple inventory Secrets found for release, using first",
			"releaseName", releaseName, "namespace", namespace, "count", len(list.Items))
	}

	inv, err := UnmarshalFromSecret(&list.Items[0])
	if err != nil {
		return nil, fmt.Errorf("parsing inventory Secret for release %q: %w", releaseName, err)
	}
	return inv, nil
}

// DeleteInventory deletes the inventory Secret for a release.
// Treats 404 (not found) as success — idempotent.
func DeleteInventory(ctx context.Context, client *kubernetes.Client, name, namespace, releaseID string) error {
	secretName := SecretName(name, releaseID)

	err := client.Clientset.CoreV1().Secrets(namespace).Delete(ctx, secretName, metav1.DeleteOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("deleting inventory Secret %q: %w", secretName, err)
	}

	output.Debug("deleted inventory Secret", "name", secretName, "namespace", namespace)
	return nil
}
