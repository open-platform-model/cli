// Package main defines the mc-router module for TCP hostname routing of Minecraft servers.
// A stateless application using itzg/mc-router:
// - module.cue: metadata and config schema
// - components.cue: component definitions with router container
// - values.cue: default values
//
// Config schema mirrors the itzg/mc-router environment variable surface area.
package main

import (
	"opmodel.dev/core@v1"
	schemas "opmodel.dev/schemas@v1"
)

// Module definition
core.#Module

// Module metadata
metadata: {
	modulePath:       "example.com/modules"
	name:             "mc-router"
	version:          "0.1.0"
	description:      "TCP hostname router for Minecraft servers"
	defaultNamespace: "default"
}

// Schema only - constraints for users, no defaults
#config: {
	// === Router Configuration ===
	router: {
		// Container image for mc-router
		image: schemas.#Image & {
			repository: string | *"itzg/mc-router"
			tag:        string | *"1.40.3"
			digest:     string | *"sha256:10881cb6b74ebf8957d309f29952ea677eccd1b474d1e106466c94e8890b0788"
		}

		// Maximum connection rate per second (optional)
		connectionRateLimit: int & >0 | *1

		// Enable debug logging
		debug: bool | *false

		// Simplify SRV record lookup
		simplifySrv: bool | *false

		// Enable PROXY protocol for downstream servers
		useProxyProtocol: bool | *false

		// Default server when no hostname matches
		defaultServer: {
			host: string
			port: _#portSchema
		}

		// Static hostname-to-server mappings
		mappings!: [...{
			externalHostname: string
			host:             string
			port?:            _#portSchema
		}]

		// Auto-scale configuration (wake/sleep StatefulSets on player connect/disconnect)
		autoScale?: {
			up?: {
				enabled: bool
			}
			down?: {
				enabled: bool
				after?:  string
			}
		}

		// Metrics backend configuration
		metrics?: {
			backend: "discard" | "expvar" | "influxdb" | "prometheus"
		}

		// REST API configuration
		api: {
			enabled: bool | *false
			port:    _#portSchema | *8080
		}
	}

	// === Networking ===
	// Minecraft listening port
	port: _#portSchema | *25565

	// Service type for network exposure
	serviceType: *"ClusterIP" | "NodePort" | "LoadBalancer"

	// === Resource Limits (catalog-standard shape) ===
	resources?: schemas.#ResourceRequirementsSchema

	// === Scaling ===
	replicas?: uint & >0 | *1
}

_#portSchema: uint & >0 & <=65535
