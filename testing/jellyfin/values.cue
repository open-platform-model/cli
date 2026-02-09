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
	configStorageSize: "10Gi"

	// Media library mount points.
	// These use emptyDir by default — operators SHOULD override with real
	// storage (NFS, hostPath, external PVCs) at ModuleRelease time.
	media: {
		tvshows: {
			mountPath: "/data/tvshows"
		}
		movies: {
			mountPath: "/data/movies"
		}
	}
}
