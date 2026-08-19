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
	outcome, err := createClusterPlatform(ctx, dyn, in)
	if err != nil {
		return err
	}
	switch outcome {
	case platformCreated:
		output.Info("seeded cluster Platform from the local default platform (write-if-absent)")
	case platformAlreadyPresent:
		output.Debug("cluster Platform already exists; write-if-absent is a no-op")
	case platformWriteForbidden:
		output.Warn("could not seed the cluster Platform (create denied by RBAC); continuing against the local platform")
	case platformWriteUnset:
		// Unreachable: the zero value is only ever returned beside an error.
	}
	return nil
}

// platformWriteOutcome is what the write-if-absent create actually did, so
// each caller narrates its own provenance instead of inheriting another
// caller's message.
type platformWriteOutcome int

const (
	// platformWriteUnset is the zero value, returned only beside an error.
	platformWriteUnset platformWriteOutcome = iota
	platformCreated
	platformAlreadyPresent
	platformWriteForbidden
)

// createClusterPlatform performs the write-if-absent create itself and
// reports the outcome without printing. The contract lives here and is
// identical for every caller: a plain create with field manager opm-cli,
// never SSA and never update, AlreadyExists as success-noop, Forbidden as a
// non-fatal outcome the caller reports in its own words.
func createClusterPlatform(ctx context.Context, dyn dynamic.Interface, in synth.PlatformInput) (platformWriteOutcome, error) {
	w := wireFromInput(in)

	// The CR keeps the name in metadata; the singleton's name is fixed.
	w.Name = ""
	specRaw, err := json.Marshal(w)
	if err != nil {
		return platformWriteUnset, fmt.Errorf("encoding Platform spec: %w", err)
	}
	var spec map[string]any
	if err := json.Unmarshal(specRaw, &spec); err != nil {
		return platformWriteUnset, fmt.Errorf("encoding Platform spec: %w", err)
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
		return platformCreated, nil
	case apierrors.IsAlreadyExists(err):
		// The API server's name-uniqueness check is the synchronization
		// primitive (D22): a concurrent create won the race — success-noop.
		return platformAlreadyPresent, nil
	case apierrors.IsForbidden(err):
		return platformWriteForbidden, nil
	default:
		return platformWriteUnset, fmt.Errorf("creating cluster Platform: %w", err)
	}
}

// defaultPlatformType is the informational discriminator a seeded Platform
// carries. It mirrors config.DefaultPlatformTemplate; the CRD requires the
// field to be non-empty and nothing matches on its value.
const defaultPlatformType = "kubernetes"

// EnsureClusterPlatformForCatalog seeds the singleton cluster Platform with a
// single subscription to catalogPath at version, under exactly the write
// contract of EnsureClusterPlatform: plain create, never SSA, never update,
// an existing Platform left untouched.
//
// It differs from EnsureClusterPlatform only in provenance. The spec was
// resolved from the registry rather than read from the local default
// platform file, so the reporting names the catalog coordinate that was
// pinned and never claims a file it did not read.
func EnsureClusterPlatformForCatalog(ctx context.Context, dyn dynamic.Interface, catalogPath, version string) error {
	in := synth.PlatformInput{
		Name: inventory.PlatformSingletonName,
		Type: defaultPlatformType,
		Subscriptions: map[string]synth.SubscriptionSpec{
			catalogPath: {Version: version},
		},
	}

	outcome, err := createClusterPlatform(ctx, dyn, in)
	if err != nil {
		return err
	}
	coordinate := catalogPath + " " + version
	switch outcome {
	case platformCreated:
		output.Info(fmt.Sprintf("seeded cluster Platform subscribed to %s", coordinate))
	case platformAlreadyPresent:
		output.Info("cluster Platform already exists; left untouched")
	case platformWriteForbidden:
		output.Warn("could not create the cluster Platform (create denied by RBAC); the operator is installed but has no Platform to reconcile")
	case platformWriteUnset:
		// Unreachable: the zero value is only ever returned beside an error.
	}
	return nil
}
