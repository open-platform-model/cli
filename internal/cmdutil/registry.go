package cmdutil

import (
	"github.com/open-platform-model/cli/internal/config"
	opmexit "github.com/open-platform-model/cli/internal/exit"
)

// NoRegistryError returns the exit error for a registry operation that
// failed (err) while no registry is configured, and nil when one is: no
// --registry, no OPM_REGISTRY, no registry in the config file and no
// CUE_REGISTRY. The missing configuration is then the cause to name, not the
// registry's answer or its silence, so it exits 2 like the refusal
// `opm registry login` gives for the same condition, with the way to
// configure one.
//
// It is asked only after the operation failed, never before it: a run with
// nothing configured can still succeed from a warm CUE module cache.
func NoRegistryError(cfg *config.GlobalConfig, op string, err error) error {
	if err == nil || cfg == nil || config.RegistryConfigured(cfg.Registry) {
		return nil
	}
	return &opmexit.ExitError{
		Code: opmexit.ExitValidationError,
		Err:  config.NewNoRegistryError(op, cfg.ConfigPath, err),
	}
}
