package kubernetes

// OPM labels applied to all managed resources.
const (
	LabelManagedBy      = "app.kubernetes.io/managed-by"
	labelManagedByValue = "open-platform-model"
	LabelReleaseName    = "module-release.opmodel.dev/name"
	// LabelComponent is the OPM infrastructure label that categorizes the type
	// of OPM-managed object (e.g., "inventory"). Distinct from LabelComponentName
	// which is set by CUE transformers on application resources.
	LabelComponent = "opmodel.dev/component"
	// LabelModuleUUID is the module identity UUID label for resource discovery.
	LabelModuleUUID = "module.opmodel.dev/uuid"
	// LabelReleaseUUID is the release identity UUID label for resource discovery.
	LabelReleaseUUID = "module-release.opmodel.dev/uuid"
)

// fieldManagerName is the field manager used for server-side apply.
const fieldManagerName = "opm"
