package main

// Production defaults for Zot registry
values: {
	image: {
		variant:    "full"
		tag:        "v2.1.14"
		digest:     ""
		pullPolicy: "IfNotPresent"
	}

	storage: {
		type:         "pvc"
		rootDir:      "/var/lib/registry"
		size:         "20Gi"
		storageClass: "standard"
		dedupe:       true
		
		gc: {
			enabled:  true
			delay:    "1h"
			interval: "24h"
		}
		
		scrub: {
			enabled:  true
			interval: "24h"
		}
	}

	http: {
		port:    5000
		address: "0.0.0.0"
	}

	log: {
		level: "info"
		audit: {
			enabled: true
		}
	}

	// Authentication with example credentials (admin:admin, user:user)
	// In production, replace with actual htpasswd content via external secret
	auth: {
		htpasswd: {
			credentials: {
				$opm:         "secret"
				$secretName:  "zot-htpasswd"
				$dataKey:     "htpasswd"
				$description: "htpasswd file with bcrypt-hashed passwords"
				// Example htpasswd with 'admin:admin' & 'user:user'
				value: """
					admin:$2y$05$vmiurPmJvHylk78HHFWuruFFVePlit9rZWGA/FbZfTEmNRneGJtha
					user:$2y$05$L86zqQDfH5y445dcMlwu6uHv.oXFgT6AiJCwpv3ehr7idc0rI3S2G
					"""
			}
		}
		accessControl: {
			adminUsers: ["admin"]
			repositories: {
				"**": {
					policies: [{
						users:   ["user"]
						actions: ["read"]
					}]
					defaultPolicy: []
				}
			}
		}
	}

	// Prometheus metrics
	metrics: {
		enabled: true
	}

	// Example sync configuration (commented out by default)
	// sync: {
	// 	registries: [{
	// 		urls:         ["https://docker.io"]
	// 		onDemand:     true
	// 		tlsVerify:    true
	// 		pollInterval: "6h"
	// 		content: [{
	// 			prefix: "library/**"
	// 		}]
	// 	}]
	// }

	replicas: 1

	resources: {
		requests: {
			memory: "256Mi"
			cpu:    "100m"
		}
		limits: {
			memory: "1Gi"
			cpu:    "500m"
		}
	}

	security: {
		runAsNonRoot:             true
		runAsUser:                1000
		runAsGroup:               1000
		readOnlyRootFilesystem:   false
		allowPrivilegeEscalation: false
		capabilities: {
			drop: ["ALL"]
		}
	}
}
