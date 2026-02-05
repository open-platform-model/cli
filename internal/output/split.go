package output

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// SplitOptions controls split file output.
type SplitOptions struct {
	// OutDir is the directory for split output
	OutDir string
	// Format specifies output format: "yaml" or "json"
	Format OutputFormat
}

// WriteSplitManifests writes each resource to a separate file.
// Files are named <lowercase-kind>-<resource-name>.<ext>
func WriteSplitManifests(resources []ResourceInfo, opts SplitOptions) error {
	if len(resources) == 0 {
		return nil
	}

	// Ensure output directory exists
	if err := os.MkdirAll(opts.OutDir, 0755); err != nil {
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

// WriteSplitUnstructured writes each unstructured resource to a separate file.
func WriteSplitUnstructured(objects []*unstructured.Unstructured, opts SplitOptions) error {
	if len(objects) == 0 {
		return nil
	}

	// Ensure output directory exists
	if err := os.MkdirAll(opts.OutDir, 0755); err != nil {
		return fmt.Errorf("creating output directory: %w", err)
	}

	// Track filenames to handle collisions
	usedNames := make(map[string]int)

	for _, obj := range objects {
		filename := buildFilenameFromUnstructured(obj, opts.Format, usedNames)
		path := filepath.Join(opts.OutDir, filename)

		if err := writeResourceFile(obj, path, opts.Format); err != nil {
			return fmt.Errorf("writing %s: %w", path, err)
		}

		Debug("wrote resource file",
			"kind", obj.GetKind(),
			"name", obj.GetName(),
			"file", path,
		)
	}

	return nil
}

// buildFilenameFromInfo creates a filename for a resource.
func buildFilenameFromInfo(res ResourceInfo, format OutputFormat, usedNames map[string]int) string {
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
func buildFilenameFromUnstructured(obj *unstructured.Unstructured, format OutputFormat, usedNames map[string]int) string {
	ext := ".yaml"
	if format == FormatJSON {
		ext = ".json"
	}

	kind := strings.ToLower(obj.GetKind())
	name := sanitizeName(obj.GetName())
	baseName := kind + "-" + name

	count, exists := usedNames[baseName]
	if exists {
		usedNames[baseName] = count + 1
		return fmt.Sprintf("%s-%d%s", baseName, count+1, ext)
	}

	usedNames[baseName] = 1
	return baseName + ext
}

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
func writeResourceFileFromInfo(res ResourceInfo, filepath string, format OutputFormat) error {
	f, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer f.Close()

	return WriteResource(res.GetObject(), format, f)
}

// writeResourceFile writes a single unstructured resource to a file.
func writeResourceFile(obj *unstructured.Unstructured, filepath string, format OutputFormat) error {
	f, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer f.Close()

	return WriteResource(obj, format, f)
}
