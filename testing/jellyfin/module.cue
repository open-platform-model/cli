// Package main defines the Jellyfin media server module.
// A single-container stateful application using the LinuxServer.io image:
// - module.cue: metadata and config schema
// - components.cue: component definitions
// - values.cue: default values
package main

import (
	"opmodel.dev/core@v0"
)

// Module definition
core.#Module

// Module metadata
metadata: {
	apiVersion:       "example.com/jellyfin@v0"
	name:             "jellyfin"
	version:          "0.1.0"
	description:      string | *"Jellyfin media server - a free software media system"
	defaultNamespace: "jellyfin"
}

// Schema only - constraints for users, no defaults
#config: {
	// Container image
	image: string

	// Exposed service port for the web UI
	port: int & >0 & <=65535

	// LinuxServer.io user/group identity
	puid: int | *1000
	pgid: int | *1000

	// Container timezone
	timezone: string

	// Optional: published server URL for client auto-discovery
	publishedServerUrl?: string

	// PVC size for the /config directory
	configStorageSize: string

	// Media library mount points (struct-keyed)
	media: [Name=string]: {
		mountPath: string
	}
}

// Values must satisfy #config - concrete values in values.cue
values: #config
