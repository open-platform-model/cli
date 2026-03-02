// Values provide concrete configuration for the RCON Web Admin module.
// These satisfy the #config schema defined in module.cue.
package main

// Concrete default values - connect to a local Minecraft server
values: {
	admin: {
		isAdmin:      true
		username:     "admin"
		password:     value: "CHANGE-ME"
		game:         "minecraft"
		rconHost:     "127.0.0.1"
		rconPort:     25575
		rconPassword: value: "CHANGE-ME"
	}
	httpPort:    80
	wsPort:      4327
	serviceType: "ClusterIP"
	resources: {
		requests: {
			cpu:    "50m"
			memory: "128Mi"
		}
	}
	// itzg/rcon writes its settings.json to /opt/rcon-web-admin-*/db/ at startup.
	// The restrictive default (readOnlyRootFilesystem + runAsUser 1000) causes EACCES.
	// Override to allow the app to write its database file.
	securityContext: {
		runAsNonRoot:             false
		readOnlyRootFilesystem:   false
		allowPrivilegeEscalation: false
		capabilities: drop: ["ALL"]
	}
}
