package loader

import (
	"fmt"
	"os"
	"path/filepath"

	"cuelang.org/go/cue"
)

// DetectReleaseKind returns the kind field from an evaluated release value.
func DetectReleaseKind(v cue.Value) (string, error) {
	kindVal := v.LookupPath(cue.ParsePath("kind"))
	if !kindVal.Exists() {
		return "", fmt.Errorf("no 'kind' field in release value")
	}
	kind, err := kindVal.String()
	if err != nil {
		return "", fmt.Errorf("decoding release kind: %w", err)
	}
	switch kind {
	case "ModuleRelease", "BundleRelease":
		return kind, nil
	default:
		return "", fmt.Errorf("unknown release kind: %q", kind)
	}
}

// resolveReleaseFile resolves either a release directory or direct file path.
func resolveReleaseFile(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("release path must not be empty")
	}
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("release path %q not found", path)
		}
		return "", fmt.Errorf("stat release path: %w", err)
	}
	if info.IsDir() {
		releasePath := filepath.Join(path, "release.cue")
		if _, err := os.Stat(releasePath); err != nil {
			if os.IsNotExist(err) {
				return "", fmt.Errorf("release path %q does not contain release.cue", path)
			}
			return "", fmt.Errorf("stat release file: %w", err)
		}
		return releasePath, nil
	}
	return path, nil
}
