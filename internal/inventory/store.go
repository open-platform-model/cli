package inventory

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"

	"github.com/open-platform-model/cli/internal/kubernetes"
	"github.com/open-platform-model/cli/internal/output"
	pkgcore "github.com/open-platform-model/cli/pkg/core"
	pkginventory "github.com/open-platform-model/cli/pkg/inventory"
)

// fieldManager is the server-side-apply field manager the CLI owns on the
// ModuleInstance CR. It matches the resource-apply manager so a single actor
// owns both the instance's workloads and its inventory record.
const fieldManager = pkgcore.LabelManagedByValue // "opm-cli"

// GetRecord reads the ModuleInstance CR for an instance by name and namespace.
// A NotFound response is returned as (nil, nil) — the "no inventory" /
// first-apply signal.
func GetRecord(ctx context.Context, client *kubernetes.Client, name, namespace string) (*Record, error) {
	obj, err := client.ResourceClient(ModuleInstanceGVR, namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("getting ModuleInstance %q: %w", name, err)
	}
	return recordFromUnstructured(obj), nil
}

// ListRecords lists ModuleInstance CRs in a namespace (pass "" for a
// cluster-wide list), sorted alphabetically by name.
func ListRecords(ctx context.Context, client *kubernetes.Client, namespace string) ([]*Record, error) {
	list, err := client.ResourceClient(ModuleInstanceGVR, namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("listing ModuleInstances: %w", err)
	}

	records := make([]*Record, 0, len(list.Items))
	for i := range list.Items {
		item := &list.Items[i]
		if item.GetName() == "" {
			output.Warn("skipping ModuleInstance with no name", "namespace", item.GetNamespace())
			continue
		}
		if !interpretableInventory(item) {
			output.Warn("skipping ModuleInstance with malformed status.inventory",
				"name", item.GetName(), "namespace", item.GetNamespace())
			continue
		}
		records = append(records, recordFromUnstructured(item))
	}

	sort.Slice(records, func(i, j int) bool {
		return records[i].Name < records[j].Name
	})
	return records, nil
}

// FindRecordByInstanceUUID resolves an instance by listing ModuleInstance CRs
// in a namespace and matching status.instanceUUID. Returns (nil, nil) when no
// CR carries the UUID.
func FindRecordByInstanceUUID(ctx context.Context, client *kubernetes.Client, namespace, instanceUUID string) (*Record, error) {
	records, err := ListRecords(ctx, client, namespace)
	if err != nil {
		return nil, err
	}
	for _, r := range records {
		if r.InstanceUUID == instanceUUID {
			return r, nil
		}
	}
	return nil, nil
}

// SpecInput is the desired ModuleInstance spec the CLI server-side-applies.
type SpecInput struct {
	Name          string
	Namespace     string
	Owner         string
	ModulePath    string
	ModuleVersion string
	// Values is the single unified values blob the render consumed. Nil omits
	// spec.values entirely.
	Values map[string]any
	// SourceLocal stamps the render-provenance annotation when true; when false
	// the annotation is omitted so SSA field ownership removes any prior value.
	SourceLocal bool
}

// ApplySpec server-side-applies the CLI-owned ModuleInstance spec document
// (owner, module reference, values, managed labels, and the provenance
// annotation). Create-or-update is handled by the apply semantics.
func ApplySpec(ctx context.Context, client *kubernetes.Client, in SpecInput) error {
	obj := &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": APIVersionModuleInstance,
		"kind":       KindModuleInstance,
		"metadata": map[string]any{
			"name":      in.Name,
			"namespace": in.Namespace,
			"labels":    crLabels(in.Name, in.Namespace),
		},
		"spec": map[string]any{
			"owner": in.Owner,
			"module": map[string]any{
				"path":    in.ModulePath,
				"version": in.ModuleVersion,
			},
		},
	}}

	if in.Values != nil {
		values, err := jsonNormalizeMap(in.Values)
		if err != nil {
			return fmt.Errorf("normalizing spec.values for ModuleInstance %q: %w", in.Name, err)
		}
		if err := unstructured.SetNestedMap(obj.Object, values, "spec", "values"); err != nil {
			return fmt.Errorf("setting spec.values for ModuleInstance %q: %w", in.Name, err)
		}
	}

	if in.SourceLocal {
		obj.SetAnnotations(map[string]string{AnnotationSource: SourceLocal})
	}

	if err := ssaApply(ctx, client, obj, in.Name, in.Namespace); err != nil {
		return err
	}
	output.Debug("applied ModuleInstance spec", "name", in.Name, "namespace", in.Namespace, "owner", in.Owner)
	return nil
}

// StatusInput is the CLI-owned status subset written on the status subresource.
// The CLI never writes conditions, observedGeneration, lastAttempted*,
// failureCounters, history, or nextRetryAt (enhancement 0006 D2/D25).
type StatusInput struct {
	Name                    string
	Namespace               string
	Inventory               pkginventory.Inventory
	InstanceUUID            string
	LastAppliedRenderDigest string
	LastAppliedSourceDigest string
	LastAppliedConfigDigest string
	LastAppliedAt           string
}

