// Package velocity defines the Velocity proxy server module.
// Velocity is a high-performance, extensible Minecraft proxy server.
// It forwards player connections from the internet to backend Minecraft servers.
//
// This minimal module covers:
//   - module.cue: metadata and config schema
//   - components.cue: stateless proxy container with network exposure
package mc_velocity

import (
	m "opmodel.dev/core/module@v1"
	schemas "opmodel.dev/schemas@v1"
)

// Module definition
m.#Module

// #workloadComponent is the name of the primary workload component in this module.
// Bundles reference this to construct Kubernetes Service DNS names:
//   {releaseName}-{instanceName}-{#workloadComponent}.{namespace}.svc
#workloadComponent: "proxy"

// Module metadata
metadata: {
	modulePath:       "example.com/modules"
	name:             "velocity-proxy"
	version:          "0.1.0"
	description:      "Velocity Minecraft proxy server"
	defaultNamespace: "default"
}

// Config schema — constraints for users, no defaults
#config: {
	// Container image for the Velocity proxy
	image: schemas.#Image & {
		repository: string | *"itzg/mc-proxy"
		tag:        string | *"latest"
		digest:     string | *""
	}

	// Velocity type — always VELOCITY for this module
	type: "VELOCITY"

	// Message of the day shown on the server list
	motd: string | *"A Velocity Proxy"

	// Online mode: verify player accounts against Mojang
	onlineMode: bool | *true

	// Maximum number of players the proxy will accept
	maxPlayers: uint & >0 & <=10000 | *500

	// Port the proxy listens on for player connections
	bindPort: _#portSchema | *25577

	// Player info forwarding mode to backend servers.
	// NONE: No forwarding (backend servers run in offline mode).
	// LEGACY: BungeeCord-compatible forwarding (Spigot backend required).
	// MODERN: Velocity native forwarding (requires Velocity support on backend).
	forwardingMode: *"MODERN" | "LEGACY" | "NONE"

	// Forwarding secret — required when forwardingMode is MODERN.
	// Shared secret between proxy and backend servers.
	forwardingSecret?: string
}

_#portSchema: uint & >0 & <=65535

debugValues: {
	image: {
		repository: "itzg/mc-proxy"
		tag:        "latest"
		digest:     ""
	}
	type: "VELOCITY"
	motd: "Welcome to the Velocity Proxy!"
	onlineMode: true
	maxPlayers: 500
	bindPort: 25577
	forwardingMode: "MODERN"
	forwardingSecret: "change-me-in-production"
}