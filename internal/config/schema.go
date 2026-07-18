// Package config provides configuration loading and management.
package config

import (
	_ "embed"
)

//go:embed schema/config.cue
var configSchemaCUE []byte

//go:embed schema/platform.cue
var platformSchemaCUE []byte
