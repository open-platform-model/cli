package valuesloadisolation

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// modulePath returns the absolute path to the test module fixture.
// The module has three conflicting values files (values.cue, values_forge.cue,
// values_testing.cue) to reproduce the production bug.
func modulePath(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not determine test file path")
	}
	return filepath.Join(filepath.Dir(file), "testdata", "module")
}

// externalValuesPath returns the path to the external values file that lives
// outside the module directory, simulating a --values / -f flag argument.
func externalValuesPath(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not determine test file path")
	}
	return filepath.Join(filepath.Dir(file), "testdata", "external_values.cue")
}

// isValuesFile reports whether a filename matches the values*.cue pattern —
// any .cue file whose base name starts with "values".
func isValuesFile(name string) bool {
	base := filepath.Base(name)
	return strings.HasPrefix(base, "values") && strings.HasSuffix(base, ".cue")
}

// extractPackageName scans a .cue file line by line and returns the package
// name from the first "package <name>" declaration found.
// Returns an error if no package declaration is found.
func extractPackageName(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("opening file: %w", err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "//") || line == "" {
			continue
		}
		if strings.HasPrefix(line, "package ") {
			parts := strings.Fields(line)
			if len(parts) == 2 {
				return parts[1], nil
			}
		}
		// Stop after the first non-comment, non-blank, non-package line.
		break
	}
	return "", fmt.Errorf("no package declaration found in %s", path)
}

// cueFilesInDir returns all .cue files in dir (non-recursive, excluding cue.mod/).
func cueFilesInDir(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)

	var files []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if strings.HasSuffix(e.Name(), ".cue") {
			files = append(files, filepath.Join(dir, e.Name()))
		}
	}
	return files
}
