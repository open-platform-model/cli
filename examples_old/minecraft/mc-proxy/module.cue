// Package main defines the Minecraft proxy module for BungeeCord/Waterfall/Velocity.
// A stateful proxy using itzg/bungeecord:
// - module.cue: metadata and config schema
// - components.cue: component definitions with proxy container
// - values.cue: default values
//
// Config schema mirrors the itzg/bungeecord Helm chart values.yaml surface area.
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
	name:             "minecraft-proxy"
	version:          "0.1.0"
	description:      "Minecraft proxy server (BungeeCord/Waterfall/Velocity)"
	defaultNamespace: "default"
}

// Schema only - constraints for users, no defaults
#config: {
	// === Proxy Configuration ===
	proxy: {
		// Container image for the proxy server
		image: schemas.#Image & {
			repository: string | *"itzg/bungeecord"
			tag:        string | *"latest"
			digest:     string | *""
		}

		// Proxy type determines which proxy software to run
		type: "BUNGEECORD" | "WATERFALL" | "VELOCITY" | "CUSTOM"

		// Check accounts against Minecraft account service
		onlineMode?: bool

		// List of plugin URLs to download at startup
		plugins?: [...string]

		// JVM memory allocation (e.g., "512M", "1G")
		memory?: string

		// General JVM options
		jvmOpts?: string

		// JVM -XX options (precede general options)
		jvmXXOpts?: string

		// Path to the proxy config file inside the container
		configFilePath?: string

		// Inline config content (YAML for BungeeCord/Waterfall, TOML for Velocity)
		configContent?: string

		// === Type-Specific Configuration ===
		// WATERFALL
		waterfallVersion?: string
		waterfallBuildId?: string

		// VELOCITY
		velocityVersion?: string

		// CUSTOM
		jarUrl?:  string
		jarFile?: string

		// === RCON Configuration ===
		rcon?: {
			enabled: bool
			port:    _#portSchema
			// RCON password — stored in a K8s Secret
			password: schemas.#Secret & {
				$secretName: "proxy-secrets"
				$dataKey:    "rcon-password"
			}
		}

		// === Extra Ports ===
		extraPorts?: [...{
			name:          string
			containerPort: _#portSchema
			protocol:      *"TCP" | "UDP"
		}]
	}

	// === Networking ===
	// Default proxy listening port
	port: _#portSchema

	// Service type for network exposure
	serviceType: "ClusterIP" | "LoadBalancer" | "NodePort"

	// === Storage Configuration ===
	storage: {
		// Proxy data volume (config, plugins, logs)
		data: {
			type: "pvc" | "hostPath" | "emptyDir"

			// For PVC
			size?:         string
			storageClass?: string

			// For hostPath
			path?:         string
			hostPathType?: "Directory" | "DirectoryOrCreate"
		}
	}

	// === Resource Limits (catalog-standard shape) ===
	resources?: schemas.#ResourceRequirementsSchema

	// === Security Context ===
	securityContext?: schemas.#SecurityContextSchema
}

_#portSchema: uint & >0 & <=65535
