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

	// CacheDir is the path to the cache directory (~/.opm/cache).
	CacheDir string

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
		CacheDir:   filepath.Join(opmHome, "cache"),
		HomeDir:    opmHome,
	}, nil
}

// PathsFromEnv returns paths considering environment overrides.
func PathsFromEnv() (*Paths, error) {
	paths, err := DefaultPaths()
	if err != nil {
		return nil, err
	}

	// Check for OPM_CONFIG override
	if configPath := os.Getenv("OPM_CONFIG"); configPath != "" {
		paths.ConfigFile = configPath
	}

	// Check for OPM_CACHE_DIR override
	if cacheDir := os.Getenv("OPM_CACHE_DIR"); cacheDir != "" {
		paths.CacheDir = cacheDir
	}

	return paths, nil
}

// ExpandPath expands ~ to the user's home directory.
func ExpandPath(path string) (string, error) {
	if len(path) == 0 {
		return path, nil
	}

	if path[0] != '~' {
		return path, nil
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	if len(path) == 1 {
		return homeDir, nil
	}

	return filepath.Join(homeDir, path[1:]), nil
}

// EnsureDir ensures a directory exists with the given permissions.
func EnsureDir(path string, perm os.FileMode) error {
	return os.MkdirAll(path, perm)
}
