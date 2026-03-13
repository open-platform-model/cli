package releasefile

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
	KindModuleRelease Kind = "ModuleRelease"
	KindBundleRelease Kind = "BundleRelease"
)

type FileRelease struct {
	Path   string
	Kind   Kind
	Module *module.Release
	Bundle *bundle.Release
}

func GetReleaseFile(ctx *cue.Context, filePath string) (*FileRelease, error) {
	val, _, err := loader.LoadReleaseFile(ctx, filePath, loader.LoadOptions{Registry: os.Getenv("CUE_REGISTRY")})
	if err != nil {
		return nil, err
	}

	kind, err := loader.DetectReleaseKind(val)
	if err != nil {
		return nil, err
	}

	switch kind {
	case string(KindModuleRelease):
		moduleRelease, err := bareModuleRelease(val, filePath)
		if err != nil {
			return nil, err
		}
		return &FileRelease{
			Path:   filePath,
			Kind:   KindModuleRelease,
			Module: moduleRelease,
		}, nil
	case string(KindBundleRelease):
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
		return nil, fmt.Errorf("unsupported release kind %q", kind)
	}
}

func bareModuleRelease(v cue.Value, fallbackName string) (*module.Release, error) {
	moduleVal := v.LookupPath(cue.ParsePath("#module"))
	moduleConfig := v.LookupPath(cue.ParsePath("#module.#config"))
	releaseMeta, err := mustModuleReleaseMetadata(v, fallbackName)
	if err != nil {
		return nil, err
	}

	return module.NewRelease(
		releaseMeta,
		module.Module{
			Metadata: bestEffortModuleMetadata(moduleVal),
			Config:   moduleConfig,
			Raw:      moduleVal,
		},
		v,
		cue.Value{},
		moduleConfig,
		cue.Value{},
	), nil
}

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
		RawCUE:   v,
		Releases: map[string]*module.Release{},
		Config:   bundleConfig,
	}, nil
}

func mustModuleReleaseMetadata(v cue.Value, fallbackName string) (*module.ReleaseMetadata, error) {
	metaVal := v.LookupPath(cue.ParsePath("metadata"))
	if !metaVal.Exists() {
		return nil, fmt.Errorf("module release metadata is required for %q", fallbackName)
	}
	if err := metaVal.Validate(cue.Concrete(true)); err != nil {
		return nil, fmt.Errorf("module release metadata must be concrete for %q: %w", fallbackName, err)
	}
	meta := &module.ReleaseMetadata{}
	if err := metaVal.Decode(meta); err != nil {
		return nil, fmt.Errorf("decoding module release metadata for %q: %w", fallbackName, err)
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
