// Values provide concrete configuration for the Jellyfin module.
// These satisfy the #config schema defined in module.cue.
package main

// Concrete default values
values: {
	// LinuxServer.io Jellyfin image
	image: "lscr.io/linuxserver/jellyfin:latest"

	// Web UI exposed port
	port: 8096

	// LinuxServer.io user/group identity (default: 1000)
	puid: 1000
	pgid: 1000

	// Container timezone
	timezone: "Etc/UTC"

	// PVC size for Jellyfin config/metadata directory.
	// Can grow to 50GB+ for large collections (thumbnails, metadata cache).
	configStorageSize: "15Gi"

	// Media library mount points with persistent storage.
	// Operators can override sizes at ModuleRelease time.
	media: {
		tvshows: {
			mountPath: "/data/tvshows"
			size:      "100Gi"
		}
		movies: {
			mountPath: "/data/movies"
			size:      "100Gi"
		}
	}
}
