// Example: Testing/Development Server (Ephemeral Storage)
// Usage: opm mod build examples/minecraft -f values-testing.cue
//
// WARNING: All data is deleted when the pod restarts!
// Only use this for quick testing and development.
package main

values: {
	server: {
		image:   "itzg/minecraft-server:latest"
		type:    "PAPER"
		version: "LATEST"
		eula:    true
		motd:    "Test Server - Data Not Saved!"

		maxPlayers: 5
		difficulty: "peaceful"
		mode:       "creative" // Creative mode for testing

		pvp:                false
		enableCommandBlock: true // Enable for testing

		rcon: {
			password: "test"
			port:     25575
		}
	}

	storage: {
		// Ephemeral storage - data deleted on pod restart
		data: {
			type: "emptyDir"
		}
		backups: {
			type: "emptyDir"
		}
	}

	// Disable backups for testing
	backup: {
		enabled: false
		image:   "itzg/mc-backup:latest"
		method:  "tar"
		interval: "24h"
		initialDelay: "5m"
	}

	// Minimal resources for testing
	resources: {
		requests: {
			cpu:    "500m"
			memory: "1Gi"
		}
		limits: {
			cpu:    "2000m"
			memory: "4Gi"
		}
	}

	port: 25565

	// ClusterIP - use kubectl port-forward for access
	serviceType: "ClusterIP"
}
