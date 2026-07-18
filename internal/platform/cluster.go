package platform

import (
	"context"
	"encoding/json"
	"fmt"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic"

	"github.com/open-platform-model/library/opm/helper/synth"

	"github.com/open-platform-model/cli/internal/inventory"
	"github.com/open-platform-model/cli/internal/output"
	pkgcore "github.com/open-platform-model/cli/pkg/core"
)

// ClusterSpecGetterFor returns a ClusterSpecGetter that reads the singleton
// cluster Platform CR via the dynamic client. NotFound and Forbidden are
// reported as warn-fallback conditions, not errors (D21).
func ClusterSpecGetterFor(dyn dynamic.Interface) ClusterSpecGetter {
	return func(ctx context.Context) (map[string]any, string, string, error) {
		plat, err := dyn.Resource(inventory.PlatformGVR).Get(ctx, inventory.PlatformSingletonName, metav1.GetOptions{})
		if err != nil {
			switch {
			case apierrors.IsNotFound(err):
				return nil, "", "no Platform CR in the cluster", nil
			case apierrors.IsForbidden(err):
				return nil, "", "reading the Platform CR was denied by RBAC", nil
			default:
				return nil, "", "", err
			}
		}
		spec, found, err := unstructured.NestedMap(plat.Object, "spec")
		if err != nil || !found {
			return nil, "", "", fmt.Errorf("cluster Platform %q has no readable spec", plat.GetName())
		}
		return spec, plat.GetName(), "", nil
	}
}

// EnsureClusterPlatform seeds the singleton cluster Platform from the
// resolved local platform spec, write-if-absent (D12/D22): a plain create
// with field manager opm-cli, treating AlreadyExists as success-noop.
// Never SSA, never update — an existing Platform is never overwritten.
// A Forbidden create degrades to a warning (D17: the render already
// succeeded against the local platform).
func EnsureClusterPlatform(ctx context.Context, dyn dynamic.Interface, in synth.PlatformInput) error {
	w := wireFromInput(in)

	// The CR keeps the name in metadata; the singleton's name is fixed.
	w.Name = ""
	specRaw, err := json.Marshal(w)
	if err != nil {
		return fmt.Errorf("encoding Platform spec: %w", err)
	}
	var spec map[string]any
	if err := json.Unmarshal(specRaw, &spec); err != nil {
		return fmt.Errorf("encoding Platform spec: %w", err)
	}

	doc := &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": inventory.GroupOpmodel + "/" + inventory.VersionV1Alpha1,
		"kind":       inventory.KindPlatform,
		"metadata": map[string]any{
			"name": inventory.PlatformSingletonName,
		},
		"spec": spec,
	}}

	_, err = dyn.Resource(inventory.PlatformGVR).Create(ctx, doc, metav1.CreateOptions{
		FieldManager: pkgcore.LabelManagedByValue, // "opm-cli"
	})
	switch {
	case err == nil:
		output.Info("seeded cluster Platform from the local default platform (write-if-absent)")
		return nil
	case apierrors.IsAlreadyExists(err):
		// The API server's name-uniqueness check is the synchronization
		// primitive (D22): a concurrent create won the race — success-noop.
		output.Debug("cluster Platform already exists; write-if-absent is a no-op")
		return nil
	case apierrors.IsForbidden(err):
		output.Warn("could not seed the cluster Platform (create denied by RBAC); continuing against the local platform")
		return nil
	default:
		return fmt.Errorf("creating cluster Platform: %w", err)
	}
}
