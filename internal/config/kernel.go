package config

import "github.com/open-platform-model/library/opm/kernel"

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
