package astpipeline

import (
	"path/filepath"
	"runtime"
	"testing"
)

// testModulePath returns the absolute path to the test module fixture.
func testModulePath(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("unable to determine test file path")
	}
	return filepath.Join(filepath.Dir(filename), "testdata", "test-module")
}
