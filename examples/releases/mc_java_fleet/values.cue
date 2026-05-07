// Single-server fleet values for the mc_java_fleet module.
// One Paper server, ephemeral storage, router exposed as ClusterIP.
package mc_java_fleet

values: {
	// releaseName must match the ModuleRelease metadata.name above.
	releaseName: "mc-java-fleet"
	domain:      "mc.example.com"
	namespace:   "default"

	servers: {
		survival: {
			paper: {}

			server: {
				motd:       "OPM Survival"
				maxPlayers: 10
				difficulty: "normal"
				mode:       "survival"
			}

			jvm: memory: "2G"

			storage: {
				data: {
					type: "emptyDir"
					size: "10Gi"
				}
				backups: {
					type: "emptyDir"
					size: "10Gi"
				}
			}

			backup: {
				enabled: false
				tar: {}
			}

			resources: {
				requests: {
					cpu:    "500m"
					memory: "2Gi"
				}
				limits: {
					cpu:    "2000m"
					memory: "4Gi"
				}
			}
		}
	}

	router: {
		port:        25565
		serviceType: "ClusterIP"
	}

	// Replace before applying — use a real Secret reference in production.
	rconPassword: value: "REPLACE-ME-rcon-password"
}
