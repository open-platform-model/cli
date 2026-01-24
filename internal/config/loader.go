package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/viper"
)

// Environment variable prefix for OPM configuration.
const envPrefix = "OPM"

// Loader handles loading and merging configuration from multiple sources.
type Loader struct {
	v *viper.Viper
}

// NewLoader creates a new configuration loader.
func NewLoader() *Loader {
	v := viper.New()

	// Set up environment variable bindings
	v.SetEnvPrefix(envPrefix)
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Bind specific environment variables
	_ = v.BindEnv("kubeconfig", "OPM_KUBECONFIG")
	_ = v.BindEnv("context", "OPM_CONTEXT")
	_ = v.BindEnv("namespace", "OPM_NAMESPACE")
	_ = v.BindEnv("registry", "OPM_REGISTRY")
	_ = v.BindEnv("cacheDir", "OPM_CACHE_DIR")

	return &Loader{v: v}
}

// Load loads configuration from the given file path.
// If configFile is empty, it uses the default config file path.
// Environment variables take precedence over file values.
func (l *Loader) Load(configFile string) (*Config, error) {
	if configFile == "" {
		var err error
		configFile, err = GetConfigFile()
		if err != nil {
			return nil, fmt.Errorf("getting config file path: %w", err)
		}
	}

	// Expand ~ in path
	expandedPath, err := ExpandPath(configFile)
	if err != nil {
		return nil, fmt.Errorf("expanding config path: %w", err)
	}

	// Set up viper for the config file
	l.v.SetConfigFile(expandedPath)
	l.v.SetConfigType("yaml")

	// Try to read config file (not an error if it doesn't exist)
	if err := l.v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			// Only return error if it's not a "file not found" error
			if !os.IsNotExist(err) {
				return nil, fmt.Errorf("reading config file: %w", err)
			}
		}
		// Config file not found is OK, we'll use defaults + env vars
	}

	// Unmarshal into Config struct
	var cfg Config
	if err := l.v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshaling config: %w", err)
	}

	return &cfg, nil
}

// LoadWithDefaults loads configuration and applies defaults.
func (l *Loader) LoadWithDefaults(configFile string) (*Config, error) {
	cfg, err := l.Load(configFile)
	if err != nil {
		return nil, err
	}

	return cfg.WithDefaults(), nil
}

// LoadFromEnvOnly loads configuration from environment variables only.
func (l *Loader) LoadFromEnvOnly() (*Config, error) {
	cfg := &Config{
		Kubeconfig: os.Getenv("OPM_KUBECONFIG"),
		Context:    os.Getenv("OPM_CONTEXT"),
		Namespace:  os.Getenv("OPM_NAMESPACE"),
		Registry:   os.Getenv("OPM_REGISTRY"),
		CacheDir:   os.Getenv("OPM_CACHE_DIR"),
	}

	return cfg.WithDefaults(), nil
}

// ConfigFileExists checks if the config file exists.
func ConfigFileExists(configFile string) (bool, error) {
	if configFile == "" {
		var err error
		configFile, err = GetConfigFile()
		if err != nil {
			return false, err
		}
	}

	expandedPath, err := ExpandPath(configFile)
	if err != nil {
		return false, err
	}

	_, err = os.Stat(expandedPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}

	return true, nil
}
