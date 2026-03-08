package zot_registry

import (
	m "opmodel.dev/core/module@v1"
	schemas "opmodel.dev/schemas@v1"
)

m.#Module

metadata: {
	modulePath:  "opmodel.dev/modules"
	name:        "zot-registry"
	version:     "0.1.0"
	description: "Production-ready Zot OCI registry with authentication, metrics, sync, and storage management"
	labels: {
		"app.kubernetes.io/component": "registry"
	}
}

#config: {
	// Image configuration
	image: {
		variant:    "full" | *"minimal"
		tag:        string | *"v2.1.14"
		digest:     string | *""
		pullPolicy: "Always" | *"IfNotPresent" | "Never"
	}

	// Storage configuration
	storage: {
		type:         "pvc" | *"emptyDir"
		rootDir:      string | *"/var/lib/registry"
		size:         string | *"20Gi"
		storageClass: string | *"standard"

		// Dedupe reduces storage by using hard links for identical blobs
		dedupe?: bool | *true

		// Garbage collection
		gc?: {
			enabled:  bool | *true
			delay:    string | *"1h"  // Delay before removing unreferenced blobs
			interval: string | *"24h" // How often to run GC
		}

		// Scrub validates data integrity (full variant only)
		scrub?: {
			enabled:  bool | *true
			interval: string | *"24h"
		}
	}

	// HTTP server configuration
	http: {
		port:    int | *5000
		address: string | *"0.0.0.0"
	}

	// Logging configuration
	log: {
		level: "debug" | *"info" | "warn" | "error"
		audit?: {
			enabled: bool | *true
		}
	}

	// Authentication and access control (optional)
	auth?: {
		htpasswd: {
			// htpasswd file content as a secret
			credentials: schemas.#Secret & {
				$secretName: string | *"zot-htpasswd"
				$dataKey:    string | *"htpasswd"
			}
		}
		accessControl?: {
			// Admin users with full permissions
			adminUsers: [...string]

			// Per-repository policies
			repositories?: [string]: {
				policies: [...{
					users: [...string]
					actions: [...("read" | "create" | "update" | "delete")]
				}]
				defaultPolicy: [...("read" | "create" | "update" | "delete")]
			}
		}
	}

	// Registry synchronization (mirroring)
	sync?: {
		registries: [...{
			// Upstream registry URLs
			urls: [...string]

			// Pull images on-demand vs periodic sync
			onDemand: bool | *true

			// Verify upstream TLS
			tlsVerify: bool | *true

			// Periodic sync interval (if not onDemand)
			pollInterval: string | *"6h"

			// Content filters
			content?: [...{
				// Repository prefix pattern (e.g., "library/**")
				prefix: string

				// Destination path override
				destination?: string

				// Tag filters
				tags?: {
					regex?:  string
					semver?: bool
				}
			}]
		}]
	}

	// Metrics endpoint
	metrics?: {
		enabled: bool | *true
	}

	// Ingress/HTTPRoute (optional)
	httpRoute?: {
		hostnames: [...string]
		tls?: {
			secretName: string
		}
		gatewayRef?: {
			name:      string
			namespace: string
		}
	}

	// Workload configuration
	replicas: int & >=1 & <=10 | *1

	// Resource requirements
	resources: schemas.#ResourceRequirementsSchema | *{
		requests: {
			memory: "256Mi"
			cpu:    "100m"
		}
		limits: {
			memory: "1Gi"
			cpu:    "500m"
		}
	}

	// Security context
	security: schemas.#SecurityContextSchema | *{
		runAsNonRoot:             true
		runAsUser:                1000
		runAsGroup:               1000
		readOnlyRootFilesystem:   false // Needs to write to /tmp and data dir
		allowPrivilegeEscalation: false
		capabilities: {
			drop: ["ALL"]
		}
	}
}
