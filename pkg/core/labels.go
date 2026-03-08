package core

// OPM Kubernetes label keys applied to all managed resources.
const (
	// LabelManagedBy is the standard Kubernetes label indicating the manager.
	// Value is always "open-platform-model".
	LabelManagedBy = "app.kubernetes.io/managed-by"

	// LabelManagedByValue is the value for the LabelManagedBy label.
	LabelManagedByValue = "open-platform-model"

	// LabelComponent is the OPM infrastructure label that categorizes the type
	// of OPM-managed object (e.g., "inventory"). Distinct from component names
	// set by CUE transformers on application resources.
	LabelComponent = "opmodel.dev/component"

	// LabelComponentName is the label injected by the CUE catalog on all application
	// resources to record which component produced them. Value is the component name.
	// Set by module.cue in the v1alpha1 catalog:
	//   labels: "component.opmodel.dev/name": name
	// Used by inventory to track provenance for component-rename safety checks.
	LabelComponentName = "component.opmodel.dev/name"

	// LabelModuleName is the label injected by the CUE catalog on all application
	// resources to record which module produced them. Value is the module name.
	LabelModuleName = "module.opmodel.dev/name"

	// LabelModuleUUID is the module identity UUID label for resource discovery.
	LabelModuleUUID = "module.opmodel.dev/uuid"

	// LabelReleaseName is the release name label.
	LabelModuleReleaseName = "module-release.opmodel.dev/name"
	LabelReleaseName = LabelModuleReleaseName

	// LabelReleaseNamespace is the release namespace label.
	LabelModuleReleaseNamespace = "module-release.opmodel.dev/namespace"
	LabelReleaseNamespace = LabelModuleReleaseNamespace

	// LabelReleaseUUID is the release identity UUID label for resource discovery.
	LabelModuleReleaseUUID = "module-release.opmodel.dev/uuid"
	LabelReleaseUUID = LabelModuleReleaseUUID

	// LabelWorkloadType is the v1alpha1 label for workload type classification.
	// Required on components using #Container resource.
	// Values: "stateless", "stateful", "daemon", "task", "scheduled-task".
	LabelWorkloadType = "core.opmodel.dev/workload-type"
)
