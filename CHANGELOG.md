# Changelog

## [1.0.0-beta.7](https://github.com/open-platform-model/cli/compare/v1.0.0-beta.6...v1.0.0-beta.7) (2026-10-03)


### Bug Fixes

* **deps:** bump library to v1.0.0-beta.3 and embed opm-operator v1.0.0-beta.5 ([#291](https://github.com/open-platform-model/cli/issues/291)) ([bd1efe1](https://github.com/open-platform-model/cli/commit/bd1efe1554bb154304d0e8cd40e40c3cd8556f67))

## [1.0.0-beta.6](https://github.com/open-platform-model/cli/compare/v1.0.0-beta.5...v1.0.0-beta.6) (2026-10-03)


### ⚠ BREAKING CHANGES

* **render:** opm instance apply, build, diff and vet now exit 2 when --namespace (-n) or OPM_NAMESPACE differs from the instance file's metadata.namespace; an exported OPM_NAMESPACE makes build --offline and vet refuse every instance file in another namespace. Drop the override, or edit metadata.namespace in the instance file.

### Features

* **render:** refuse a namespace override that disagrees with the instance file ([#284](https://github.com/open-platform-model/cli/issues/284)) ([1d9f475](https://github.com/open-platform-model/cli/commit/1d9f475b2147e068faede50d0aa33c66a0c1fd74))


### Bug Fixes

* **cli:** generate the command reference and drop citations from help ([#275](https://github.com/open-platform-model/cli/issues/275)) ([8e3c15e](https://github.com/open-platform-model/cli/commit/8e3c15e4e5f9e8325dbe5eb42bfc8e78cd04e95d))
* **kubernetes:** apply instance resources by kind and weight ([#289](https://github.com/open-platform-model/cli/issues/289)) ([75e5f26](https://github.com/open-platform-model/cli/commit/75e5f2681cd619802d45716c167811c94422a416))
* never delete CRDs or Namespaces on prune or instance delete; list them as left behind ([#285](https://github.com/open-platform-model/cli/issues/285)) ([cd00874](https://github.com/open-platform-model/cli/commit/cd0087460453a798eeec309ccffaa03160dac809))


### Code Refactoring

* **config:** drop the retired k8s catalog ([#277](https://github.com/open-platform-model/cli/issues/277)) ([db4f9fd](https://github.com/open-platform-model/cli/commit/db4f9fd93be486ccfe90ffd8c415e8d1761ab3a2))

## [1.0.0-beta.5](https://github.com/open-platform-model/cli/compare/v1.0.0-beta.4...v1.0.0-beta.5) (2026-10-01)


### Features

* add apply --wait, module eval and a container image; fix health, apply and kubeconfig ([#266](https://github.com/open-platform-model/cli/issues/266)) ([f3569b2](https://github.com/open-platform-model/cli/commit/f3569b24168e7671122e23b966612301062e30b9))


### Bug Fixes

* **deps:** embed opm-operator v1.0.0-beta.4 ([#269](https://github.com/open-platform-model/cli/issues/269)) ([a43ffdb](https://github.com/open-platform-model/cli/commit/a43ffdbd50f288671e3e2decdca0e006f7479eac))


### Documentation

* **site:** add the Install the CLI page ([#268](https://github.com/open-platform-model/cli/issues/268)) ([af7b52c](https://github.com/open-platform-model/cli/commit/af7b52ca6c84febafe08e7dd5040adb61a942307))

## [1.0.0-beta.4](https://github.com/open-platform-model/cli/compare/v1.0.0-beta.3...v1.0.0-beta.4) (2026-10-01)


### Bug Fixes

* **deps:** bump golang.org/x/term to v0.46.0 ([#221](https://github.com/open-platform-model/cli/issues/221)) ([a5cd82e](https://github.com/open-platform-model/cli/commit/a5cd82ef04e867cc217bf211146f008a02fc666e))

## [1.0.0-beta.3](https://github.com/open-platform-model/cli/compare/v1.0.0-beta.2...v1.0.0-beta.3) (2026-09-30)


### Bug Fixes

* **deps:** embed opm-operator v1.0.0-beta.2 ([#258](https://github.com/open-platform-model/cli/issues/258)) ([4f17e74](https://github.com/open-platform-model/cli/commit/4f17e740a6ebebb184624261aaf71ad917fa134d))

## [1.0.0-beta.2](https://github.com/open-platform-model/cli/compare/v1.0.0-beta.1...v1.0.0-beta.2) (2026-09-30)


### Bug Fixes

* **deps:** embed opm-operator v1.0.0-beta.1 ([#256](https://github.com/open-platform-model/cli/issues/256)) ([1cb10ca](https://github.com/open-platform-model/cli/commit/1cb10ca08ce0e053c0963ad1d86e0b3b954a2c53))

## [1.0.0-beta.1](https://github.com/open-platform-model/cli/compare/v1.0.0-alpha.27...v1.0.0-beta.1) (2026-09-30)


### Bug Fixes

* **deps:** move the cli to the beta line on library v1.0.0-beta.1 ([#255](https://github.com/open-platform-model/cli/issues/255)) ([2e90c24](https://github.com/open-platform-model/cli/commit/2e90c2474564afbd2ef162280976b480a08ded5f))
* **inventory:** compare only MAJOR.MINOR in the operator ceiling ([#249](https://github.com/open-platform-model/cli/issues/249)) ([776ae91](https://github.com/open-platform-model/cli/commit/776ae91ffa55571f920d74256ef24b000df7e899))

## [1.0.0-alpha.27](https://github.com/open-platform-model/cli/compare/v1.0.0-alpha.26...v1.0.0-alpha.27) (2026-09-30)


### Bug Fixes

* **deps:** bump the templates to catalogs/opm 4.4.3 ([d5b5c01](https://github.com/open-platform-model/cli/commit/d5b5c01ad913b43c74a6501539168f488ef78980))
* **deps:** bump the templates to core v2.0.0-alpha.13 and catalogs/opm 4.4.2 ([5d372c0](https://github.com/open-platform-model/cli/commit/5d372c06cadb2261f9590c690b759e8299c48de4))


### Documentation

* **site:** adopt the hugo page dialect ([#246](https://github.com/open-platform-model/cli/issues/246)) ([7d8f44b](https://github.com/open-platform-model/cli/commit/7d8f44b68a86f86ae7a79dfd42a8236bba4e2cb8))

## [1.0.0-alpha.26](https://github.com/open-platform-model/cli/compare/v1.0.0-alpha.25...v1.0.0-alpha.26) (2026-09-30)


### Bug Fixes

* **platform:** name colliding contracts in platform check and render refusals ([#244](https://github.com/open-platform-model/cli/issues/244)) ([721c715](https://github.com/open-platform-model/cli/commit/721c715182df5b578dd047a05b066113cee9af05))

## [1.0.0-alpha.25](https://github.com/open-platform-model/cli/compare/v1.0.0-alpha.24...v1.0.0-alpha.25) (2026-09-30)


### ⚠ BREAKING CHANGES

* **platform:** platform modules passed with --platform or to opm platform check must pin opmodel.dev/core at v2.0.0-alpha.12 or later (cue mod get opmodel.dev/core@v2.0.0-alpha.12 in the module). Platforms enabling two majors of one provider catalog, or two providers of a contract whose defining catalog is disabled or absent, now fail opm platform check.

### Bug Fixes

* **deps:** bump the templates to core v2.0.0-alpha.12 and catalogs/opm 4.4.1 ([e6d9f21](https://github.com/open-platform-model/cli/commit/e6d9f210b80326454270e4de33e3325bd496173b))
* **platform:** name the registry entries providing an over-subscribed contract ([#241](https://github.com/open-platform-model/cli/issues/241)) ([4d3ace5](https://github.com/open-platform-model/cli/commit/4d3ace52a63635e38ef38215f7b200acb837bdab))

## [1.0.0-alpha.24](https://github.com/open-platform-model/cli/compare/v1.0.0-alpha.23...v1.0.0-alpha.24) (2026-09-29)


### ⚠ BREAKING CHANGES

* **platform:** renders no longer fall back to ~/.opm/platform/. Pass it with --platform ~/.opm/platform to keep rendering against it.
* **config:** config init no longer seeds a local default platform.

### Features

* **platform:** resolve instance platforms from the cluster, then their own deps ([#236](https://github.com/open-platform-model/cli/issues/236)) ([bef84a0](https://github.com/open-platform-model/cli/commit/bef84a0f3d7f4fd339d31f0d92a275e87ab2f3d5))
* **render:** add --skip-unprovided to render what the platform can ([#238](https://github.com/open-platform-model/cli/issues/238)) ([da2dc42](https://github.com/open-platform-model/cli/commit/da2dc423392c71e1908c6302b1c98051a2065220))

## [1.0.0-alpha.23](https://github.com/open-platform-model/cli/compare/v1.0.0-alpha.22...v1.0.0-alpha.23) (2026-09-29)


### ⚠ BREAKING CHANGES

* **cmd:** opm instance build <module-dir> is refused; run opm module build <module-dir> (same flags and output). opm instance build no longer accepts --name.

### Features

* **cmd:** add opm instance init ([#235](https://github.com/open-platform-model/cli/issues/235)) ([50795da](https://github.com/open-platform-model/cli/commit/50795da9873e3afe96f919de056efe2dfd220c3a))
* **cmd:** build and apply published modules; decide instance build by package kind ([#234](https://github.com/open-platform-model/cli/issues/234)) ([d221753](https://github.com/open-platform-model/cli/commit/d22175364ca95431bbce05529bc5c3135968c0e1))
* **cmd:** render module build and vet against the module's own deps ([#232](https://github.com/open-platform-model/cli/issues/232)) ([874bce8](https://github.com/open-platform-model/cli/commit/874bce8393cf9a5c1e02588fa79a2d02169a2749))

## [1.0.0-alpha.22](https://github.com/open-platform-model/cli/compare/v1.0.0-alpha.21...v1.0.0-alpha.22) (2026-09-28)


### Features

* **cmd:** add module and catalog tidy commands ([#231](https://github.com/open-platform-model/cli/issues/231)) ([e1c3122](https://github.com/open-platform-model/cli/commit/e1c3122939e593e68309bdb2e46c158960f22e21))
* **platform:** fail opm platform check on comparable transformer predicates ([#223](https://github.com/open-platform-model/cli/issues/223)) ([548f588](https://github.com/open-platform-model/cli/commit/548f5880439bcd0d2455ef41b65751203e32b767))
* **platform:** render against the cluster's effective registry and add opm platform pull ([#224](https://github.com/open-platform-model/cli/issues/224)) ([ceb1872](https://github.com/open-platform-model/cli/commit/ceb1872cb64547479113d8844dc6de48c2196d5a))
* **render:** refuse a render whose objects share one apply identity ([#226](https://github.com/open-platform-model/cli/issues/226)) ([44ad8f1](https://github.com/open-platform-model/cli/commit/44ad8f1a062ed1d615a168aa879c6ee1f00ee3bf))


### Bug Fixes

* **deps:** bump library to v1.0.0-alpha.31 ([dd10166](https://github.com/open-platform-model/cli/commit/dd10166c50c8dcedea278b61dcc302051862319c))
* **deps:** bump seeded catalog pin and template deps ([dd88a56](https://github.com/open-platform-model/cli/commit/dd88a5677529f3bad6faf1d2b823ae15f117dec0))
* **deps:** embed opm-operator v1.0.0-alpha.19 ([7cd50cb](https://github.com/open-platform-model/cli/commit/7cd50cbfbb9817a7b3993fc3c4bce84a0dc326a8)), closes [#214](https://github.com/open-platform-model/cli/issues/214)
* **instance:** read a bare argument as an instance name ([433112b](https://github.com/open-platform-model/cli/commit/433112b9697833d39a6fd71c2f141474d2f404c2))
* **templates:** make the minimal template render ([6314cd5](https://github.com/open-platform-model/cli/commit/6314cd532f0a9a6480c52fc6391ab73ad36cc371))


### Documentation

* **openspec:** allow NNNN:DN:Rn enhancement requirement citations ([2c4eead](https://github.com/open-platform-model/cli/commit/2c4eead6b56f21183cdb5688052ce3f40a99e8df))
* **site:** add the pages cli owns on the documentation site ([c792b88](https://github.com/open-platform-model/cli/commit/c792b887e6518fc8a55095ba0a13291232bafb2f))

## [1.0.0-alpha.21](https://github.com/open-platform-model/cli/compare/v1.0.0-alpha.20...v1.0.0-alpha.21) (2026-09-15)


### Features

* **platform:** add opm platform check ([#218](https://github.com/open-platform-model/cli/issues/218)) ([2bfebd9](https://github.com/open-platform-model/cli/commit/2bfebd98ef54b98fd68627a24c89a059b400616a))

## [1.0.0-alpha.20](https://github.com/open-platform-model/cli/compare/v1.0.0-alpha.19...v1.0.0-alpha.20) (2026-09-14)


### Features

* **render:** honor local-module.cue replacements on library alpha.29 ([#209](https://github.com/open-platform-model/cli/issues/209)) ([7ae324f](https://github.com/open-platform-model/cli/commit/7ae324f80b24d1ddba3a6924c9a21e5f73df79ac))


### Bug Fixes

* **deps:** bump core to v2.0.0-alpha.9, the catalogs, and library to v1.0.0-alpha.30 ([#216](https://github.com/open-platform-model/cli/issues/216)) ([85cd36a](https://github.com/open-platform-model/cli/commit/85cd36ae442cbf19e3dfe31aa66d80c2e2c29e1f))


### Code Refactoring

* **cmdutil:** acquire the instance path argument through the kernel ([#215](https://github.com/open-platform-model/cli/issues/215)) ([789496f](https://github.com/open-platform-model/cli/commit/789496fe9700165d2f50be8d81be22d1161f49b3))
* **compat:** adopt the catalog comparator from the library ([#206](https://github.com/open-platform-model/cli/issues/206)) ([99a245f](https://github.com/open-platform-model/cli/commit/99a245f87ae3248531c23ece48616912c45d7ab1))
* **config:** construct every kernel through config.NewKernel ([#212](https://github.com/open-platform-model/cli/issues/212)) ([df5b05e](https://github.com/open-platform-model/cli/commit/df5b05e19ad71c1f044bedad0f0f6ec3600337d4))
* **errors:** drop the last two unreferenced helpers ([#217](https://github.com/open-platform-model/cli/issues/217)) ([ecdc448](https://github.com/open-platform-model/cli/commit/ecdc44850dd08317417e70745808f9a11f13ea7a))
* **kernel:** migrate to library alpha.28 acquire verbs and verdict rows ([#207](https://github.com/open-platform-model/cli/issues/207)) ([7a78b88](https://github.com/open-platform-model/cli/commit/7a78b8832c8eb58ee550de20a6567bca7e0269ee))
* render-switch follow-ups from verification ([#204](https://github.com/open-platform-model/cli/issues/204)) ([31e5643](https://github.com/open-platform-model/cli/commit/31e56439193d85b8fb710e53e685596cb7f883ce))
* **render:** carry the library's metadata types and delete pkg/module ([#211](https://github.com/open-platform-model/cli/issues/211)) ([0e6a462](https://github.com/open-platform-model/cli/commit/0e6a462a21aa24ec35bb3463da44ad98a884b7ac))

## [1.0.0-alpha.19](https://github.com/open-platform-model/cli/compare/v1.0.0-alpha.18...v1.0.0-alpha.19) (2026-09-04)


### ⚠ BREAKING CHANGES

* **config:** opm config init writes ~/.opm/platform/ (cue.mod pinning core and both first-party catalogs, platform.cue importing them) instead of the data-only ~/.opm/platform.cue, and opm config vet builds that module through the kernel loader. A leftover platform.cue fails vet until 'opm config init --force' migrates it. The render path still reads the legacy file until cli-render-switch lands; both ship in one release train.

### Features

* **config:** seed the local default platform as a CUE module (0019 D5) ([#203](https://github.com/open-platform-model/cli/issues/203)) ([0263b4e](https://github.com/open-platform-model/cli/commit/0263b4e98254c141c63be09e04911cbdac87943c))


### Bug Fixes

* **deps:** bump library to v1.0.0-alpha.23 ([#199](https://github.com/open-platform-model/cli/issues/199)) ([e5deff7](https://github.com/open-platform-model/cli/commit/e5deff7561f4c05d6179623445a64e0a413bbbda))

## [1.0.0-alpha.18](https://github.com/open-platform-model/cli/compare/v1.0.0-alpha.17...v1.0.0-alpha.18) (2026-08-31)


### Bug Fixes

* **deps:** bump library to v1.0.0-alpha.22, catalog_opm to the v4 line (4.0.1) ([#197](https://github.com/open-platform-model/cli/issues/197)) ([656b835](https://github.com/open-platform-model/cli/commit/656b835d853afc6f1c8f613ef0556111055ba09d))

## [1.0.0-alpha.17](https://github.com/open-platform-model/cli/compare/v1.0.0-alpha.16...v1.0.0-alpha.17) (2026-08-31)


### ⚠ BREAKING CHANGES

* **instance:** opm instance handoff is removed with no replacement; the CLI offers no path from CLI to operator ownership. Operator-owned instances are created outside the CLI (kubectl, GitOps) and edited through the thin-editor apply path. The ownership model, render digest, and provenance annotation are unchanged.

### Features

* **instance:** remove the handoff command ([#196](https://github.com/open-platform-model/cli/issues/196)) ([7ae153f](https://github.com/open-platform-model/cli/commit/7ae153f7bd169e6e41dae0ed3e467c0c5fc74195))


### Bug Fixes

* **deps:** operator v1.0.0-alpha.14 and platform seed on catalog 2.0.0 ([#191](https://github.com/open-platform-model/cli/issues/191)) ([ff3dd0c](https://github.com/open-platform-model/cli/commit/ff3dd0c91e1a99ecd9dba9a8ad3628bfaf119003))
* **operator:** terminating guard on install and e2e applier precondition ([#193](https://github.com/open-platform-model/cli/issues/193)) ([ef714c7](https://github.com/open-platform-model/cli/commit/ef714c73ef99833020fa0ce52fb6fa181566817b))

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
