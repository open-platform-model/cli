// Concrete values for the Jellyfin release.
// See the public module schema at opmodel.dev/modules/jellyfin@v1
// (or modules/jellyfin/module.cue) for the full #config surface.
package jellyfin

values: {
	port:        8096
	puid:        1000
	pgid:        1000
	timezone:    "Europe/Stockholm"
	serviceType: "ClusterIP"

	storage: {
		config: {
			mountPath: "/config"
			type:      "pvc"
			size:      "10Gi"
		}
	}

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
}
