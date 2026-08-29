# Changelog

## [1.0.0-alpha.16](https://github.com/open-platform-model/cli/compare/v1.0.0-alpha.15...v1.0.0-alpha.16) (2026-08-29)


### Features

* **publish:** run the kernel module loader as a publish and vet gate ([#186](https://github.com/open-platform-model/cli/issues/186)) ([80aa234](https://github.com/open-platform-model/cli/commit/80aa2347b871924ba6c93ab785ef7c4c2c29618e))


### Bug Fixes

* **scaffold:** write identity Version as a plain literal ([#183](https://github.com/open-platform-model/cli/issues/183)) ([f3d722f](https://github.com/open-platform-model/cli/commit/f3d722f78886b8436e37c66262c5208098823f92))

## [1.0.0-alpha.15](https://github.com/open-platform-model/cli/compare/v1.0.0-alpha.14...v1.0.0-alpha.15) (2026-08-27)


### Miscellaneous Chores

* **deps:** bump catalogs/opm to v2.0.0-alpha.6 in examples and templates ([#174](https://github.com/open-platform-model/cli/issues/174)) ([8926091](https://github.com/open-platform-model/cli/commit/89260912191ae303a4e4a7d15a9525d0d7bd4a1d))
* **fixtures:** seed platform on catalogs/opm alpha.6 and republish podinfo at 0.1.5 ([77d9fc0](https://github.com/open-platform-model/cli/commit/77d9fc0d7892db020832433b62fd22c06ff3c09c))
* **fixtures:** seed platform on catalogs/opm alpha.6 and republish podinfo at 0.1.5 ([#177](https://github.com/open-platform-model/cli/issues/177)) ([a6870b8](https://github.com/open-platform-model/cli/commit/a6870b848a555f3a59ae63887096b2842c01434a))

## [1.0.0-alpha.14](https://github.com/open-platform-model/cli/compare/v1.0.0-alpha.13...v1.0.0-alpha.14) (2026-08-27)


### Features

* **config:** seed the default platform with both first-party catalogs ([#163](https://github.com/open-platform-model/cli/issues/163)) ([4af7cc9](https://github.com/open-platform-model/cli/commit/4af7cc95658a21092a74c389555a1370af4cb4e7))
* **openspec:** wire enhancement delivery-log declarations into the workflow ([53a7d15](https://github.com/open-platform-model/cli/commit/53a7d15c061b193866b0b3528186259bac2e0c6b))
* **operator:** seed the cluster Platform from the resolved catalog ([#156](https://github.com/open-platform-model/cli/issues/156)) ([331c696](https://github.com/open-platform-model/cli/commit/331c696918da91848facae70d1abe394db92f9ac))
* **publish:** exempt beta/GA members on a prerelease module line ([#172](https://github.com/open-platform-model/cli/issues/172)) ([b9b42ae](https://github.com/open-platform-model/cli/commit/b9b42ae561d9011f49a1c2a62fc6dc791dbd279e))
* **publish:** exempt dev builds from the compat gate ([#166](https://github.com/open-platform-model/cli/issues/166)) ([0df6014](https://github.com/open-platform-model/cli/commit/0df601459da2dc12e930d1e2a1b4f729e3e7af9d))


### Bug Fixes

* **openspec:** quote design rules so they parse as strings ([2370bd6](https://github.com/open-platform-model/cli/commit/2370bd6b967f9d59caa18e9e1652c033f85b9125))
* **render:** carry resolved platform spec for cluster seeding ([ca82241](https://github.com/open-platform-model/cli/commit/ca82241a35ef6f448847b9fe1ae92d9408afb4a7))


### Documentation

* **config:** propose seeding both catalogs in the default platform ([#161](https://github.com/open-platform-model/cli/issues/161)) ([78b9110](https://github.com/open-platform-model/cli/commit/78b911059bcf364198b7d590a9ae8993b912c50b))
* **openspec:** archive seed-both-catalogs and sync delta specs ([e5e4b53](https://github.com/open-platform-model/cli/commit/e5e4b53708555f91691b3d87bb52776601a613a5))


### Miscellaneous Chores

* **deps:** bump core, catalogs/opm and k8s.io pins in examples and templates ([#164](https://github.com/open-platform-model/cli/issues/164)) ([e441dc6](https://github.com/open-platform-model/cli/commit/e441dc6317636ca20a4e297d80eb8c9f0bd64c4e))
* **openspec:** archive operator-install-platform and sync delta specs ([#162](https://github.com/open-platform-model/cli/issues/162)) ([6ffca34](https://github.com/open-platform-model/cli/commit/6ffca34c5a8ee9eb7af38f7e5135c8c790ed0715))
* **openspec:** regenerate skills to 1.9.0, preserving local additions ([10b41b9](https://github.com/open-platform-model/cli/commit/10b41b90fa1aa4aeafeccef5500ff74b251e67c6))

## [1.0.0-alpha.13](https://github.com/open-platform-model/cli/compare/v1.0.0-alpha.12...v1.0.0-alpha.13) (2026-08-18)


### Miscellaneous Chores

* **openspec:** withdraw the cue-binary-integration change ([#154](https://github.com/open-platform-model/cli/issues/154)) ([01ba23f](https://github.com/open-platform-model/cli/commit/01ba23f3d57b32e00d42fc5690cf3281d00a30ea))

## [1.0.0-alpha.12](https://github.com/open-platform-model/cli/compare/v1.0.0-alpha.11...v1.0.0-alpha.12) (2026-08-18)


### Features

* **cmd:** add i and ins aliases to the instance command group ([#151](https://github.com/open-platform-model/cli/issues/151)) ([96ea63c](https://github.com/open-platform-model/cli/commit/96ea63c71c97b364642ed052f32a6b1ef9337b73))
* **fixtures:** move podinfo to testing.opmodel.dev and publish it to GHCR ([#153](https://github.com/open-platform-model/cli/issues/153)) ([33abb0e](https://github.com/open-platform-model/cli/commit/33abb0e2d3912fd5cb3c0d6aed25d1a77628b386))

## [1.0.0-alpha.11](https://github.com/open-platform-model/cli/compare/v1.0.0-alpha.10...v1.0.0-alpha.11) (2026-08-18)


### Features

* **cmd:** template modules - fetch-based mod init from published templates ([#149](https://github.com/open-platform-model/cli/issues/149)) ([6410e02](https://github.com/open-platform-model/cli/commit/6410e02d5b8ab1d861b8ec67bf6568e24d9d6485))

## [1.0.0-alpha.10](https://github.com/open-platform-model/cli/compare/v1.0.0-alpha.9...v1.0.0-alpha.10) (2026-08-17)


### Features

* **cmd:** module and catalog version set over an idempotent writer toolkit ([#145](https://github.com/open-platform-model/cli/issues/145)) ([4de3b0b](https://github.com/open-platform-model/cli/commit/4de3b0be508d09cf38e8adcdba9ddf81775f4eee))
* **publish:** catalog member, posture and compat gates plus registry check ([#147](https://github.com/open-platform-model/cli/issues/147)) ([ad8f2e9](https://github.com/open-platform-model/cli/commit/ad8f2e9e2f794b069d0c5c628300c9445dbd378e))
* **publish:** identity-driven publish pipeline for modules and catalogs ([#144](https://github.com/open-platform-model/cli/issues/144)) ([938055e](https://github.com/open-platform-model/cli/commit/938055e5f2caacbf6487ba4ce2b4f16139c93160))
* **registry:** add opm registry login over a docker config writer ([#148](https://github.com/open-platform-model/cli/issues/148)) ([8374728](https://github.com/open-platform-model/cli/commit/8374728cbf6ebafa7cce7147946f4b17c4f515f6))


### Documentation

* **openspec:** rename login command to opm registry login (0011 D24) ([54eff29](https://github.com/open-platform-model/cli/commit/54eff2902f3e62d281fe825fe76c6c2f24b7c859))


### Miscellaneous Chores

* **openspec:** deny delivery-operation tasks in tasks.md ([#142](https://github.com/open-platform-model/cli/issues/142)) ([62007e4](https://github.com/open-platform-model/cli/commit/62007e4f7f252dba79c17c7f860ed0383e3598e9))

## [1.0.0-alpha.9](https://github.com/open-platform-model/cli/compare/v1.0.0-alpha.8...v1.0.0-alpha.9) (2026-08-15)


### ⚠ BREAKING CHANGES

* cross the CLI to core v2 and scalar platform subscriptions ([#141](https://github.com/open-platform-model/cli/issues/141))

### Features

* cross the CLI to core v2 and scalar platform subscriptions ([#141](https://github.com/open-platform-model/cli/issues/141)) ([691d794](https://github.com/open-platform-model/cli/commit/691d794be25396e756e34f6741c1e3a5a6ec5d4d))


### Documentation

* allow plain Claude co-author trailer; keep session-ID ban ([778fee4](https://github.com/open-platform-model/cli/commit/778fee41fbb37efb9f499b5517fb3355fc4585d3))


### Miscellaneous Chores

* **registry:** ghcr-first defaults, fix stale ci localhost mapping ([386a796](https://github.com/open-platform-model/cli/commit/386a796017ee9c40dd2288bba5df5a30a0ef22d7))

## [1.0.0-alpha.8](https://github.com/open-platform-model/cli/compare/v1.0.0-alpha.7...v1.0.0-alpha.8) (2026-08-07)


### Documentation

* forbid bare at-sign mentions in GitHub-destined text ([2d51cfb](https://github.com/open-platform-model/cli/commit/2d51cfbb177c65acb46063988d1b7f5415a5e581))

## [1.0.0-alpha.7](https://github.com/open-platform-model/cli/compare/v1.0.0-alpha.6...v1.0.0-alpha.7) (2026-08-03)


### Performance Improvements

* **cli:** remove client-side api throttling and redundant listings ([#135](https://github.com/open-platform-model/cli/issues/135)) ([eaab04e](https://github.com/open-platform-model/cli/commit/eaab04ef835e409ba808fd70ec972be6d2d1759d))

## [1.0.0-alpha.6](https://github.com/open-platform-model/cli/compare/v1.0.0-alpha.5...v1.0.0-alpha.6) (2026-07-28)


### Documentation

* forbid AI attribution and session links in commits and PRs ([#128](https://github.com/open-platform-model/cli/issues/128)) ([ccfde69](https://github.com/open-platform-model/cli/commit/ccfde6966bd494bb5abdf2bfe1f3f49645890bf4))

## [1.0.0-alpha.5](https://github.com/open-platform-model/cli/compare/v1.0.0-alpha.4...v1.0.0-alpha.5) (2026-07-21)


### Documentation

* **openspec:** draft test-coverage-and-fixture-hygiene change ([#120](https://github.com/open-platform-model/cli/issues/120)) ([37001cc](https://github.com/open-platform-model/cli/commit/37001cc459a579a0bf2991cd77a3795a810f91d7))
* **operator:** refresh stale version examples to v1.0.0-alpha.4 ([13022e8](https://github.com/open-platform-model/cli/commit/13022e82479baf89bdb946dae93c2b4d8c530261))

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
