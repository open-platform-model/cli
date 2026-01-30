// Package testutil provides test helpers for CLI tests.
package testutil

import (
	"os"
	"path/filepath"
	"testing"
)

// TempDir creates a temporary directory for tests and returns a cleanup function.
func TempDir(t *testing.T) (string, func()) {
	t.Helper()
	dir, err := os.MkdirTemp("", "opm-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	return dir, func() {
		if err := os.RemoveAll(dir); err != nil {
			t.Logf("warning: failed to remove temp dir %s: %v", dir, err)
		}
	}
}

// FixturePath returns the absolute path to a test fixture.
func FixturePath(t *testing.T, parts ...string) string {
	t.Helper()
	// Find the cli directory by walking up from the test
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}

	// Walk up to find cli/tests/fixtures
	dir := wd
	for {
		fixturesPath := filepath.Join(dir, "tests", "fixtures")
		if _, err := os.Stat(fixturesPath); err == nil {
			return filepath.Join(append([]string{fixturesPath}, parts...)...)
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("could not find tests/fixtures directory from %s", wd)
		}
		dir = parent
	}
}

// WriteFile creates a file with the given content in the specified directory.
func WriteFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("failed to create parent dirs for %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write file %s: %v", path, err)
	}
	return path
}

// CopyFixture copies a fixture directory to a temporary location.
func CopyFixture(t *testing.T, fixtureName string) string {
	t.Helper()
	src := FixturePath(t, fixtureName)
	dst, cleanup := TempDir(t)
	t.Cleanup(cleanup)

	if err := copyDir(src, dst); err != nil {
		t.Fatalf("failed to copy fixture %s: %v", fixtureName, err)
	}
	return dst
}

func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		dstPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(dstPath, info.Mode())
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(dstPath, data, info.Mode())
	})
}
