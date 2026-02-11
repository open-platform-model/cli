package output

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// SplitOptions controls split file output.
type SplitOptions struct {
	// OutDir is the directory for split output
	OutDir string
	// Format specifies output format: "yaml" or "json"
	Format Format
}

// WriteSplitManifests writes each resource to a separate file.
// Files are named <lowercase-kind>-<resource-name>.<ext>
func WriteSplitManifests(resources []ResourceInfo, opts SplitOptions) error {
	if len(resources) == 0 {
		return nil
	}

	// Ensure output directory exists
	if err := os.MkdirAll(opts.OutDir, 0o755); err != nil {
		return fmt.Errorf("creating output directory: %w", err)
	}

	// Track filenames to handle collisions
	usedNames := make(map[string]int)

	for _, res := range resources {
		filename := buildFilenameFromInfo(res, opts.Format, usedNames)
		path := filepath.Join(opts.OutDir, filename)

		if err := writeResourceFileFromInfo(res, path, opts.Format); err != nil {
			return fmt.Errorf("writing %s: %w", path, err)
		}

		Debug("wrote resource file",
			"kind", res.GetKind(),
			"name", res.GetName(),
			"file", path,
		)
	}

	return nil
}

// buildFilenameFromInfo creates a filename for a resource.
func buildFilenameFromInfo(res ResourceInfo, format Format, usedNames map[string]int) string {
	ext := ".yaml"
	if format == FormatJSON {
		ext = ".json"
	}

	kind := strings.ToLower(res.GetKind())
	name := sanitizeName(res.GetName())
	baseName := kind + "-" + name

	count, exists := usedNames[baseName]
	if exists {
		usedNames[baseName] = count + 1
		return fmt.Sprintf("%s-%d%s", baseName, count+1, ext)
	}

	usedNames[baseName] = 1
	return baseName + ext
}

// buildFilenameFromUnstructured creates a filename for an unstructured resource.

// sanitizeName makes a name safe for use in filenames.
func sanitizeName(name string) string {
	replacer := strings.NewReplacer(
		"/", "-",
		"\\", "-",
		":", "-",
		"*", "-",
		"?", "-",
		"\"", "",
		"<", "",
		">", "",
		"|", "-",
	)
	return replacer.Replace(name)
}

// writeResourceFileFromInfo writes a single resource to a file.
func writeResourceFileFromInfo(res ResourceInfo, destPath string, format Format) error {
	f, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer f.Close()

	return writeResource(res.GetObject(), format, f)
}

// writeResourceFile writes a single unstructured resource to a file.
