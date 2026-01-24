package version

import (
	"bytes"
	"os/exec"
	"regexp"
	"strings"
)

// cueVersionRegex matches CUE version output like "cue version v0.11.1"
var cueVersionRegex = regexp.MustCompile(`v?\d+\.\d+\.\d+(?:-[a-zA-Z0-9.]+)?`)

// DetectCUEBinary finds and checks the CUE binary installation.
func DetectCUEBinary() CUEBinaryInfo {
	// Look for cue in PATH
	path, err := exec.LookPath("cue")
	if err != nil {
		return CUEBinaryInfo{
			Found:      false,
			Compatible: false,
			Message:    "CUE binary not found in PATH",
		}
	}

	// Get CUE version
	version, err := getCUEVersion(path)
	if err != nil {
		return CUEBinaryInfo{
			Path:       path,
			Found:      true,
			Compatible: false,
			Message:    "failed to get CUE version: " + err.Error(),
		}
	}

	// Check compatibility
	compatible := CUEVersionCompatible(CUESDKVersion, version)
	message := CompatibilityMessage(CUESDKVersion, version)

	return CUEBinaryInfo{
		Version:    version,
		Path:       path,
		Found:      true,
		Compatible: compatible,
		Message:    message,
	}
}

// getCUEVersion executes 'cue version' and extracts the version string.
func getCUEVersion(cuePath string) (string, error) {
	cmd := exec.Command(cuePath, "version")
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	if err := cmd.Run(); err != nil {
		return "", err
	}

	return extractVersion(out.String())
}

// extractVersion extracts the version number from CUE version output.
func extractVersion(output string) (string, error) {
	// CUE version output format:
	// cue version v0.11.1
	//
	// go version go1.22.0
	// ...

	match := cueVersionRegex.FindString(output)
	if match == "" {
		// Try to find version in first line
		lines := strings.Split(output, "\n")
		if len(lines) > 0 {
			match = cueVersionRegex.FindString(lines[0])
		}
	}

	if match == "" {
		return "", &versionParseError{output: output}
	}

	// Ensure "v" prefix
	if !strings.HasPrefix(match, "v") {
		match = "v" + match
	}

	return match, nil
}

// versionParseError indicates failure to parse CUE version output.
type versionParseError struct {
	output string
}

func (e *versionParseError) Error() string {
	return "failed to parse CUE version from output: " + e.output
}
