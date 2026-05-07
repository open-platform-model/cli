// Concrete values for the Garage release.
// adminToken and rpcSecret are REQUIRED — replace the placeholders below
// before applying to a real cluster.
//
// Generate a strong rpcSecret with: openssl rand -hex 32
package garage

values: {
	// REQUIRED — admin API bearer token. Replace before applying.
	adminToken: "REPLACE-ME-admin-token"

	// REQUIRED — 64 hex chars. Generate with: openssl rand -hex 32
	// The value below is a placeholder of the correct shape.
	rpcSecret: "0000000000000000000000000000000000000000000000000000000000000000"

	region:      "garage"
	serviceType: "ClusterIP"

	storage: {
		type: "pvc"
		size: "5Gi"
	}

	resources: {
		requests: {
			cpu:    "100m"
			memory: "256Mi"
		}
		limits: {
			cpu:    "500m"
			memory: "512Mi"
		}
	}
}
