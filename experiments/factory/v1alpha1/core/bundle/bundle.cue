package bundle

import (
	cue_uuid "uuid"
	t "opmodel.dev/core/types@v1"
	module "opmodel.dev/core/module@v1"
	policy "opmodel.dev/core/policy@v1"
)

// Local alias — workaround: CUE's import tracker does not always see module.#Module
// when used only inside definition fields. Declaring the alias at package scope
// ensures the import is tracked correctly.
#Module: module.#Module

// #BundleInstance: A single module instance within a #Bundle.
//
// Each instance carries:
//   - module!:   a required reference to a concrete #Module
//   - metadata:  name (auto-derived from map key), required namespace, optional labels/annotations
//   - values?:   optional config values satisfying the module's #config schema
//
// The bundle author uses `values` to wire bundle-level #config fields into the
// module's config. Three patterns are supported:
//   1. Hardcode:   values: { maxPlayers: 50 }          — consumer cannot override
//   2. Wire:       values: { maxPlayers: C.maxPlayers } — consumer sets via bundle #config
//   3. Passthrough: values: C.myModule                 — consumer gets the full module schema
//
// metadata.name is auto-derived from the #instances map key by the #Bundle constraint.
// The bundle author does not set it manually.
#BundleInstance: {
	module!: #Module
	metadata: {
		// name is auto-derived from the #instances map key — do not set manually.
		name!: t.#NameType

		// namespace is the target Kubernetes namespace for this module instance.
		namespace!: t.#NameType

		// Optional labels inherited by all resources in this module instance.
		labels?: t.#LabelsAnnotationsType

		// Optional annotations inherited by all resources in this module instance.
		annotations?: t.#LabelsAnnotationsType
	}

	// values wires config into the module's #config schema.
	// If omitted, module defaults apply. The type constraint ensures values
	// are validated against the module's declared #config at definition time.
	values?: module.#config
}

// #Bundle: Defines a collection of modules grouped for distribution.
// Bundles enable grouping related modules for easier deployment and management.
#Bundle: {
	apiVersion: "opmodel.dev/core/v1alpha1"
	kind:       "Bundle"

	metadata: {
		modulePath!: t.#ModulePathType                                     // Example: "opmodel.dev/bundles/core"
		name!:       t.#NameType                                           // Example: "example-bundle"
		version!:    t.#MajorVersionType                                   // Example: "v1"
		fqn:         t.#BundleFQNType & "\(modulePath)/\(name):\(version)" // Example: "example.com/bundles/game-stack:v1"

		// Unique identifier for the bundle, computed as a UUID v5 (SHA1) of the FQN using the OPM namespace UUID.
		uuid: t.#UUIDType & cue_uuid.SHA1(t.OPMNamespace, fqn)
		#definitionName: (t.#KebabToPascal & {"in": name}).out

		// Human-readable description of the bundle
		description?: string

		// Optional metadata labels for categorization and filtering
		labels?: t.#LabelsAnnotationsType

		// Optional metadata annotations for bundle behavior hints
		annotations?: t.#LabelsAnnotationsType
	}

	// Instances in this bundle — each maps a name to a module instance.
	// metadata.name is automatically set to the map key; the author does not set it manually.
	// The same module MAY appear multiple times under different instance names.
	#instances!: [instanceName=string]: #BundleInstance & {metadata: name: string | *instanceName}

	// Bundle-level policies — cross-module governance.
	// appliesTo.matchLabels selects components across all modules in the bundle.
	#policies?: [string]: policy.#Policy

	// Bundle-level config schema — consumer-facing.
	// Bundle author wires this to module configs via CUE unification.
	#config!: _

	// debugValues: Example values for testing and debugging.
	// It is unified and validated in the runtime.
	debugValues: _
}

#BundleDefinitionMap: [string]: #Bundle
