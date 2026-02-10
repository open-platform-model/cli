// Package identity provides constants for OPM module identity computation.
package identity

// OPMNamespaceUUID is the UUID v5 namespace for computing deterministic identities.
// This documents the namespace used by the CUE catalog schemas for identity computation.
// Computed as: uuid.SHA1(uuid.NameSpaceDNS, "opmodel.dev")
//
// Note: Release identities are computed by the CUE catalog schemas and extracted
// from metadata.labels during build. Go does not compute UUIDs directly.
const OPMNamespaceUUID = "c1cbe76d-5687-5a47-bfe6-83b081b15413"
