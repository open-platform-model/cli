// Values provide concrete configuration for the Minecraft proxy module.
// These satisfy the #config schema defined in module.cue.
package main

// Concrete default values - testing configuration
values: {
	proxy: {
		type:       "BUNGEECORD"
		onlineMode: false
		memory:     "256M"
	}
	storage: data: {
		type: "emptyDir"
	}
	port:        25577
	serviceType: "ClusterIP"
	resources: {
		requests: {
			cpu:    "100m"
			memory: "256Mi"
		}
	}
}
