// Package release provides release building functionality for OPM modules.
package release

// Options configures release building.
type Options struct {
	Name      string // Required: release name
	Namespace string // Required: target namespace
}
