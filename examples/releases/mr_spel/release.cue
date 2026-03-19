// mr_spel — Wolf game streaming server release.
//
// Streams GPU-accelerated desktops and games to Moonlight clients.
// Multi-user: each paired client gets an isolated session.
//
// Target: bare-metal node with an Intel/AMD GPU running Talos/K8s.
// Networking: LoadBalancer (MetalLB) for external Moonlight access.
package mr_spel

import (
	mr "opmodel.dev/core/modulerelease@v1"
	wolf "opmodel.dev/examples/modules/wolf"
)

// Module release definition
mr.#ModuleRelease

metadata: {
	name:      "wolf"
	namespace: "gaming"
}

#module: wolf

values: {
	// Wolf display name shown in Moonlight's host list
	wolf: {
		hostname:            "spel"
		supportHevc:         true
		logLevel:            "INFO"
		stopContainerOnExit: true
	}

	// Intel/AMD GPU via DRI (change type to "nvidia" and set driverPath for NVIDIA)
	gpu: {
		type:       "intelamd"
		renderNode: "/dev/dri/renderD128"
	}

	// Docker-in-Docker: cache game images on a PVC to avoid slow re-pulls
	dind: {
		storage: {
			type:         "pvc"
			size:         "100Gi"
			storageClass: "local-path"
		}
		resources: {
			requests: {cpu: "500m", memory: "512Mi"}
			limits: {cpu: "4000m", memory: "4Gi"}
		}
	}

	// LoadBalancer for external Moonlight access (requires MetalLB or equivalent)
	networking: serviceType: "LoadBalancer"

	// Persistent storage for Wolf config, paired clients, and per-user app state
	storage: config: {
		type:         "pvc"
		size:         "50Gi"
		storageClass: "local-path"
	}

	// Wolf container resource limits
	resources: {
		requests: {cpu: "1000m", memory: "2Gi"}
		limits: {cpu: "8000m", memory: "8Gi"}
	}
}
