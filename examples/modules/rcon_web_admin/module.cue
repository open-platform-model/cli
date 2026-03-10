// Package main defines the RCON Web Admin module for managing game servers via browser.
// A stateless web application using itzg/rcon:
// - module.cue: metadata and config schema
// - components.cue: component definitions with dual-port networking (HTTP + WebSocket)
// - values.cue: default values
//
// Config schema mirrors the itzg/rcon container environment variables.
package rcon_web_admin

import (
	m "opmodel.dev/core/module@v1"
	schemas "opmodel.dev/schemas@v1"
)

// Module definition
m.#Module

// #workloadComponent is the name of the primary workload component in this module.
#workloadComponent: "admin"

// Module metadata
metadata: {
	modulePath:       "example.com/modules"
	name:             "rcon-web-admin"
	version:          "0.1.0"
	description:      "Web-based RCON admin console for game servers"
	defaultNamespace: "default"
}

// Schema only - constraints for users, no defaults
#config: {
	// === Admin Configuration ===
	admin: {
		// Container image for rcon-web-admin
		image: schemas.#Image & {
			repository: string | *"itzg/rcon"
			tag:        string | *"latest"
			digest:     string | *""
		}

		// Whether this user has admin privileges in the web UI
		isAdmin: bool

		// Login credentials for the web UI
		username: string
		// Web UI login password — stored in a K8s Secret
		password: schemas.#Secret & {
			$secretName: "admin-credentials"
			$dataKey:    "password"
		}

		// Game type (determines command palette and widget set)
		game: string

		// Optional: Display name for the server in the web UI
		serverName?: string

		// RCON connection details for the target game server
		rconHost: string
		rconPort: _#portSchema
		// RCON password for the target game server — stored in a K8s Secret
		rconPassword: schemas.#Secret & {
			$secretName: "admin-credentials"
			$dataKey:    "rcon-password"
		}

		// Optional: Restrict available commands to this list
		restrictCommands?: [...string]

		// Optional: Restrict available widgets to this list
		restrictWidgets?: [...string]

		// Optional: Prevent users from changing widget options
		immutableWidgetOptions?: bool

		// Optional: Use WebSocket-based RCON (for servers that support it)
		websocketRcon?: bool
	}

	// === Networking ===
	// HTTP port for the web UI
	httpPort: _#portSchema

	// WebSocket port for real-time RCON communication
	wsPort: _#portSchema

	// Service type for network exposure
	serviceType: "ClusterIP" | "LoadBalancer" | "NodePort"

	// Optional: HTTPRoute for Gateway API based ingress to the web UI
	httpRoute?: {
		hostnames: [...string]
		gatewayRef?: {
			name:       string
			namespace?: string
		}
	}

	// === Resource Limits (catalog-standard shape) ===
	resources?: schemas.#ResourceRequirementsSchema

	// === Security Context ===
	securityContext?: schemas.#SecurityContextSchema
}

_#portSchema: uint & >0 & <=65535


debugValues: {
	admin: {
		isAdmin:      true
		username:     "admin"
		password:     value: "change-me"
		game:         "minecraft"
		serverName:   "Minecraft Java"
		rconHost:     "mc-java-server.default.svc"
		rconPort:     25575
		rconPassword: value: "change-me"
	}
	httpPort:    8080
	wsPort:      4327
	serviceType: "ClusterIP"
	resources: requests: {
		cpu:    "50m"
		memory: "128Mi"
	}
	securityContext: {
		runAsNonRoot:             false
		readOnlyRootFilesystem:   false
		allowPrivilegeEscalation: false
		capabilities: drop: ["ALL"]
	}
}
