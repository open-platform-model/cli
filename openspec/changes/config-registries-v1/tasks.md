# Tasks

## 1. Registry Types

- [ ] 1.1 Create `internal/config/registry.go` with RegistryEntry struct
- [ ] 1.2 Define RegistryMap type as `map[string]RegistryEntry`
- [ ] 1.3 Implement `ToCUERegistry()` method (sort by prefix length, longest first)
- [ ] 1.4 Implement `ParseCUERegistry()` function (handle single URL backward compat)

## 2. Config Struct Update

- [ ] 2.1 Update `Config` struct: replace `Registry string` with `Registries RegistryMap`
- [ ] 2.2 Update `OPMConfig` struct to use RegistryMap
- [ ] 2.3 Update `DefaultConfig()` to return empty RegistryMap

## 3. Config Loading

- [ ] 3.1 Update `BootstrapRegistry()` to extract registries map via regex
- [ ] 3.2 Update `loadFullConfig()` to parse registries field from CUE
- [ ] 3.3 Update `ResolveRegistry()` to handle RegistryMap and CUE_REGISTRY format
- [ ] 3.4 Implement precedence: flag > OPM_REGISTRY env > config.registries

## 4. Config Template

- [ ] 4.1 Update `templates.go` default config to use registries format
- [ ] 4.2 Update `opm config init` output

## 5. Testing

- [ ] 5.1 Unit tests for `ToCUERegistry()` (empty map, single entry, multiple entries, insecure)
- [ ] 5.2 Unit tests for `ParseCUERegistry()` (valid formats, single URL, invalid input)
- [ ] 5.3 Unit tests for precedence resolution (flag > env > config)
- [ ] 5.4 Unit tests for backward compatibility (single URL in OPM_REGISTRY)
- [ ] 5.5 Integration test with multiple registries in config.cue

## 6. Validation

- [ ] 6.1 Run `task fmt`
- [ ] 6.2 Run `task lint`
- [ ] 6.3 Run `task test`
