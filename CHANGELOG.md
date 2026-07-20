# Changelog

## [1.0.0-alpha.4](https://github.com/open-platform-model/cli/compare/v1.0.0-alpha.3...v1.0.0-alpha.4) (2026-07-20)


### Features

* **instance:** add handoff and operator-owned apply/delete (0006 C3) ([#116](https://github.com/open-platform-model/cli/issues/116)) ([093b976](https://github.com/open-platform-model/cli/commit/093b9761453437e7ceb8c15c75c156ebbd971d94))

## [1.0.0-alpha.3](https://github.com/open-platform-model/cli/compare/v1.0.0-alpha.2...v1.0.0-alpha.3) (2026-07-18)


### Miscellaneous Chores

* **openspec:** archive cli-kernel-adoption and sync 25 delta specs ([#114](https://github.com/open-platform-model/cli/issues/114)) ([cb70108](https://github.com/open-platform-model/cli/commit/cb70108a6a0b38c87ec6fa22216cc055f67d6b26))

## [1.0.0-alpha.2](https://github.com/open-platform-model/cli/compare/v1.0.0-alpha.1...v1.0.0-alpha.2) (2026-07-18)


### ⚠ BREAKING CHANGES

* ~/.opm/config.cue no longer accepts providers or cacheDir and ~/.opm is no longer a CUE module; re-run opm config init. The render path errors on providers until kernel adoption (Phase C of the same change) lands; the phases ship as one PR.

### Features

* render through the library kernel and simplify ~/.opm to two data files (0006 C2) ([#112](https://github.com/open-platform-model/cli/issues/112)) ([2ba7c40](https://github.com/open-platform-model/cli/commit/2ba7c4084d7c3ee57bfdfa8d3a5ab4a35e504aa0))

## [1.0.0-alpha.1](https://github.com/open-platform-model/cli/compare/v1.0.0-alpha...v1.0.0-alpha.1) (2026-07-17)


### Features

* **cli:** operator install command ([#105](https://github.com/open-platform-model/cli/issues/105)) ([5ab639b](https://github.com/open-platform-model/cli/commit/5ab639bcd88f54489722a01f436d217a2870c9e6))


### Documentation

* **openspec:** draft cli-cr-inventory-backend change (0006 C1) ([4cc446b](https://github.com/open-platform-model/cli/commit/4cc446baaaa628d8033be68dc79d9daa850a42f7))


### Code Refactoring

* **cli:** rename go module to github.com/open-platform-model/cli ([#101](https://github.com/open-platform-model/cli/issues/101)) ([35fe6e3](https://github.com/open-platform-model/cli/commit/35fe6e3db51febaccae274dfa477588985c1a1f8))


### Miscellaneous Chores

* drop the sticky release-as override from release-please ([#110](https://github.com/open-platform-model/cli/issues/110)) ([b312be3](https://github.com/open-platform-model/cli/commit/b312be32a8743b2a9b42f627f7f352caab262d9f))
* **main:** release 1.0.0-alpha ([#102](https://github.com/open-platform-model/cli/issues/102)) ([26cfcf5](https://github.com/open-platform-model/cli/commit/26cfcf5aac7bd25626356341b7a796ac08d45266))
* **main:** release 1.0.0-alpha ([#104](https://github.com/open-platform-model/cli/issues/104)) ([39ab8c2](https://github.com/open-platform-model/cli/commit/39ab8c22c12e4beb5b7f9e959248f1fa53b40ae9))
* **main:** release 1.0.0-alpha ([#107](https://github.com/open-platform-model/cli/issues/107)) ([2dd0f69](https://github.com/open-platform-model/cli/commit/2dd0f69496420d1caa84784eff6f4d8f8b081a3f))

## [1.0.0-alpha](https://github.com/open-platform-model/cli/compare/v1.0.0-alpha...v1.0.0-alpha) (2026-07-16)


### Features

* **cli:** operator install command ([#105](https://github.com/open-platform-model/cli/issues/105)) ([5ab639b](https://github.com/open-platform-model/cli/commit/5ab639bcd88f54489722a01f436d217a2870c9e6))


### Documentation

* **openspec:** draft cli-cr-inventory-backend change (0006 C1) ([4cc446b](https://github.com/open-platform-model/cli/commit/4cc446baaaa628d8033be68dc79d9daa850a42f7))


### Code Refactoring

* **cli:** rename go module to github.com/open-platform-model/cli ([#101](https://github.com/open-platform-model/cli/issues/101)) ([35fe6e3](https://github.com/open-platform-model/cli/commit/35fe6e3db51febaccae274dfa477588985c1a1f8))


### Miscellaneous Chores

* **main:** release 1.0.0-alpha ([#102](https://github.com/open-platform-model/cli/issues/102)) ([26cfcf5](https://github.com/open-platform-model/cli/commit/26cfcf5aac7bd25626356341b7a796ac08d45266))
* **main:** release 1.0.0-alpha ([#104](https://github.com/open-platform-model/cli/issues/104)) ([39ab8c2](https://github.com/open-platform-model/cli/commit/39ab8c22c12e4beb5b7f9e959248f1fa53b40ae9))

## [1.0.0-alpha](https://github.com/open-platform-model/cli/compare/v1.0.0-alpha...v1.0.0-alpha) (2026-07-16)


### Features

* **cli:** operator install command ([#105](https://github.com/open-platform-model/cli/issues/105)) ([5ab639b](https://github.com/open-platform-model/cli/commit/5ab639bcd88f54489722a01f436d217a2870c9e6))


### Code Refactoring

* **cli:** rename go module to github.com/open-platform-model/cli ([#101](https://github.com/open-platform-model/cli/issues/101)) ([35fe6e3](https://github.com/open-platform-model/cli/commit/35fe6e3db51febaccae274dfa477588985c1a1f8))


### Miscellaneous Chores

* **main:** release 1.0.0-alpha ([#102](https://github.com/open-platform-model/cli/issues/102)) ([26cfcf5](https://github.com/open-platform-model/cli/commit/26cfcf5aac7bd25626356341b7a796ac08d45266))

## [1.0.0-alpha](https://github.com/open-platform-model/cli/compare/v1.0.0-alpha...v1.0.0-alpha) (2026-07-01)


### Code Refactoring

* **cli:** rename go module to github.com/open-platform-model/cli ([#101](https://github.com/open-platform-model/cli/issues/101)) ([35fe6e3](https://github.com/open-platform-model/cli/commit/35fe6e3db51febaccae274dfa477588985c1a1f8))

## [1.0.0-alpha](https://github.com/open-platform-model/cli/compare/v0.6.0...v1.0.0-alpha) (2026-06-30)


### Features

* **module:** add `opm module apply` subcommand ([04d93aa](https://github.com/open-platform-model/cli/commit/04d93aaa931a42f054a4d6290826caa98f97bd5a))
* **security-audit:** add registry/k8s cli security audit skill ([20d010c](https://github.com/open-platform-model/cli/commit/20d010c19b573db779b1731c37c196caf178d4a3))


### Documentation

* **commit:** allow co-authored-by attribution trailer ([e187fd7](https://github.com/open-platform-model/cli/commit/e187fd70be54bca03023cfe5d2c80f2dd8865163))
* drop ADR workflow section from CLAUDE.md ([a22554e](https://github.com/open-platform-model/cli/commit/a22554e3dce7b52063f2519ae7158e6966597eda))
* require claude co-authorship trailer in commits ([#89](https://github.com/open-platform-model/cli/issues/89)) ([232aa06](https://github.com/open-platform-model/cli/commit/232aa062349f18bf87c6aa5bebb4d099a34f57c8))


### Miscellaneous Chores

* configure release-please for the v1 alpha prerelease line ([#96](https://github.com/open-platform-model/cli/issues/96)) ([cc9efe8](https://github.com/open-platform-model/cli/commit/cc9efe871bba5dd0e4ab48626026e811378960e2))
* **deps:** bump module deps in examples and fixtures ([010aa1e](https://github.com/open-platform-model/cli/commit/010aa1e46d44b1584bb4abc1e7f7f0f5a7749015))
* **rfc:** Add handoff rfc ([061544b](https://github.com/open-platform-model/cli/commit/061544bff98b786c050ebca298b1ebd3fc89a2c3))
* **skill:** Add instructions on how to write commit messages ([7d17bb6](https://github.com/open-platform-model/cli/commit/7d17bb60a4dfee7832f69d384414a3f0667de04b))

## [0.6.0](https://github.com/open-platform-model/cli/compare/v0.5.1...v0.6.0) (2026-05-07)


### Features

* **config:** auto-resolve dependencies on `opm config init` ([f852b7b](https://github.com/open-platform-model/cli/commit/f852b7b460d5f59aa4e5a204367ddf6ffcca363f))
* **config:** auto-resolve dependencies on `opm config init` ([d01944d](https://github.com/open-platform-model/cli/commit/d01944d47a261647bbf83346d716865fc253e5fd))

## [0.5.1](https://github.com/open-platform-model/cli/compare/v0.5.0...v0.5.1) (2026-05-06)


### Miscellaneous Chores

* **openspec:** archive module-synthetic-build and sync specs ([b00aab5](https://github.com/open-platform-model/cli/commit/b00aab52f5c33408ac2f9df9c1f4bfd1e23ce8c1))

## [0.5.0](https://github.com/open-platform-model/cli/compare/v0.4.0...v0.5.0) (2026-05-06)


### Features

* **module:** add synthetic release build for module directories ([996cb9f](https://github.com/open-platform-model/cli/commit/996cb9f69c2c18b44f20583b51039d137ec59965))

## [0.4.0](https://github.com/open-platform-model/cli/compare/v0.3.0...v0.4.0) (2026-05-06)


### Features

* **config:** default registry to ghcr.io/open-platform-model ([1e54ea9](https://github.com/open-platform-model/cli/commit/1e54ea97cad99df8730efb030f018fc7d74d3d6a))

## [0.3.0](https://github.com/open-platform-model/cli/compare/v0.2.0...v0.3.0) (2026-05-05)


### Features

* **render:** inject runtime identity via #runtimeName ([f76f03f](https://github.com/open-platform-model/cli/commit/f76f03f1014e845c335aa392ec8ec0242a71dfeb))


### Bug Fixes

* **module-init:** scaffolds now vet clean and reject bad names ([ad2c3ed](https://github.com/open-platform-model/cli/commit/ad2c3eda7a8e4d07e5740b0347c37f374700b8fc))


### Documentation

* **enhancements:** remove duplicate metadata tables from template sub-files ([ce010f8](https://github.com/open-platform-model/cli/commit/ce010f8a5c3e22055cacb9efa9a77398ab300504))
* rename poc-controller references to opm-operator ([92212b8](https://github.com/open-platform-model/cli/commit/92212b85cd4fa6b5c914f250da59e51b0a6aec46))


### Miscellaneous Chores

* **cue-deps:** bump core/v1alpha1 pin to v1.3.9 in examples and fixtures ([db23e1a](https://github.com/open-platform-model/cli/commit/db23e1a4f95fee736f28aa7c82e776ff44cb5f40))
* **deps:** bump cuelang.org/go to v0.16.1 ([60b7ab0](https://github.com/open-platform-model/cli/commit/60b7ab05a6bd70edf804ee12cd463426a57a29d1))
* rename examples task update-deps to deps:update ([480a81c](https://github.com/open-platform-model/cli/commit/480a81c434db7eb309097a9bf033b6a3dad5af11))
