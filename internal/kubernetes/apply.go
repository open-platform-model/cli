package kubernetes

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"

	"github.com/opmodel/cli/internal/build"
	"github.com/opmodel/cli/internal/output"
)

// ApplyOptions configures an apply operation.
type ApplyOptions struct {
	// DryRun performs a server-side dry run without persisting changes.
	DryRun bool

	// Wait waits for resources to be ready after apply.
	Wait bool

	// Timeout is the maximum time to wait for resources.
	Timeout time.Duration
}

// applyResult contains the outcome of an apply operation.
type applyResult struct {
	// Applied is the number of resources successfully applied.
	Applied int

	// Errors contains per-resource errors (non-fatal).
	Errors []resourceError
}

// resourceError captures an error for a specific resource.
type resourceError struct {
	// Resource identifies the resource.
	Kind      string
	Name      string
	Namespace string

	// Err is the error.
	Err error
}

func (e *resourceError) Error() string {
	if e.Namespace != "" {
		return fmt.Sprintf("%s/%s in %s: %v", e.Kind, e.Name, e.Namespace, e.Err)
	}
	return fmt.Sprintf("%s/%s: %v", e.Kind, e.Name, e.Err)
}

// Apply performs server-side apply for a set of rendered resources.
// Resources are assumed to be already ordered by weight (from RenderResult).
func Apply(ctx context.Context, client *Client, resources []*build.Resource, meta build.ModuleMetadata, opts ApplyOptions) (*applyResult, error) {
	result := &applyResult{}
	modLog := output.ModuleLogger(meta.Name)

	for _, res := range resources {
		// Inject OPM labels
		injectLabels(res, meta)

		// Apply the resource
		status, err := applyResource(ctx, client, res.Object, opts)
		if err != nil {
			modLog.Warn(fmt.Sprintf("applying %s/%s: %v", res.Kind(), res.Name(), err))
			result.Errors = append(result.Errors, resourceError{
				Kind:      res.Kind(),
				Name:      res.Name(),
				Namespace: res.Namespace(),
				Err:       err,
			})
			continue
		}

		result.Applied++
		modLog.Info(output.FormatResourceLine(res.Kind(), res.Namespace(), res.Name(), status))
	}

	return result, nil
}

// injectLabels adds OPM labels to a resource if not already present.
func injectLabels(res *build.Resource, meta build.ModuleMetadata) {
	labels := res.Object.GetLabels()
	if labels == nil {
		labels = make(map[string]string)
	}

	// Always set managed-by and module labels
	labels[LabelManagedBy] = labelManagedByValue
	labels[LabelModuleName] = meta.Name
	labels[labelModuleNamespace] = meta.Namespace
	if meta.Version != "" {
		labels[labelModuleVersion] = meta.Version
	}

	// Set component label from resource metadata
	if res.Component != "" {
		labels[LabelComponentName] = res.Component
	}

	// Set identity labels if available
	if meta.Identity != "" {
		labels[labelModuleID] = meta.Identity
	}
	if meta.ReleaseIdentity != "" {
		labels[labelReleaseID] = meta.ReleaseIdentity
	}

	res.Object.SetLabels(labels)
}

// applyResource performs server-side apply for a single resource.
// Returns the status of the operation (created, configured, or unchanged).
func applyResource(ctx context.Context, client *Client, obj *unstructured.Unstructured, opts ApplyOptions) (string, error) {
	gvr := gvrFromObject(obj)
	ns := obj.GetNamespace()

	// Check if resource already exists to determine status after apply.
	var existingVersion string
	if ns != "" {
		existing, err := client.Dynamic.Resource(gvr).Namespace(ns).Get(ctx, obj.GetName(), metav1.GetOptions{})
		if err == nil {
			existingVersion = existing.GetResourceVersion()
		}
	} else {
		existing, err := client.Dynamic.Resource(gvr).Get(ctx, obj.GetName(), metav1.GetOptions{})
		if err == nil {
			existingVersion = existing.GetResourceVersion()
		}
	}
	// If GET fails (NotFound or other), existingVersion stays empty -> "created"

	data, err := json.Marshal(obj)
	if err != nil {
		return "", fmt.Errorf("marshaling resource: %w", err)
	}

	patchOpts := metav1.PatchOptions{
		FieldManager: fieldManagerName,
		Force:        boolPtr(true),
	}

	if opts.DryRun {
		patchOpts.DryRun = []string{metav1.DryRunAll}
	}

	var result *unstructured.Unstructured
	var patchErr error
	if ns != "" {
		result, patchErr = client.Dynamic.Resource(gvr).Namespace(ns).Patch(
			ctx, obj.GetName(), types.ApplyPatchType, data, patchOpts,
		)
	} else {
		result, patchErr = client.Dynamic.Resource(gvr).Patch(
			ctx, obj.GetName(), types.ApplyPatchType, data, patchOpts,
		)
	}

	if patchErr != nil {
		return "", patchErr
	}

	// Determine status from before/after comparison.
	if existingVersion == "" {
		return output.StatusCreated, nil
	}
	if result != nil && result.GetResourceVersion() == existingVersion {
		return output.StatusUnchanged, nil
	}
	return output.StatusConfigured, nil
}

