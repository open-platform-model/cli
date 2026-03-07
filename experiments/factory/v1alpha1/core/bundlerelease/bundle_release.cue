package bundlerelease

import (
	cue_uuid "uuid"
	t "opmodel.dev/core/types@v1"
	modulerelease "opmodel.dev/core/modulerelease@v1"
	bundle "opmodel.dev/core/bundle@v1"
)

// #BundleRelease: The concrete deployment instance for a #Bundle.
// Binds a bundle to consumer-supplied values, producing a map of #ModuleRelease
// instances ready for provider/transformer rendering.
#BundleRelease: {
	apiVersion: "opmodel.dev/core/v1alpha1"
	kind:       "BundleRelease"

	metadata: {
		name!: t.#NameType

		// Generate a stable UUID for this release based on the bundle's UUID and the release name.
		uuid: t.#UUIDType & cue_uuid.SHA1(t.OPMNamespace, "\(#bundleMetadata.uuid):\(name)")

		labels?:      t.#LabelsAnnotationsType
		annotations?: t.#LabelsAnnotationsType
	}

	// Reference to the Bundle to deploy.
	#bundle!:        bundle.#Bundle
	#bundleMetadata: #bundle.metadata

	// Concrete values satisfying #bundle.#config.
	values: _

	// Unify the bundle definition with consumer-supplied values.
	// This resolves all #config references within #instances.
	let unifiedBundle = #bundle & {#config: values}

	// Capture the release name as a let alias before the comprehension.
	// This allows CUE to resolve the concrete string value of metadata.name
	// directly in the interpolation without carrying the #NameType constraints
	// into the comprehension scope, which would cause an invalid interpolation error.
	let _name = metadata.name

	// Output: each #BundleInstance becomes a #ModuleRelease.
	// Keys are the instance names from #bundle.#instances.
	//
	// Release name: "{bundleRelease.name}-{instanceName}"
	// Namespace:    inst.metadata.namespace (required on every #BundleInstance)
	// Values:       inst.values if set by the bundle author; omitted otherwise
	//               (module defaults apply when values is absent)
	releases: {
		for instName, inst in unifiedBundle.#instances {
			let _ns   = inst.metadata.namespace
			let _inst = inst
			(instName): modulerelease.#ModuleRelease & {
				metadata: {
					name:      "\(_name)-\(instName)"
					namespace: _ns
				}
				#module: _inst.module

				// Pass inst.values if the bundle author provided them.
				// Guard via != _|_ so omitted values don't inject bottom into #ModuleRelease.
				if _inst.values != _|_ {
					values: _inst.values
				}
			}
		}
	}
}

#BundleReleaseMap: [string]: #BundleRelease
