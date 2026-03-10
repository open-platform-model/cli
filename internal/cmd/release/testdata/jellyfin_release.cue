// Minimal ModuleRelease fixture for unit testing.
// This is not a valid OPM module release — it's used to verify that
// LoadReleaseFile() can parse and evaluate a ModuleRelease file.
{
	kind: "ModuleRelease"
	metadata: {
		name:      "jellyfin"
		namespace: "media"
		version:   "0.1.0"
	}
	module: "example.com/modules/jellyfin:0.1.0"
	values: {
		replicas: 1
		image:    "jellyfin/jellyfin:latest"
	}
}
