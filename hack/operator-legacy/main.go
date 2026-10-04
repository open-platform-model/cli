// Command operator-legacy records what every opm-operator release that
// published an install manifest put on a cluster, for the proof list
// `opm operator install` migrates an earlier manifest install by
// (internal/operator/legacy.go). It is not linked into the opm binary.
//
//	go run ./hack/operator-legacy            # writes internal/operator/testdata/legacy-manifests.json
//
// It lists the releases of github.com/open-platform-model/opm-operator,
// keeps the published operator releases (tag v<semver>) that attach an
// install.yaml asset, and skips every other tag: an operator module release
// (opm_operator-vX.Y.Z) attaches an install.yaml too, but that manifest is a
// render of the module, whose objects carry the operator instance's identity
// and are its own. For each kept release it writes the group, kind,
// namespace, name and labels of every object, and the Deployment's selector;
// never a spec.
//
// It needs the network and runs by hand, only when an operator release that
// attaches a manifest is missing from the file. GITHUB_TOKEN, when set,
// authenticates the release listing.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"sort"
	"time"

	"golang.org/x/mod/semver"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	utilyaml "k8s.io/apimachinery/pkg/util/yaml"
)

// outFile is where the record lives, relative to the repository root.
const outFile = "internal/operator/testdata/legacy-manifests.json"

const releasesURL = "https://api.github.com/repos/open-platform-model/opm-operator/releases?per_page=100&page=%d"

// manifestAsset is the release asset an operator release's manifest is.
const manifestAsset = "install.yaml"

// Release is one operator release's manifest, reduced to what the proof reads.
type Release struct {
	Tag     string   `json:"tag"`
	Objects []Object `json:"objects"`
}

// Object is one object of a manifest: its identity, its labels and, for a
// Deployment, its pod selector.
type Object struct {
	Group     string            `json:"group"`
	Kind      string            `json:"kind"`
	Namespace string            `json:"namespace,omitempty"`
	Name      string            `json:"name"`
	Labels    map[string]string `json:"labels,omitempty"`
	Selector  map[string]string `json:"selector,omitempty"`
}

// operatorTag is the shape of an operator release tag.
var operatorTag = regexp.MustCompile(`^v\d+\.\d+\.\d+(-[0-9A-Za-z.-]+)?$`)

// IsOperatorRelease reports whether a release tag is an operator release
// (v<semver>), the only kind whose install.yaml is a source of the list.
// An operator module release (opm_operator-vX.Y.Z) and anything malformed
// are not.
func IsOperatorRelease(tag string) bool {
	return operatorTag.MatchString(tag) && semver.IsValid(tag)
}

type ghRelease struct {
	TagName string `json:"tag_name"`
	Draft   bool   `json:"draft"`
	Assets  []struct {
		Name string `json:"name"`
		URL  string `json:"browser_download_url"`
	} `json:"assets"`
}

func main() {
	if err := run(context.Background(), outFile); err != nil {
		fmt.Fprintln(os.Stderr, "operator-legacy:", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, out string) error {
	client := &http.Client{Timeout: 60 * time.Second}
	rels, err := listReleases(ctx, client)
	if err != nil {
		return err
	}
	var records []Release
	for _, r := range rels {
		if r.Draft || !IsOperatorRelease(r.TagName) {
			continue
		}
		for _, a := range r.Assets {
			if a.Name != manifestAsset {
				continue
			}
			data, err := get(ctx, client, a.URL, false)
			if err != nil {
				return fmt.Errorf("%s: %w", r.TagName, err)
			}
			objs, err := ParseManifest(data)
			if err != nil {
				return fmt.Errorf("%s: %w", r.TagName, err)
			}
			records = append(records, Release{Tag: r.TagName, Objects: objs})
		}
	}
	sort.Slice(records, func(i, j int) bool { return semver.Compare(records[i].Tag, records[j].Tag) < 0 })
	data, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(out, append(data, '\n'), 0o600)
}

func listReleases(ctx context.Context, client *http.Client) ([]ghRelease, error) {
	var all []ghRelease
	for page := 1; ; page++ {
		data, err := get(ctx, client, fmt.Sprintf(releasesURL, page), true)
		if err != nil {
			return nil, err
		}
		var rels []ghRelease
		if err := json.Unmarshal(data, &rels); err != nil {
			return nil, fmt.Errorf("decoding the release list: %w", err)
		}
		if len(rels) == 0 {
			return all, nil
		}
		all = append(all, rels...)
	}
}

func get(ctx context.Context, client *http.Client, url string, api bool) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return nil, err
	}
	if api {
		req.Header.Set("Accept", "application/vnd.github+json")
		if tok := os.Getenv("GITHUB_TOKEN"); tok != "" {
			req.Header.Set("Authorization", "Bearer "+tok)
		}
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET %s: %s", url, resp.Status)
	}
	return io.ReadAll(resp.Body)
}

// ParseManifest reduces a multi-document manifest to its objects' identity,
// labels and Deployment selector, in manifest order.
func ParseManifest(data []byte) ([]Object, error) {
	dec := utilyaml.NewYAMLOrJSONDecoder(bytes.NewReader(data), 4096)
	var objs []Object
	for {
		var m map[string]any
		if err := dec.Decode(&m); errors.Is(err, io.EOF) {
			return objs, nil
		} else if err != nil {
			return nil, err
		}
		if len(m) == 0 {
			continue
		}
		u := &unstructured.Unstructured{Object: m}
		o := Object{
			Group:     u.GroupVersionKind().Group,
			Kind:      u.GetKind(),
			Namespace: u.GetNamespace(),
			Name:      u.GetName(),
			Labels:    u.GetLabels(),
		}
		if o.Kind == "Deployment" {
			sel, _, err := unstructured.NestedStringMap(m, "spec", "selector", "matchLabels")
			if err != nil {
				return nil, fmt.Errorf("selector of Deployment %s: %w", o.Name, err)
			}
			o.Selector = sel
		}
		objs = append(objs, o)
	}
}
