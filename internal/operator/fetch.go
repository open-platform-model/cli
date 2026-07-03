package operator

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

// operatorReleaseBaseURL is the GitHub releases download base for opm-operator.
// Release assets live at <base>/<tag>/install.yaml.
const operatorReleaseBaseURL = "https://github.com/open-platform-model/opm-operator/releases/download"

// fetchTimeout bounds how long a --version fetch may take.
const fetchTimeout = 30 * time.Second

// fetchManifest downloads install.yaml from the opm-operator GitHub release
// for the given tag over HTTPS, bounded by fetchTimeout. No checksum or
// signature verification is performed (0006/D35). baseURL is injectable so
// tests can point it at a stub server instead of GitHub.
func fetchManifest(ctx context.Context, baseURL, tag string) ([]byte, error) {
	url := fmt.Sprintf("%s/%s/install.yaml", baseURL, tag)

	ctx, cancel := context.WithTimeout(ctx, fetchTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("building request for opm-operator %s: %w", tag, err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching opm-operator %s from %s: %w", tag, url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetching opm-operator %s: server returned %s for %s", tag, resp.Status, url)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading opm-operator %s manifest from %s: %w", tag, url, err)
	}

	return data, nil
}
