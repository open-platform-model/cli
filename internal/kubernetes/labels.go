// Package kubernetes provides Kubernetes client operations for the OPM CLI.
package kubernetes

// Standard labels used by OPM to identify and manage resources.
const (
	// LabelManagedBy indicates the tool managing this resource.
	LabelManagedBy = "app.kubernetes.io/managed-by"

	// LabelManagedByValue is the value for LabelManagedBy for OPM-managed resources.
	LabelManagedByValue = "open-platform-model"

	// LabelModuleName identifies the OPM module that created this resource.
	LabelModuleName = "module.opmodel.dev/name"

	// LabelModuleNamespace identifies the namespace the module is deployed to.
	LabelModuleNamespace = "module.opmodel.dev/namespace"

	// LabelModuleVersion identifies the version of the module.
	LabelModuleVersion = "module.opmodel.dev/version"

	// LabelComponentName identifies the component within a module.
	LabelComponentName = "component.opmodel.dev/name"

	// LabelBundleName identifies the OPM bundle that created this resource.
	LabelBundleName = "bundle.opmodel.dev/name"

	// LabelBundleNamespace identifies the namespace the bundle is deployed to.
	LabelBundleNamespace = "bundle.opmodel.dev/namespace"

	// LabelBundleVersion identifies the version of the bundle.
	LabelBundleVersion = "bundle.opmodel.dev/version"
)

// FieldManager is the name used for server-side apply field ownership.
const FieldManager = "opm"

// Labels represents a set of Kubernetes labels.
type Labels map[string]string

// ModuleLabels creates the standard labels for a module's resources.
func ModuleLabels(name, namespace, version, component string) Labels {
	labels := Labels{
		LabelManagedBy:       LabelManagedByValue,
		LabelModuleName:      name,
		LabelModuleNamespace: namespace,
	}

	if version != "" {
		labels[LabelModuleVersion] = version
	}

	if component != "" {
		labels[LabelComponentName] = component
	}

	return labels
}

// BundleLabels creates the standard labels for a bundle's resources.
func BundleLabels(name, namespace, version string) Labels {
	labels := Labels{
		LabelManagedBy:       LabelManagedByValue,
		LabelBundleName:      name,
		LabelBundleNamespace: namespace,
	}

	if version != "" {
		labels[LabelBundleVersion] = version
	}

	return labels
}

// Merge merges additional labels into this label set.
// Existing labels are not overwritten.
func (l Labels) Merge(other Labels) Labels {
	result := make(Labels, len(l)+len(other))
	for k, v := range l {
		result[k] = v
	}
	for k, v := range other {
		if _, exists := result[k]; !exists {
			result[k] = v
		}
	}
	return result
}

// ModuleSelector returns a label selector for finding resources belonging to a module.
func ModuleSelector(name, namespace string) map[string]string {
	return map[string]string{
		LabelManagedBy:       LabelManagedByValue,
		LabelModuleName:      name,
		LabelModuleNamespace: namespace,
	}
}

// BundleSelector returns a label selector for finding resources belonging to a bundle.
func BundleSelector(name, namespace string) map[string]string {
	return map[string]string{
		LabelManagedBy:       LabelManagedByValue,
		LabelBundleName:      name,
		LabelBundleNamespace: namespace,
	}
}

// IsOPMManaged checks if a resource is managed by OPM based on its labels.
func IsOPMManaged(labels map[string]string) bool {
	return labels[LabelManagedBy] == LabelManagedByValue
}

// InjectLabels adds the given labels to an unstructured object.
// Existing labels are preserved; new labels are added.
func InjectLabels(obj map[string]interface{}, labels Labels) {
	// Get or create metadata
	metadata, ok := obj["metadata"].(map[string]interface{})
	if !ok {
		metadata = make(map[string]interface{})
		obj["metadata"] = metadata
	}

	// Get or create labels
	existingLabels, ok := metadata["labels"].(map[string]interface{})
	if !ok {
		existingLabels = make(map[string]interface{})
	}

	// Add new labels (don't overwrite existing)
	for k, v := range labels {
		if _, exists := existingLabels[k]; !exists {
			existingLabels[k] = v
		}
	}

	metadata["labels"] = existingLabels
}

// SelectorString converts a selector map to a label selector string.
func SelectorString(selector map[string]string) string {
	if len(selector) == 0 {
		return ""
	}

	var parts []string
	for k, v := range selector {
		parts = append(parts, k+"="+v)
	}

	result := parts[0]
	for i := 1; i < len(parts); i++ {
		result += "," + parts[i]
	}
	return result
}
