# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- `--release-id` flag on `opm mod delete` for UUID-based resource deletion
- `--release-id` flag on `opm mod status` for UUID-based status lookup
- `module-release.opmodel.dev/uuid` and `module.opmodel.dev/uuid` labels on all applied resources
- Identity display (Module ID, Release ID) in `opm mod status` output
- `NoResourcesFoundError` type with descriptive error messages

### Changed

- **BREAKING**: `--name` and `--release-id` flags are mutually exclusive on `delete` and `status` commands (previously union)
- **BREAKING**: `opm mod delete` and `opm mod status` return error when no resources match the selector (previously silent success)
- **BREAKING**: `--name` is no longer required on `opm mod delete` — exactly one of `--name` or `--release-id` is required
- **BREAKING**: `--name` is no longer required on `opm mod status` — exactly one of `--name` or `--release-id` is required
- Resource discovery uses a single selector per invocation (removed union/deduplication logic)
- Release identity is extracted from CUE catalog labels instead of computed in Go

### Removed

- `identity.ComputeReleaseIdentity()` Go function (release identity now comes from CUE evaluation)
- `NoResourcesMessage()` helper (replaced by `NoResourcesFoundError` type)
- Union discovery logic and UID-based deduplication in `DiscoverResources`
