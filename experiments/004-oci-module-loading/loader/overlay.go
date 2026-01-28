package loader

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"cuelang.org/go/cue/load"
)

// applyPathOverlays adds local path overlays to the load configuration.
// Files from overlay paths take precedence over registry-fetched modules.
func (l *Loader) applyPathOverlays() error {
	if l.loadCfg.Overlay == nil {
		l.loadCfg.Overlay = make(map[string]load.Source)
	}

	for _, path := range l.cfg.PathOverlays {
		if l.cfg.Verbose {
			fmt.Fprintf(os.Stderr, "[loader] Applying path overlay: %s\n", path)
		}

		absPath, err := filepath.Abs(path)
		if err != nil {
			return fmt.Errorf("failed to resolve path %s: %w", path, err)
		}

		// Check if path exists
		if _, err := os.Stat(absPath); err != nil {
			return fmt.Errorf("overlay path does not exist: %s", absPath)
		}

		// Walk the directory tree and add all .cue files to overlay
		count := 0
		err = filepath.Walk(absPath, func(p string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			// Skip directories
			if info.IsDir() {
				return nil
			}

			// Only process .cue files
			if !strings.HasSuffix(p, ".cue") {
				return nil
			}

			// Read file content
			content, err := os.ReadFile(p)
			if err != nil {
				return fmt.Errorf("failed to read file %s: %w", p, err)
			}

			// Calculate the overlay path relative to the module root
			// This ensures the overlay is applied correctly in the CUE module namespace
			relPath, err := filepath.Rel(absPath, p)
			if err != nil {
				relPath = filepath.Base(p)
			}

			// Add to overlay map
			overlayKey := filepath.Join(absPath, relPath)
			l.loadCfg.Overlay[overlayKey] = load.FromBytes(content)
			count++

			if l.cfg.Verbose {
				fmt.Fprintf(os.Stderr, "[loader]   + %s\n", relPath)
			}

			return nil
		})

		if err != nil {
			return fmt.Errorf("failed to walk overlay path %s: %w", path, err)
		}

		if l.cfg.Verbose {
			fmt.Fprintf(os.Stderr, "[loader] Added %d file(s) from %s\n", count, path)
		}
	}

	return nil
}