// ApplyStatus server-side-applies the CLI-owned status subset on the status
// subresource with field manager opm-cli.
func ApplyStatus(ctx context.Context, client *kubernetes.Client, in StatusInput) error {
	status := map[string]any{
		"inventory": inventoryToWire(in.Inventory),
	}
	setIfNotEmpty(status, "instanceUUID", in.InstanceUUID)
	setIfNotEmpty(status, "lastAppliedRenderDigest", in.LastAppliedRenderDigest)
	setIfNotEmpty(status, "lastAppliedSourceDigest", in.LastAppliedSourceDigest)
	setIfNotEmpty(status, "lastAppliedConfigDigest", in.LastAppliedConfigDigest)
	setIfNotEmpty(status, "lastAppliedAt", in.LastAppliedAt)

	obj := &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": APIVersionModuleInstance,
		"kind":       KindModuleInstance,
		"metadata": map[string]any{
			"name":      in.Name,
			"namespace": in.Namespace,
		},
		"status": status,
	}}

	if err := ssaApply(ctx, client, obj, in.Name, in.Namespace, "status"); err != nil {
		return err
	}
	output.Debug("applied ModuleInstance status", "name", in.Name, "revision", in.Inventory.Revision)
	return nil
}

// DeleteCR deletes the ModuleInstance CR. NotFound is treated as success
// (idempotent delete).
func DeleteCR(ctx context.Context, client *kubernetes.Client, name, namespace string) error {
	err := client.ResourceClient(ModuleInstanceGVR, namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("deleting ModuleInstance %q: %w", name, err)
	}
	output.Debug("deleted ModuleInstance CR", "name", name, "namespace", namespace)
	return nil
}

func ssaApply(ctx context.Context, client *kubernetes.Client, obj *unstructured.Unstructured, name, namespace string, subresources ...string) error {
	data, err := json.Marshal(obj.Object)
	if err != nil {
		return fmt.Errorf("marshaling ModuleInstance %q: %w", name, err)
	}
	_, err = client.ResourceClient(ModuleInstanceGVR, namespace).Patch(
		ctx, name, types.ApplyPatchType, data,
		metav1.PatchOptions{FieldManager: fieldManager, Force: output.BoolPtr(true)},
		subresources...,
	)
	if err != nil {
		return fmt.Errorf("applying ModuleInstance %q: %w", name, err)
	}
	return nil
}

func recordFromUnstructured(obj *unstructured.Unstructured) *Record {
	rec := &Record{
		Name:                    obj.GetName(),
		Namespace:               obj.GetNamespace(),
		Owner:                   nestedString(obj.Object, "spec", "owner"),
		ModulePath:              nestedString(obj.Object, "spec", "module", "path"),
		ModuleVersion:           nestedString(obj.Object, "spec", "module", "version"),
		InstanceUUID:            nestedString(obj.Object, "status", "instanceUUID"),
		LastAppliedRenderDigest: nestedString(obj.Object, "status", "lastAppliedRenderDigest"),
		LastAppliedSourceDigest: nestedString(obj.Object, "status", "lastAppliedSourceDigest"),
		LastAppliedConfigDigest: nestedString(obj.Object, "status", "lastAppliedConfigDigest"),
		LastAppliedAt:           nestedString(obj.Object, "status", "lastAppliedAt"),
	}

	//nolint:errcheck // best-effort read; a wrong-typed status.inventory yields empty inventory
	if invMap, ok, _ := unstructured.NestedMap(obj.Object, "status", "inventory"); ok {
		rec.Inventory = inventoryFromWire(invMap)
	} else {
		rec.Inventory = pkginventory.Inventory{Entries: []pkginventory.InventoryEntry{}}
	}

	if obj.GetAnnotations()[AnnotationSource] == SourceLocal {
		rec.SourceLocal = true
	}
	return rec
}

// nestedString reads a string at the given path, best-effort: a missing or
// wrong-typed field yields "".
func nestedString(obj map[string]any, fields ...string) string {
	v, _, _ := unstructured.NestedString(obj, fields...) //nolint:errcheck // best-effort read; wrong-typed field → ""
	return v
}

// interpretableInventory reports whether a CR's status.inventory is either
// absent or a well-formed object. A present-but-malformed status.inventory
// (e.g. a scalar) means the CR cannot be interpreted and should be skipped.
func interpretableInventory(obj *unstructured.Unstructured) bool {
	raw, found, err := unstructured.NestedFieldNoCopy(obj.Object, "status", "inventory")
	if err != nil || !found {
		return true // absent inventory is fine (first apply / no status yet)
	}
	_, ok := raw.(map[string]any)
	return ok
}

func crLabels(name, namespace string) map[string]any {
	return map[string]any{
		pkgcore.LabelManagedBy:               pkgcore.LabelManagedByValue,
		pkgcore.LabelModuleInstanceName:      name,
		pkgcore.LabelModuleInstanceNamespace: namespace,
	}
}

func setIfNotEmpty(m map[string]any, key, value string) {
	if value != "" {
		m[key] = value
	}
}

// jsonNormalizeMap round-trips a map through JSON so every value is an
// unstructured-safe type (string, bool, float64, nil, map[string]any, []any).
// The unstructured converter panics on Go integer types that a CUE decode can
// produce; JSON normalization eliminates them.
func jsonNormalizeMap(in map[string]any) (map[string]any, error) {
	data, err := json.Marshal(in)
	if err != nil {
		return nil, err
	}
	var out map[string]any
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, err
	}
	return out, nil
}
