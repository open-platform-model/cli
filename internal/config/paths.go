package config

import (
	"os"
	"path/filepath"
)

// Paths contains standard filesystem paths for OPM.
type Paths struct {
	// ConfigFile is the path to the config file (~/.opm/config.yaml).
	ConfigFile string

	// CacheDir is the path to the cache directory (~/.opm/cache).
	CacheDir string

	// HomeDir is the OPM home directory (~/.opm).
	HomeDir string
}

// DefaultPaths returns the default paths for OPM.
func DefaultPaths() (*Paths, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	opmHome := filepath.Join(homeDir, ".opm")

	return &Paths{
		ConfigFile: filepath.Join(opmHome, "config.yaml"),
		CacheDir:   filepath.Join(opmHome, "cache"),
		HomeDir:    opmHome,
	}, nil
}

// GetConfigFile returns the config file path.
// If OPM_CONFIG is set, it takes precedence.
func GetConfigFile() (string, error) {
	if envPath := os.Getenv("OPM_CONFIG"); envPath != "" {
		return envPath, nil
	}

	paths, err := DefaultPaths()
	if err != nil {
		return "", err
	}

	return paths.ConfigFile, nil
}

// GetCacheDir returns the cache directory path.
// If OPM_CACHE_DIR is set, it takes precedence.
func GetCacheDir() (string, error) {
	if envPath := os.Getenv("OPM_CACHE_DIR"); envPath != "" {
		return envPath, nil
	}

	paths, err := DefaultPaths()
	if err != nil {
		return "", err
	}

	return paths.CacheDir, nil
}

// GetHomeDir returns the OPM home directory path.
func GetHomeDir() (string, error) {
	paths, err := DefaultPaths()
	if err != nil {
		return "", err
	}

	return paths.HomeDir, nil
}

// EnsureHomeDir creates the OPM home directory if it doesn't exist.
func EnsureHomeDir() error {
	homeDir, err := GetHomeDir()
	if err != nil {
		return err
	}

	return os.MkdirAll(homeDir, 0o755)
}

// EnsureCacheDir creates the cache directory if it doesn't exist.
func EnsureCacheDir() error {
	cacheDir, err := GetCacheDir()
	if err != nil {
		return err
	}

	return os.MkdirAll(cacheDir, 0o755)
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

	// Handle ~/path/to/something
	if path[1] == '/' || path[1] == filepath.Separator {
		return filepath.Join(homeDir, path[2:]), nil
	}

	// Handle ~username (not supported, return as-is)
	return path, nil
}
