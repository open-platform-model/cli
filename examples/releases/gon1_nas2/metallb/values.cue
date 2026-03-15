// MetalLB release values for gon1-nas2.
// L2 address pool: 10.10.0.180–10.10.0.199 (applied manually post-deploy).
package metallb

values: {
	image: {
		tag:        "v0.15.3"
		pullPolicy: "IfNotPresent"
	}

	controller: {
		logLevel: "info"
		replicas: 1
		resources: {
			requests: {
				cpu:    "100m"
				memory: "64Mi"
			}
			limits: {
				cpu:    "300m"
				memory: "128Mi"
			}
		}
	}

	speaker: {
		logLevel: "info"
		resources: {
			requests: {
				cpu:    "100m"
				memory: "64Mi"
			}
			limits: {
				cpu:    "300m"
				memory: "128Mi"
			}
		}
	}
}
