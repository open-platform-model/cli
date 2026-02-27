// Values provide concrete configuration for the Jellyfin module.
// These satisfy the #config schema defined in module.cue.
package main

// Concrete default values
values: {
	// PVC size for Jellyfin config/metadata directory.
	// Can grow to 50GB+ for large collections (thumbnails, metadata cache).
	configStorageSize: 1
}
