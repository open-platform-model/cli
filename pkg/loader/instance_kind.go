package loader

import (
	"fmt"
	"os"
	"path/filepath"

	"cuelang.org/go/cue"
)

// kindModuleInstance is the core@v1 module-instance wire kind (was
// "ModuleInstance", enhancement 0002 D-X1.1).
const kindModuleInstance = "ModuleInstance"

// DetectInstanceKind returns the kind field from an evaluated instance value.
//
// Was: DetectReleaseKind (enhancement 0002 D8 hard-rename).
func DetectInstanceKind(v cue.Value) (string, error) {
	kindVal := v.LookupPath(cue.ParsePath("kind"))
	if !kindVal.Exists() {
		return "", fmt.Errorf("no 'kind' field in instance value")
	}
	kind, err := kindVal.String()
	if err != nil {
		return "", fmt.Errorf("decoding instance kind: %w", err)
	}
	switch kind {
	// "ModuleInstance" is the core@v1 wire kind (was "ModuleRelease", 0002 D-X1.1).
	// The bundle path was removed in 0002 X2 (D15); "BundleRelease" is no longer
	// recognized and falls through to the unknown-kind error.
	case kindModuleInstance:
		return kind, nil
	default:
		return "", fmt.Errorf("unknown instance kind: %q", kind)
	}
}

// resolveInstanceFile resolves either an instance directory or direct file path.
// Inside a directory the loader requires instance.cue (was release.cue, 0002 D9);
// instance.cue is not accepted as a fallback (D8 hard-rename, no alias).
//
// Was: resolveReleaseFile (enhancement 0002 D8 hard-rename).
func resolveInstanceFile(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("instance path must not be empty")
	}
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("instance path %q not found", path)
		}
		return "", fmt.Errorf("stat instance path: %w", err)
	}
	if info.IsDir() {
		instancePath := filepath.Join(path, "instance.cue")
		if _, err := os.Stat(instancePath); err != nil {
			if os.IsNotExist(err) {
				return "", fmt.Errorf("instance path %q does not contain instance.cue", path)
			}
			return "", fmt.Errorf("stat instance file: %w", err)
		}
		return instancePath, nil
	}
	return path, nil
}
