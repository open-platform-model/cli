// Package instancefile detects and loads an on-disk instance file
// (a #ModuleInstance) into bare parse data.
//
// Was: package releasefile (enhancement 0002 D10 — package name carried the
// `release` token). The bundle path was removed in 0002 X2 (D15, supersedes
// D7): bundle support was unreachable dead code, so KindBundleRelease,
// *bundle.Instance, and the bare/must bundle helpers are gone.
package instancefile

import (
	"fmt"
	"os"

	"cuelang.org/go/cue"

	"github.com/opmodel/cli/pkg/loader"
	"github.com/opmodel/cli/pkg/module"
)

type Kind string

const (
	// KindModuleInstance was KindModuleInstance; value flips to the core@v1 wire
	// string "ModuleInstance" (enhancement 0002 D-X1.1).
	KindModuleInstance Kind = "ModuleInstance"
)

// ModuleParseData holds the raw parse data from a module instance file.
// This is the pre-preparation state — values have not been validated or filled.
// Use module.ParseModuleInstance to construct a fully prepared *module.Instance.
type ModuleParseData struct {
	// Spec is the raw instance spec CUE value (before values filling).
	Spec cue.Value

	// Module is the best-effort module info extracted from #module.
	Module module.Module

	// Metadata is the best-effort instance metadata decoded from the spec.
	// Available for early display and error messages before values are applied.
	Metadata *module.InstanceMetadata
}

// FileInstance is the container returned by GetInstanceFile: it holds a
// module-instance parse-data. The struct name is kept verbatim (it doubles as
// an X3 workflow surface).
type FileInstance struct {
	Path   string
	Kind   Kind
	Module *ModuleParseData
}

func GetInstanceFile(ctx *cue.Context, filePath string) (*FileInstance, error) {
	val, _, err := loader.LoadInstanceFile(ctx, filePath, loader.LoadOptions{Registry: os.Getenv("CUE_REGISTRY")})
	if err != nil {
		return nil, err
	}

	kind, err := loader.DetectInstanceKind(val)
	if err != nil {
		return nil, err
	}

	switch kind {
	case string(KindModuleInstance):
		parseData, err := bareModuleInstance(val, filePath)
		if err != nil {
			return nil, err
		}
		return &FileInstance{
			Path:   filePath,
			Kind:   KindModuleInstance,
			Module: parseData,
		}, nil
	default:
		// Defensive: DetectInstanceKind already rejects non-ModuleInstance kinds,
		// so this is unreachable today. Kept as a guard with wording aligned to
		// DetectInstanceKind's "unknown instance kind" for a single consistent message.
		return nil, fmt.Errorf("unknown instance kind: %q", kind)
	}
}

func bareModuleInstance(v cue.Value, fallbackName string) (*ModuleParseData, error) {
	moduleVal := v.LookupPath(cue.ParsePath("#module"))
	moduleConfig := v.LookupPath(cue.ParsePath("#module.#config"))
	instanceMeta, err := mustModuleInstanceMetadata(v, fallbackName)
	if err != nil {
		return nil, err
	}

	return &ModuleParseData{
		Spec: v,
		Module: module.Module{
			Metadata: bestEffortModuleMetadata(moduleVal),
			Config:   moduleConfig,
			Raw:      moduleVal,
		},
		Metadata: instanceMeta,
	}, nil
}

func mustModuleInstanceMetadata(v cue.Value, fallbackName string) (*module.InstanceMetadata, error) {
	metaVal := v.LookupPath(cue.ParsePath("metadata"))
	if !metaVal.Exists() {
		return nil, fmt.Errorf("module instance metadata is required for %q", fallbackName)
	}
	if err := metaVal.Validate(cue.Concrete(true)); err != nil {
		return nil, fmt.Errorf("module instance metadata must be concrete for %q: %w", fallbackName, err)
	}
	meta := &module.InstanceMetadata{}
	if err := metaVal.Decode(meta); err != nil {
		return nil, fmt.Errorf("decoding module instance metadata for %q: %w", fallbackName, err)
	}
	return meta, nil
}

func bestEffortModuleMetadata(v cue.Value) *module.ModuleMetadata {
	meta := &module.ModuleMetadata{}
	if mv := v.LookupPath(cue.ParsePath("metadata")); mv.Exists() {
		if err := mv.Decode(meta); err != nil {
			return meta
		}
	}
	return meta
}
