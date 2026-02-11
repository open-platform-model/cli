// Package config provides configuration loading and management.
package config

import (
	"os"
	"path/filepath"
)

// Paths contains standard filesystem paths for the CLI.
type Paths struct {
	// ConfigFile is the path to the config file (~/.opm/config.cue).
	ConfigFile string

	// HomeDir is the path to the OPM home directory (~/.opm).
	HomeDir string
}

// DefaultPaths returns the default paths, expanding ~ to the user's home directory.
func DefaultPaths() (*Paths, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	opmHome := filepath.Join(homeDir, ".opm")
	return &Paths{
		ConfigFile: filepath.Join(opmHome, "config.cue"),
		HomeDir:    opmHome,
	}, nil
}
