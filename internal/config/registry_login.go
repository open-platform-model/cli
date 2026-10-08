package config

import (
	"os"
	"path/filepath"
	"strings"

	"cuelang.org/go/mod/modconfig"
	"cuelang.org/go/mod/modfile"
)

// RegistryLoginHint is the next step after a registry refused the caller:
// the login command for host, in the form `opm registry login` takes. With
// no known host it is the bare command, which resolves the configured
// mapping and lists the hosts when it names several.
func RegistryLoginHint(host string) string {
	const hint = "Log in to the registry, then retry:  opm registry login"
	if host == "" {
		return hint
	}
	return hint + " " + host
}

// platformRegistryHost names the registry host a platform module's
// dependencies are fetched from, in the form `opm registry login` takes: the
// host, with +insecure when it is served over plain HTTP. It routes every
// dependency the module file at dir declares through the registry mapping
// (CUE_REGISTRY syntax; empty reads CUE_REGISTRY from the environment) and
// answers only when all of them reach one host. A refused fetch is met
// somewhere in the dependency graph and its error names no host, so with
// dependencies on several hosts, with none declared, or with a module file
// or a mapping that does not parse, it returns "": no single host is known
// to be the one that refused.
//
// The mapping's own host count is not the measure: CUE adds its central
// registry as the catch-all of every prefix mapping, the cli's default
// included, so such a mapping always holds two hosts.
func platformRegistryHost(dir, registry string) string {
	modPath := filepath.Join(dir, filepath.FromSlash(PlatformModuleFileName))
	data, err := os.ReadFile(modPath)
	if err != nil {
		return ""
	}
	mf, err := modfile.ParseNonStrict(data, modPath)
	if err != nil {
		return ""
	}
	resolver, err := modconfig.NewResolver(&modconfig.Config{CUERegistry: registry})
	if err != nil {
		return ""
	}
	host := ""
	for dep, pin := range mf.Deps {
		base, _, _ := strings.Cut(dep, "@")
		loc, ok := resolver.ResolveToLocation(base, pin.Version)
		if !ok || loc.Host == "" {
			return ""
		}
		h := loc.Host
		if loc.Insecure {
			h += "+insecure"
		}
		if host != "" && h != host {
			return ""
		}
		host = h
	}
	return host
}
