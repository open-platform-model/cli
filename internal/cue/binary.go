// Package cue provides CUE SDK integration and binary delegation.
package cue

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	oerrors "github.com/opmodel/cli/internal/errors"
	"github.com/opmodel/cli/internal/output"
	"github.com/opmodel/cli/internal/version"
)

// CheckCUEVersion validates that the CUE binary version is compatible with the SDK.
func CheckCUEVersion() error {
	info := version.GetCUEBinaryInfo()

	if !info.Found {
		return oerrors.NewNotFoundError(
			"CUE binary not found in PATH",
			"",
			"Install CUE from https://cuelang.org/docs/install/",
		)
	}

	if info.Version == "" {
		output.Warn("could not determine CUE binary version, proceeding anyway")
		return nil
	}

	if !info.Compatible {
		return oerrors.NewVersionError(
			fmt.Sprintf("%s.x", extractMajorMinor(version.CUESDKVersion)),
			info.Version,
		)
	}

	output.Debug("CUE version check passed",
		"binary", info.Version,
		"sdk", version.CUESDKVersion,
		"path", info.Path,
	)

	return nil
}

// extractMajorMinor extracts MAJOR.MINOR from a version string.
func extractMajorMinor(v string) string {
	v = strings.TrimPrefix(v, "v")
	parts := strings.Split(v, ".")
	if len(parts) < 2 {
		return v
	}
	return parts[0] + "." + parts[1]
}

// RunCUECommand executes a CUE command with proper environment setup.
func RunCUECommand(dir string, args []string, registry string) error {
	// Find the CUE binary
	cuePath, err := exec.LookPath("cue")
	if err != nil {
		return oerrors.NewNotFoundError(
			"CUE binary not found in PATH",
			"",
			"Install CUE from https://cuelang.org/docs/install/",
		)
	}

	cmd := exec.Command(cuePath, args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Set up environment with CUE_REGISTRY if provided
	env := os.Environ()
	if registry != "" {
		env = append(env, "CUE_REGISTRY="+registry)
		output.Debug("setting CUE_REGISTRY", "registry", registry)
	}
	cmd.Env = env

	output.Debug("running CUE command",
		"cmd", cuePath,
		"args", strings.Join(args, " "),
		"dir", dir,
	)

	if err := cmd.Run(); err != nil {
		// Check if it's an exit error with a non-zero code
		if exitErr, ok := err.(*exec.ExitError); ok {
			// CUE command failed with validation errors
			return oerrors.Wrap(oerrors.ErrValidation,
				fmt.Sprintf("cue %s failed with exit code %d", args[0], exitErr.ExitCode()))
		}
		return fmt.Errorf("executing cue command: %w", err)
	}

	return nil
}

// Vet runs `cue vet` on the specified directory.
func Vet(dir string, concrete bool, registry string) error {
	if err := CheckCUEVersion(); err != nil {
		return err
	}

	args := []string{"vet", "./..."}
	if concrete {
		args = append(args, "--concrete")
	}

	return RunCUECommand(dir, args, registry)
}

// Tidy runs `cue mod tidy` on the specified directory.
func Tidy(dir string, registry string) error {
	if err := CheckCUEVersion(); err != nil {
		return err
	}

	args := []string{"mod", "tidy"}
	return RunCUECommand(dir, args, registry)
}
