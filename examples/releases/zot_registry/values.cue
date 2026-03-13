package main

// Production defaults for Zot registry (OPM Catalog)
values: {
	image: {
		variant:    "minimal"
		tag:        "v2.1.14"
		digest:     ""
		pullPolicy: "IfNotPresent"
	}

	storage: {
		type:         "pvc"
		rootDir:      "/var/lib/registry"
		size:         "50Gi"
		storageClass: "standard"
		dedupe:       true
		
		gc: {
			enabled:  true
			delay:    "1h"
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

	auth: {
		htpasswd: {
			credentials: {
				value:        "admin:$2y$05$vmiurPmJvHylk78HHFWuruFFVePlit9rZWGA/FbZfTEmNRneGJtha\n"
			}
		}
		accessControl: {
			adminUsers: ["admin"]
			repositories: {
				"**": {
					policies: []
					defaultPolicy: ["read"] // Allows public/anonymous read access
				}
			}
		}
	}

	httpRoute: {
		hostnames: ["registry.example.com"]
		gatewayRef: {
			name:      "external-gateway"
			namespace: "gateway-system"
		}
	}

	// Prometheus metrics
	metrics: {
		enabled: true
	}

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
