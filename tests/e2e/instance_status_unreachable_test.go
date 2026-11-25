package e2e

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// unreachableKubeconfig is a kubeconfig whose API server refuses every
// connection: nothing listens on port 1 of the loopback address.
const unreachableKubeconfig = `apiVersion: v1
kind: Config
clusters:
- name: unreachable
  cluster:
    server: https://127.0.0.1:1
    insecure-skip-tls-verify: true
users:
- name: nobody
  user:
    token: not-a-real-token
contexts:
- name: unreachable
  context:
    cluster: unreachable
    user: nobody
    namespace: default
current-context: unreachable
`

// TestE2E_InstanceStatus_UnreachableClusterExits3 runs `opm instance status`
// against a cluster that refuses the connection. The record read is the
// command's first request, so it is where the unreachable cluster fails: the
// command exits 3 (connectivity error) and prints the error once. It needs no
// kind cluster.
func TestE2E_InstanceStatus_UnreachableClusterExits3(t *testing.T) {
	kubeconfig := filepath.Join(t.TempDir(), "kubeconfig")
	require.NoError(t, os.WriteFile(kubeconfig, []byte(unreachableKubeconfig), 0o600))

	_, stderr, err := runOPMWithEnv(t, t.TempDir(), homeDir, 60*time.Second,
		"instance", "status", "my-app", "-n", "default",
		"--config", repoPath(t, "hack/opm-config.cue"),
		"--kubeconfig", kubeconfig,
		"--context", "unreachable",
	)

	var exitErr *exec.ExitError
	require.True(t, errors.As(err, &exitErr), "opm must exit non-zero, got err=%v\nstderr:\n%s", err, stderr)
	assert.Equal(t, 3, exitErr.ExitCode(), "exit code\nstderr:\n%s", stderr)
	assert.Equal(t, 1, strings.Count(stderr, "connection refused"),
		"the record-read error must appear once on stderr\nstderr:\n%s", stderr)
}
