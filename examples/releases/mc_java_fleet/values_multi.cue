// Multi-server fleet values for the mc_java_fleet module.
// Two Paper servers (survival + creative) sharing one mc-router exposed
// as a LoadBalancer. Players reach each server by hostname:
//   survival.mc.example.com → survival server
//   creative.mc.example.com → creative server
//
// Use with: opm release build .../release.cue -f .../values_multi.cue
package mc_java_fleet

values: {
	releaseName: "mc-java-fleet"
	domain:      "mc.example.com"
	namespace:   "default"

	servers: {
		survival: {
			paper: {}
			server: {
				motd:       "OPM Survival"
				maxPlayers: 20
				difficulty: "normal"
				mode:       "survival"
			}
			jvm: memory: "2G"
			storage: {
				data: {type:    "emptyDir", size: "10Gi"}
				backups: {type: "emptyDir", size: "10Gi"}
			}
			backup: {enabled: false, tar: {}}
		}

		creative: {
			paper: {}
			server: {
				motd:       "OPM Creative"
				maxPlayers: 10
				difficulty: "peaceful"
				mode:       "creative"
				pvp:        false
			}
			jvm: memory: "2G"
			storage: {
				data: {type:    "emptyDir", size: "10Gi"}
				backups: {type: "emptyDir", size: "10Gi"}
			}
			backup: {enabled: false, tar: {}}
		}
	}

	router: {
		port:        25565
		serviceType: "LoadBalancer"
		defaultServer: {
			host: "mc-java-fleet-server-survival.default.svc"
			port: 25565
		}
	}

	rconPassword: value: "REPLACE-ME-rcon-password"
}
