# Tasks

## 1. OCI Client

- [ ] 1.1 Create `internal/oci/` package
- [ ] 1.2 Implement push with oras-go
- [ ] 1.3 Implement pull with oras-go
- [ ] 1.4 Handle authentication via ~/.docker/config.json

## 2. Publish Command

- [ ] 2.1 Implement `mod_publish.go`
- [ ] 2.2 Validate module before publish (vet)
- [ ] 2.3 Package as OCI artifact (CUE-compatible format)
- [ ] 2.4 Push to registry with SemVer tag
- [ ] 2.5 Add `--force` flag to overwrite existing version

## 3. Get Command

- [ ] 3.1 Implement `mod_get.go`
- [ ] 3.2 Resolve version (reject @latest)
- [ ] 3.3 Pull from registry to CUE cache
- [ ] 3.4 Update `module.cue` deps field
- [ ] 3.5 Resolve transitive dependencies

## 4. Update Command

- [ ] 4.1 Implement `mod_update.go`
- [ ] 4.2 Query registry for newer versions
- [ ] 4.3 Default to patch/minor only
- [ ] 4.4 Add `--major` flag for major updates
- [ ] 4.5 Add `--check` flag for CI (non-zero exit if updates available)
- [ ] 4.6 Interactive confirmation prompt

## 5. Tidy Command

- [ ] 5.1 Implement `mod_tidy.go`
- [ ] 5.2 Analyze imports to find used dependencies
- [ ] 5.3 Remove unused from `module.cue` deps
- [ ] 5.4 Clean local cache of unused modules

## 6. Exit Codes

- [ ] 6.1 Implement standardized exit codes (0-5)
- [ ] 6.2 Add clear error messages with hints
- [ ] 6.3 Graceful network failure handling (no stack traces)

## 7. Testing

- [ ] 7.1 Unit tests with mock registry
- [ ] 7.2 Integration tests with local registry (zot)
- [ ] 7.3 Test diamond dependency resolution (MVS)
