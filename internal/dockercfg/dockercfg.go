// Package dockercfg edits the standard OCI/docker credential file — the
// store CUE's resolver reads for both push and pull (0011 D11). The file is
// shared property (docker, podman, credential helpers, future tools), so the
// writer edits exactly one auths entry and passes every other key through a
// raw-message envelope untouched.
package dockercfg

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Path returns the credential file a write must land in, mirroring the read
// side's resolution order so the written entry is the one the push will
// find: $DOCKER_CONFIG/config.json, else ~/.docker/config.json.
func Path() (string, error) {
	if dir := os.Getenv("DOCKER_CONFIG"); dir != "" {
		return filepath.Join(dir, "config.json"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolving home directory: %w", err)
	}
	return filepath.Join(home, ".docker", "config.json"), nil
}

// Upsert sets auths[host] to a basic credential for user:secret in the file
// at path, creating the file 0600 (and its directory) when absent. Only the
// targeted host's entry changes: the top level and the auths map are decoded
// as raw messages, so credsStore, credHelpers, foreign hosts, and any key
// this package has never heard of survive the rewrite. A file that is not
// valid JSON is refused rather than rewritten. The write is a same-directory
// temp file plus atomic rename; docker itself takes no file lock, and the
// rename bounds concurrent damage the same way.
func Upsert(path, host, user, secret string) error {
	top := map[string]json.RawMessage{}
	mode := os.FileMode(0o600)
	data, err := os.ReadFile(path)
	switch {
	case err == nil:
		if err := json.Unmarshal(data, &top); err != nil {
			return fmt.Errorf("%s is not valid JSON, refusing to rewrite it: %w", path, err)
		}
		if info, statErr := os.Stat(path); statErr == nil {
			mode = info.Mode().Perm()
		}
	case os.IsNotExist(err):
		// Fresh file: start from the empty envelope.
	default:
		return fmt.Errorf("reading %s: %w", path, err)
	}

	auths := map[string]json.RawMessage{}
	if raw, ok := top["auths"]; ok {
		if err := json.Unmarshal(raw, &auths); err != nil {
			return fmt.Errorf("%s: auths is not a JSON object, refusing to rewrite it: %w", path, err)
		}
	}

	entry, err := json.Marshal(map[string]string{
		"auth": base64.StdEncoding.EncodeToString([]byte(user + ":" + secret)),
	})
	if err != nil {
		return fmt.Errorf("encoding credential for %s: %w", host, err)
	}
	auths[host] = entry

	rawAuths, err := json.Marshal(auths)
	if err != nil {
		return fmt.Errorf("encoding auths: %w", err)
	}
	top["auths"] = rawAuths

	out, err := json.MarshalIndent(top, "", "\t")
	if err != nil {
		return fmt.Errorf("encoding %s: %w", path, err)
	}
	return writeAtomic(path, append(out, '\n'), mode)
}

// writeAtomic lands data at path via a same-directory temp file and rename,
// creating the directory when absent.
func writeAtomic(path string, data []byte, mode os.FileMode) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("creating %s: %w", dir, err)
	}
	tmp, err := os.CreateTemp(dir, ".config-*.json")
	if err != nil {
		return fmt.Errorf("staging write to %s: %w", path, err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName) // no-op after a successful rename

	if err := tmp.Chmod(mode); err != nil {
		tmp.Close()
		return fmt.Errorf("setting permissions on %s: %w", path, err)
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("writing %s: %w", path, err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("replacing %s: %w", path, err)
	}
	return nil
}
