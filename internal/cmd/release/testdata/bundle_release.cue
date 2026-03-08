// Minimal BundleRelease fixture for unit testing.
// Used to verify that BundleRelease files are rejected by render commands.
{
	kind: "BundleRelease"
	metadata: {
		name:      "my-bundle"
		namespace: "default"
		version:   "1.0.0"
	}
	releases: [
		{
			module: "example.com/modules/frontend:0.2.0"
			values: {replicas: 2}
		},
		{
			module: "example.com/modules/backend:0.3.0"
			values: {replicas: 1}
		},
	]
}