// gvrFromObject derives GroupVersionResource from an unstructured object.
func gvrFromObject(obj *unstructured.Unstructured) schema.GroupVersionResource {
	gvk := obj.GroupVersionKind()
	return schema.GroupVersionResource{
		Group:    gvk.Group,
		Version:  gvk.Version,
		Resource: kindToResource(gvk.Kind),
	}
}

// knownKindResources maps Kind to its plural resource name for well-known types.
// This avoids incorrect heuristic pluralization (e.g., Endpoints -> endpointses).
var knownKindResources = map[string]string{
	"Namespace":                        "namespaces",
	"ServiceAccount":                   "serviceaccounts",
	"Secret":                           "secrets",
	"ConfigMap":                        "configmaps",
	"PersistentVolume":                 "persistentvolumes",
	"PersistentVolumeClaim":            "persistentvolumeclaims",
	"Service":                          "services",
	"Endpoints":                        "endpoints",
	"EndpointSlice":                    "endpointslices",
	"ClusterRole":                      "clusterroles",
	"ClusterRoleBinding":               "clusterrolebindings",
	"Role":                             "roles",
	"RoleBinding":                      "rolebindings",
	"StorageClass":                     "storageclasses",
	"Deployment":                       "deployments",
	"StatefulSet":                      "statefulsets",
	"DaemonSet":                        "daemonsets",
	"ReplicaSet":                       "replicasets",
	"Job":                              "jobs",
	"CronJob":                          "cronjobs",
	"Ingress":                          "ingresses",
	"IngressClass":                     "ingressclasses",
	"NetworkPolicy":                    "networkpolicies",
	"HorizontalPodAutoscaler":          "horizontalpodautoscalers",
	"VerticalPodAutoscaler":            "verticalpodautoscalers",
	"PodDisruptionBudget":              "poddisruptionbudgets",
	"ValidatingWebhookConfiguration":   "validatingwebhookconfigurations",
	"MutatingWebhookConfiguration":     "mutatingwebhookconfigurations",
	"CustomResourceDefinition":         "customresourcedefinitions",
	"ResourceQuota":                    "resourcequotas",
	"LimitRange":                       "limitranges",
	"Pod":                              "pods",
	"Node":                             "nodes",
	"Event":                            "events",
	"PriorityClass":                    "priorityclasses",
	"ValidatingAdmissionPolicy":        "validatingadmissionpolicies",
	"ValidatingAdmissionPolicyBinding": "validatingadmissionpolicybindings",
}

// kindToResource converts a Kind to its plural resource name.
// Uses a known lookup table for common types, falls back to heuristic.
func kindToResource(kind string) string {
	if resource, ok := knownKindResources[kind]; ok {
		return resource
	}
	return heuristicPluralize(kind)
}

// heuristicPluralize applies simple English pluralization rules.
func heuristicPluralize(kind string) string {
	lower := strings.ToLower(kind)
	switch {
	case strings.HasSuffix(lower, "ss") || strings.HasSuffix(lower, "sh") || strings.HasSuffix(lower, "ch") || strings.HasSuffix(lower, "x"):
		return lower + "es"
	case strings.HasSuffix(lower, "s"):
		// Already plural (e.g., Endpoints)
		return lower
	case strings.HasSuffix(lower, "y") && !isVowel(lower[len(lower)-2]):
		return lower[:len(lower)-1] + "ies"
	default:
		return lower + "s"
	}
}

func isVowel(b byte) bool {
	return b == 'a' || b == 'e' || b == 'i' || b == 'o' || b == 'u'
}

func boolPtr(b bool) *bool {
	return &b
}
