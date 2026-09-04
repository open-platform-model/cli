// Package config provides configuration loading and management.
package config

import (
	_ "embed"
)

//go:embed schema/config.cue
var configSchemaCUE []byte

// platformSchemaCUE is the legacy data-only platform file projection, used
// by LoadLegacyPlatformFile only (deleted by cli-render-switch).
//
//go:embed schema/platform.cue
var platformSchemaCUE []byte
