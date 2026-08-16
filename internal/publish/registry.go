package publish

import (
	"context"
	"fmt"
	"os"

	"cuelang.org/go/mod/modconfig"
	"cuelang.org/go/mod/modregistry"
	"cuelang.org/go/mod/module"
	"cuelang.org/go/mod/modzip"
	"golang.org/x/mod/semver"

	"github.com/open-platform-model/cli/internal/cueedit"
)

// newRegistryClient builds a module-registry client through the same resolver
// chain reads use: modconfig routes by CUE registry syntax and layers Docker
// and Podman credentials under CUE's logins.json, so push and pull share one
// path (D11). No os.Setenv — the resolved registry rides the config struct.
func newRegistryClient(registry string) (*modregistry.Client, error) {
	resolver, err := modconfig.NewResolver(&modconfig.Config{CUERegistry: registry})
	if err != nil {
		return nil, fmt.Errorf("resolving registry configuration: %w", err)
	}
	return modregistry.NewClientWithResolver(resolver), nil
}

// gateAlreadyPublished is D15's refusal 8: a published tag names fixed bytes
// permanently, so publishing over one is refused — no flag or mode turns it
// into a success. An unreachable registry is a *ConnectivityError, not a
// refusal: the artifact was never judged.
func gateAlreadyPublished(ctx context.Context, p *Plan, opts Options) error {
	if p.DeclaredPath == "" || p.Tag == "" {
		return nil
	}
	client, err := newRegistryClient(opts.Registry)
	if err != nil {
		return err
	}
	versions, err := client.ModuleVersions(ctx, p.DeclaredPath)
	if err != nil {
		return &ConnectivityError{
			Op:  fmt.Sprintf("listing published versions of %s", p.DeclaredPath),
			Err: err,
		}
	}
	p.RegistryChecked = true
	p.registryClient = client
	for _, v := range versions {
		if v != p.Tag {
			continue
		}
		p.refuse(Refusal{
			Headline:    fmt.Sprintf("%s already holds %s", p.DeclaredPath, p.Tag),
			Consequence: "A published tag names fixed bytes permanently. Publishing cannot replace\nit, and this command will not silently do nothing.",
			Action:      fmt.Sprintf("Bump and retry:  opm %s version set %s", p.Kind, nextPatch(p.Tag)),
		})
		return nil
	}
	return nil
}

// Push publishes the plan: write the --version fill into the working tree
// when one is pending (D12 — the pushed bytes come from disk), zip the
// committed directory, and put it through CUE's own module machinery. Callers
// invoke it only on a GO plan outside --dry-run.
func Push(ctx context.Context, opts Options, p *Plan) error {
	if !p.Go() {
		return fmt.Errorf("internal error: push invoked on a refused plan")
	}
	if p.FillVersion != "" {
		if _, err := cueedit.SetIdentityVersion(p.Dir, p.FillVersion); err != nil {
			return fmt.Errorf("filling identity Version: %w", err)
		}
	}

	mv, err := module.NewVersion(p.DeclaredPath, p.Tag)
	if err != nil {
		return fmt.Errorf("forming module version from %s and %s: %w", p.DeclaredPath, p.Tag, err)
	}

	zipFile, err := os.CreateTemp("", "opm-publish-*.zip")
	if err != nil {
		return fmt.Errorf("creating zip staging file: %w", err)
	}
	defer func() {
		zipFile.Close()
		os.Remove(zipFile.Name())
	}()

	if err := modzip.CreateFromDir(zipFile, mv, p.Dir); err != nil {
		return fmt.Errorf("zipping %s: %w", p.Dir, err)
	}
	info, err := zipFile.Stat()
	if err != nil {
		return fmt.Errorf("reading zip staging file: %w", err)
	}

	client := p.registryClient
	if client == nil {
		if client, err = newRegistryClient(opts.Registry); err != nil {
			return err
		}
	}
	if err := client.PutModule(ctx, mv, zipFile, info.Size()); err != nil {
		return &ConnectivityError{
			Op:  fmt.Sprintf("pushing %s:%s", p.RegistryRepo, p.Tag),
			Err: err,
		}
	}
	return nil
}

// nextPatch suggests the patch bump after tag for refusal 8's action.
func nextPatch(tag string) string {
	if !semver.IsValid(tag) {
		return "<next-version>"
	}
	var major, minor, patch int
	if _, err := fmt.Sscanf(semver.Canonical(tag), "v%d.%d.%d", &major, &minor, &patch); err != nil {
		return "<next-version>"
	}
	return fmt.Sprintf("%d.%d.%d", major, minor, patch+1)
}
