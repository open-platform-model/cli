// Package instancefile detects and loads an on-disk instance file
// (a #ModuleInstance or #BundleRelease) into bare parse data.
//
// Was: package releasefile (enhancement 0002 D10 — package name carried the
// `release` token). X1 renames the container and the module path; the bundle
// path (KindBundleRelease, *bundle.Release, bare/must bundle helpers) is left
// verbatim and reconciled by X2 in the same atomic PR.
package instancefile

import (
	"fmt"
	"os"

	"cuelang.org/go/cue"

	"github.com/opmodel/cli/pkg/bundle"
	"github.com/opmodel/cli/pkg/loader"
	"github.com/opmodel/cli/pkg/module"
)

type Kind string

const (
	// KindModuleInstance was KindModuleRelease; value flips to the core@v1 wire
	// string "ModuleInstance" (enhancement 0002 D-X1.1).
	KindModuleInstance Kind = "ModuleInstance"
	// KindBundleRelease is left untouched (0002 X2 renames the bundle path).
	KindBundleRelease Kind = "BundleRelease"
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

// FileRelease is the dual-purpose container returned by GetInstanceFile: it
// holds either a module-instance parse-data or a bundle. The struct name is
// kept verbatim (it doubles as an X3 workflow surface); X2 renames the bundle
// path it carries.
type FileRelease struct {
	Path   string
	Kind   Kind
	Module *ModuleParseData
	Bundle *bundle.Release // TODO(0002 X2): rename bundle path
}

func GetInstanceFile(ctx *cue.Context, filePath string) (*FileRelease, error) {
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
		return &FileRelease{
			Path:   filePath,
			Kind:   KindModuleInstance,
			Module: parseData,
		}, nil
	case string(KindBundleRelease): // TODO(0002 X2): KindBundleInstance
		bundleRelease, err := bareBundleRelease(val)
		if err != nil {
			return nil, err
		}
		return &FileRelease{
			Path:   filePath,
			Kind:   KindBundleRelease,
			Bundle: bundleRelease,
		}, nil
	default:
		return nil, fmt.Errorf("unsupported instance kind %q", kind)
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

// bareBundleRelease is left verbatim for X2 (0002). TODO(0002 X2): rename to
// bareBundleInstance and retype to the bundle-instance form.
func bareBundleRelease(v cue.Value) (*bundle.Release, error) {
	bundleVal := v.LookupPath(cue.ParsePath("#bundle"))
	bundleConfig := v.LookupPath(cue.ParsePath("#bundle.#config"))
	releaseMeta, err := mustBundleReleaseMetadata(v)
	if err != nil {
		return nil, err
	}

	return &bundle.Release{
		Metadata: releaseMeta,
		Bundle: bundle.Bundle{
			Metadata: bestEffortBundleMetadata(bundleVal),
			Data:     bundleVal,
		},
		Spec:     v,
		Releases: map[string]*module.Instance{},
		Config:   bundleConfig,
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

// mustBundleReleaseMetadata is left verbatim for X2 (0002).
func mustBundleReleaseMetadata(v cue.Value) (*bundle.ReleaseMetadata, error) {
	metaVal := v.LookupPath(cue.ParsePath("metadata"))
	if !metaVal.Exists() {
		return nil, fmt.Errorf("bundle release metadata is required")
	}
	if err := metaVal.Validate(cue.Concrete(true)); err != nil {
		return nil, fmt.Errorf("bundle release metadata must be concrete: %w", err)
	}
	meta := &bundle.ReleaseMetadata{}
	if err := metaVal.Decode(meta); err != nil {
		return nil, fmt.Errorf("decoding bundle release metadata: %w", err)
	}
	return meta, nil
}

func bestEffortBundleMetadata(v cue.Value) *bundle.BundleMetadata {
	meta := &bundle.BundleMetadata{}
	if mv := v.LookupPath(cue.ParsePath("metadata")); mv.Exists() {
		if err := mv.Decode(meta); err != nil {
			return meta
		}
	}
	return meta
}
