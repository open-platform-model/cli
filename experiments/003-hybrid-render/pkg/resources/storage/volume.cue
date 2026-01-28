package storage

import (
	core "test.com/experiment/pkg/core@v0"
	schemas "test.com/experiment/pkg/schemas@v0"
)

//////////////////////////////////////////////////////////////////
//// Volume Resource Definition
/////////////////////////////////////////////////////////////////

#VolumesResource: close(core.#Resource & {
	metadata: {
		apiVersion:  "opm.dev/resources/storage@v0"
		name:        "Volumes"
		description: "A volume definition for workloads"
		labels: {
			// "core.opm.dev/category":    "storage"
			"core.opm.dev/persistence": "true"
		}
	}

	// Default values for volumes resource
	#defaults: #VolumesDefaults

	// OpenAPIv3-compatible schema defining the structure of the volume spec
	#spec: volumes: [volumeName=string]: schemas.#VolumeSchema & {name: string | *volumeName}
})

#Volumes: close(core.#Component & {
	#resources: {(#VolumesResource.metadata.fqn): #VolumesResource}
})

#VolumesDefaults: close(schemas.#VolumeSchema & {
	// Default empty dir medium
	emptyDir?: {
		medium: *"node" | "memory"
	}
})
