// Values provide concrete configuration for the mc-monitor module.
// These satisfy the #config schema defined in module.cue.
package main

// Concrete default values - Prometheus export monitoring a single Java server
values: {
	// === Export Mode ===
	// Prometheus HTTP scrape endpoint on :8080
	prometheus: {
		port: 8080
	}

	// === Server Targets ===
	// Single Java server — update host to match your minecraft-java service name
	javaServers: [{
		host: "server.default.svc"
		// port defaults to 25565
	}]

	// === Shared Settings ===
	timeout: "1m0s"

	// === Networking ===
	serviceType: "ClusterIP"

	// === Resources ===
	// mc-monitor is lightweight (~15MB binary, minimal memory)
	resources: {
		requests: {
			cpu:    "50m"
			memory: "64Mi"
		}
		limits: {
			cpu:    "200m"
			memory: "128Mi"
		}
	}
}
