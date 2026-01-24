package cue

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/opmodel/cli/internal/version"
)

var (
	// ErrCUENotFound is returned when the CUE binary is not found.
	ErrCUENotFound = errors.New("cue binary not found")
	// ErrCUEVersionMismatch is returned when the CUE binary version is incompatible.
	ErrCUEVersionMismatch = errors.New("cue binary version mismatch")
)

// Binary wraps calls to the external CUE binary.
type Binary struct {
	// Path is the path to the CUE binary. If empty, "cue" is used from PATH.
	Path string

	// Stdout for CUE command output. If nil, os.Stdout is used.
	Stdout io.Writer

	// Stderr for CUE command errors. If nil, os.Stderr is used.
	Stderr io.Writer
}

// NewBinary creates a new Binary wrapper using "cue" from PATH.
func NewBinary() *Binary {
	return &Binary{
		Path:   "cue",
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}
}

// CheckVersion verifies the CUE binary version is compatible with the SDK.
func (b *Binary) CheckVersion(ctx context.Context) error {
	info := version.DetectCUEBinary()

	if !info.Found {
		return ErrCUENotFound
	}

	if !info.Compatible {
		return fmt.Errorf("%w: SDK version %s, binary version %s",
			ErrCUEVersionMismatch, version.CUESDKVersion, info.Version)
	}

	return nil
}

// Vet runs `cue vet` on the specified directory.
func (b *Binary) Vet(ctx context.Context, dir string, concrete bool) error {
	if err := b.CheckVersion(ctx); err != nil {
		return err
	}

	args := []string{"vet", "./..."}
	if concrete {
		args = append(args, "--concrete")
	}

	return b.run(ctx, dir, args...)
}

// Tidy runs `cue mod tidy` on the specified directory.
func (b *Binary) Tidy(ctx context.Context, dir string) error {
	if err := b.CheckVersion(ctx); err != nil {
		return err
	}

	return b.run(ctx, dir, "mod", "tidy")
}

// Fmt runs `cue fmt` on the specified directory.
func (b *Binary) Fmt(ctx context.Context, dir string) error {
	if err := b.CheckVersion(ctx); err != nil {
		return err
	}

	return b.run(ctx, dir, "fmt", "./...")
}

// Eval runs `cue eval` on the specified directory and returns the output.
func (b *Binary) Eval(ctx context.Context, dir string, expr string, format string) ([]byte, error) {
	if err := b.CheckVersion(ctx); err != nil {
		return nil, err
	}

	args := []string{"eval"}
	if expr != "" {
		args = append(args, "-e", expr)
	}
	if format != "" {
		args = append(args, "--out", format)
	}
	args = append(args, "./...")

	return b.runCapture(ctx, dir, args...)
}

// Export runs `cue export` on the specified directory and returns the output.
func (b *Binary) Export(ctx context.Context, dir string, format string) ([]byte, error) {
	if err := b.CheckVersion(ctx); err != nil {
		return nil, err
	}

	args := []string{"export"}
	if format != "" {
		args = append(args, "--out", format)
	}
	args = append(args, "./...")

	return b.runCapture(ctx, dir, args...)
}

// run executes a CUE command in the specified directory.
func (b *Binary) run(ctx context.Context, dir string, args ...string) error {
	cmd := exec.CommandContext(ctx, b.path(), args...)
	cmd.Dir = dir
	cmd.Stdout = b.stdout()
	cmd.Stderr = b.stderr()

	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return fmt.Errorf("cue %s failed with exit code %d",
				strings.Join(args, " "), exitErr.ExitCode())
		}
		return fmt.Errorf("cue %s: %w", strings.Join(args, " "), err)
	}

	return nil
}

// runCapture executes a CUE command and captures its output.
func (b *Binary) runCapture(ctx context.Context, dir string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, b.path(), args...)
	cmd.Dir = dir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("cue %s failed with exit code %d: %s",
				strings.Join(args, " "), exitErr.ExitCode(), stderr.String())
		}
		return nil, fmt.Errorf("cue %s: %w", strings.Join(args, " "), err)
	}

	return stdout.Bytes(), nil
}

func (b *Binary) path() string {
	if b.Path != "" {
		return b.Path
	}
	return "cue"
}

func (b *Binary) stdout() io.Writer {
	if b.Stdout != nil {
		return b.Stdout
	}
	return os.Stdout
}

func (b *Binary) stderr() io.Writer {
	if b.Stderr != nil {
		return b.Stderr
	}
	return os.Stderr
}
