package instinit

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"slices"

	"github.com/open-platform-model/cli/internal/cuemod"
	"github.com/open-platform-model/cli/internal/publish"
)

// stagingPrefix names the sibling directory a package is staged in.
const stagingPrefix = ".opm-instance-init-"

// writeOps are the side effects Write depends on, injectable for tests.
type writeOps struct {
	tidy   func(ctx context.Context, dir string, opts cuemod.TidyOptions) (cuemod.TidyResult, error)
	rename func(from, to string) error
}

var defaultOps = writeOps{tidy: cuemod.Tidy, rename: os.Rename}

// Write stages files in a sibling directory of dir, completes the staged
// module's dependency closure with cuemod.Tidy against registry, and renames
// the result onto dir. On any error nothing remains at dir or beside it.
//
// A registry that cannot be reached while tidy resolves the closure comes
// back as a *publish.ConnectivityError; every other failure, a registry
// answer that a dependency is not held among them, is wrapped as
// "initializing <dir>". cuemod.IsConnectivityError takes that decision from
// the library's classification of the flattened tidy error, not from its
// text.
func Write(ctx context.Context, dir string, files Files, registry string) error {
	return write(ctx, dir, files, registry, defaultOps)
}

func write(ctx context.Context, dir string, files Files, registry string, ops writeOps) (err error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return fmt.Errorf("initializing %s: %w", dir, err)
	}
	staging, err := os.MkdirTemp(filepath.Dir(abs), stagingPrefix+filepath.Base(abs)+"-")
	if err != nil {
		return fmt.Errorf("initializing %s: %w", dir, err)
	}
	defer func() {
		if err != nil {
			if rmErr := os.RemoveAll(staging); rmErr != nil {
				err = errors.Join(err, fmt.Errorf("removing %s: %w", staging, rmErr))
			}
		}
	}()
	// MkdirTemp creates 0o700; the package is a project tree, not a secret.
	if err := os.Chmod(staging, 0o755); err != nil {
		return fmt.Errorf("initializing %s: %w", dir, err)
	}

	if err := writeFiles(staging, files); err != nil {
		return fmt.Errorf("initializing %s: %w", dir, err)
	}

	if _, err := ops.tidy(ctx, staging, cuemod.TidyOptions{Registry: registry}); err != nil {
		if cuemod.IsConnectivityError(err) {
			return &publish.ConnectivityError{
				Op:  fmt.Sprintf("resolving the dependencies of %s (registry %s)", dir, registry),
				Err: err,
			}
		}
		return fmt.Errorf("initializing %s: resolving dependencies: %w", dir, err)
	}

	if _, statErr := os.Lstat(abs); !errors.Is(statErr, fs.ErrNotExist) {
		return fmt.Errorf("initializing %s: the directory appeared while initializing", dir)
	}
	if err := ops.rename(staging, abs); err != nil {
		return fmt.Errorf("initializing %s: %w", dir, err)
	}
	return nil
}

// writeFiles writes every file under root in path order.
func writeFiles(root string, files Files) error {
	for _, name := range slices.Sorted(maps.Keys(files)) {
		target := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(target, files[name], 0o644); err != nil { //nolint:gosec // G306: package sources are project files, not secrets
			return err
		}
	}
	return nil
}
