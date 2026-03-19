// mr_spel — Wolf game streaming server release.
//
// Streams GPU-accelerated desktops and games to Moonlight clients.
// Multi-user: each paired client gets an isolated session.
//
// Target: lnn1-mrspel — bare-metal Talos node, AMD Radeon 780M iGPU (renderD128).
// Networking: hostNetwork on 192.168.11.224 — Moonlight connects on standard Wolf ports.
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

	// Docker-in-Docker: emptyDir for experimentation (layers rebuilt on pod restart).
	// Switch to pvc + local-path once the Talos user volume is provisioned.
	dind: {
		storage: {
			type: "emptyDir"
		}
		resources: {
			requests: {cpu: "500m", memory: "512Mi"}
			limits: {cpu: "4000m", memory: "4Gi"}
		}
	}

	// ClusterIP — Wolf uses hostNetwork: true so Moonlight connects directly to
	// 192.168.11.224 on Wolf's standard ports. A ClusterIP Service still provides
	// internal DNS for observability but carries no external traffic.
	networking: serviceType: "ClusterIP"

	// hostPath on the EPHEMERAL partition — no PVC needed.
	// /var/lib is writable on Talos (EPHEMERAL, 498GB sdb4).
	// Switch to pvc + local-path once the Talos user volume is provisioned.
	storage: config: {
		type:         "hostPath"
		path:         "/var/lib/wolf"
		hostPathType: "DirectoryOrCreate"
	}

	// Wolf container resource limits
	resources: {
		requests: {cpu: "1000m", memory: "2Gi"}
		limits: {cpu: "8000m", memory: "8Gi"}
	}
}
