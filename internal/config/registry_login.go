package config

import "cuelang.org/go/mod/modconfig"

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

// soleRegistryHost names the one host the registry mapping (CUE_REGISTRY
// syntax; empty reads CUE_REGISTRY from the environment) holds, in the form
// `opm registry login` takes: the host, with +insecure when it is served
// over plain HTTP. It returns "" when the mapping holds several hosts, none,
// or does not parse: a refusal met somewhere in a dependency graph is then
// not known to come from any one of them.
func soleRegistryHost(registry string) string {
	resolver, err := modconfig.NewResolver(&modconfig.Config{CUERegistry: registry})
	if err != nil {
		return ""
	}
	hosts := resolver.AllHosts()
	if len(hosts) != 1 {
		return ""
	}
	if hosts[0].Insecure {
		return hosts[0].Name + "+insecure"
	}
	return hosts[0].Name
}
