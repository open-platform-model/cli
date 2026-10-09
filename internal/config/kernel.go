package config

import (
	"fmt"
	"os"

	"github.com/open-platform-model/library/opm/kernel"
)

// NewKernel is the CLI's single kernel construction site: one Kernel per
// command invocation, carrying the resolved registry mapping. The mapping
// enters the kernel exactly once, here; module acquisition reads it from
// the kernel, and the library seeds the kernel's schema loader from the same
// value (kernel.New), so the core schema resolves through the same mapping
// without a separate schema-loader option. An empty registry falls back to
// the process environment (CUE_REGISTRY).
func NewKernel(registry string) *kernel.Kernel {
	return kernel.New(kernel.WithRegistry(registry))
}

// RegistryConfigured reports whether anything names a registry for the
// kernel NewKernel(registry) builds: the resolved registry (--registry, then
// OPM_REGISTRY, then the config file), or CUE_REGISTRY in the process
// environment, which the kernel falls back to when the resolved one is empty.
// When it is false, CUE resolves every module from its central registry,
// which does not hold opmodel.dev.
func RegistryConfigured(registry string) bool {
	return registry != "" || os.Getenv("CUE_REGISTRY") != ""
}

// NoRegistryError reports a registry operation that failed while no registry
// was configured (RegistryConfigured is false). It names the missing
// configuration as the cause and the way to supply it; the operation's own
// error stays wrapped and in the message.
type NoRegistryError struct {
	// Op names the operation that failed.
	Op string
	// ConfigPath is the resolved config file path, "" when unknown.
	ConfigPath string
	// ConfigExists is true when a file exists at ConfigPath: opm config init
	// then refuses without --force.
	ConfigExists bool
	Err          error
}

// NewNoRegistryError builds the error for a failed op, reading whether the
// config file at configPath exists.
func NewNoRegistryError(op, configPath string, err error) *NoRegistryError {
	e := &NoRegistryError{Op: op, ConfigPath: configPath, Err: err}
	if configPath != "" {
		_, statErr := os.Stat(configPath)
		e.ConfigExists = statErr == nil
	}
	return e
}

func (e *NoRegistryError) Error() string {
	file := "the config file"
	if e.ConfigPath != "" {
		file = e.ConfigPath
	}
	next := "To write a config file with the default registry, run:  opm config init"
	if e.ConfigExists {
		next = "The config file exists and sets none. Add a registry field to it, or replace the whole file\n  with the default one (this drops its other settings):  opm config init --force"
	}
	return fmt.Sprintf("no registry is configured: %s: %v\n  Set one with --registry, OPM_REGISTRY or the registry field of %s.\n  %s",
		e.Op, e.Err, file, next)
}

func (e *NoRegistryError) Unwrap() error { return e.Err }
