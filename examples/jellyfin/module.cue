// Package main defines the Jellyfin media server module.
// A single-container stateful application using the LinuxServer.io image:
// - module.cue: metadata and config schema
// - components.cue: component definitions
// - values.cue: default values
package main

import (
	"opmodel.dev/core@v1"
	schemas "opmodel.dev/schemas@v1"
)

// Module definition
core.#Module

// Module metadata
metadata: {
	modulePath:       "opmodel.dev/modules"
	name:             "jellyfin"
	version:          "0.1.0"
	description:      string | *"Jellyfin media server - a free software media system"
	defaultNamespace: "jellyfin"
}

// Schema only - constraints for users, no defaults
#config: {
	// Container image
	image: schemas.#Image & {
		repository: string | *"linuxserver/jellyfin"
		tag:        string | *"latest"
		digest:     string | *""
	}

	// Exposed service port for the web UI
	port: int & >0 & <=65535 | *8096

	// LinuxServer.io user/group identity
	puid: int | *1000
	pgid: int | *1000

	// Container timezone
	timezone: string | *"Etc/UTC"

	// Optional: published server URL for client auto-discovery
	publishedServerUrl?: string

	// PVC size for the /config directory
	configStorageSize: string | *"10Gi"

	// Media library mount points with persistent storage
	media?: [Name=string]: {
		mountPath: string
		type:      "pvc" | *"emptyDir"
		size:      string
	}
}
