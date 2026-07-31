# Changelog

## [0.1.0-alpha.8](https://github.com/oxmonty/biscuit/compare/v0.1.0-alpha.7...v0.1.0-alpha.8) (2026-07-31)


### Features

* add output control to generated CLIs ([ae41b19](https://github.com/oxmonty/biscuit/commit/ae41b1982aa206fb69dfcc06c8e03224fb2a0951))
* add retries, exit-code contract, --debug redaction to generated client ([e55af86](https://github.com/oxmonty/biscuit/commit/e55af8694f530c8af3c7871e9a87883612cce4f0))
* add transparent pagination with declared and built-in schemes ([13946e4](https://github.com/oxmonty/biscuit/commit/13946e46b15606d83a0ff52c78c1d70658f3ef1b))
* expand [@file](https://github.com/file) arguments and multipart uploads in generated CLIs ([42dcc7a](https://github.com/oxmonty/biscuit/commit/42dcc7ae87b3d5864c71c4fb8f550d7228bc36c4))
* map securitySchemes to auth flags and env vars in generated CLIs ([a114b0b](https://github.com/oxmonty/biscuit/commit/a114b0be259b01c99a7d3987206237318104a0d3))
* render spec-derived mock server and smoke suite into generated repos ([3e94f52](https://github.com/oxmonty/biscuit/commit/3e94f52e1823bfa14a104b6bceab527d1b001b5e))
* stream SSE responses as line-per-event JSONL in generated CLIs ([28818f2](https://github.com/oxmonty/biscuit/commit/28818f2d3c37ce21c9b5ddf4ba4bd37e8fb5aaa2))


### Bug Fixes

* address code-review findings on the execution layer ([4f8b68b](https://github.com/oxmonty/biscuit/commit/4f8b68b2a017e92ebe00f2a83bb42b4da867486c))
* strip query strings from path keys and drop empty-named flags ([5776d23](https://github.com/oxmonty/biscuit/commit/5776d239f57071785dadd2211898795b27199fba))

## [0.1.0-alpha.7](https://github.com/oxmonty/biscuit/compare/v0.1.0-alpha.6...v0.1.0-alpha.7) (2026-07-27)


### Bug Fixes

* **release:** write gendocs output outside dist ([f6680a8](https://github.com/oxmonty/biscuit/commit/f6680a882db405784d5f124ae39ff895c443d1c5))

## [0.1.0-alpha.6](https://github.com/oxmonty/biscuit/compare/v0.1.0-alpha.5...v0.1.0-alpha.6) (2026-07-27)


### Features

* **cli:** add channel-aware upgrade with update alias ([66d5449](https://github.com/oxmonty/biscuit/commit/66d5449b0dcd7a621fae219d5141d14204c1cfd0))
* E4 repo scaffolding — biscuit generate emits a complete repo ([48f3fc5](https://github.com/oxmonty/biscuit/commit/48f3fc521ce44e4b13d1272fd3a31fcbfc36aad3))
* **install:** verify checksums and fail clearly on bad versions ([3fa6bd9](https://github.com/oxmonty/biscuit/commit/3fa6bd9623cb0190881ac49b533bf1d8748ba27e))
* **release:** bundle man pages and shell completions into archives and casks ([b11c3d3](https://github.com/oxmonty/biscuit/commit/b11c3d3cf1477fd6baf3e03cd1fa5badb32787d5))
* **render:** emit the complete generated repo from the template tree ([34808f6](https://github.com/oxmonty/biscuit/commit/34808f6c586980ab9e1a5f15b5a6cebdbfab5c76))
* **render:** generate README, Makefile, docs bundling, and quickstart help ([0423f60](https://github.com/oxmonty/biscuit/commit/0423f604a85bd409e8c2c3dc02855081c715e179))
* **render:** SETUP.md and publishing skill create the GitHub repo ([84745fb](https://github.com/oxmonty/biscuit/commit/84745fb7a98278ad3d0ad1de1fd6bdc81405978f))
* **render:** template release mechanics into generated repos ([6e2e554](https://github.com/oxmonty/biscuit/commit/6e2e55404adb8bc0a6adc655c2aa0919c9c5c2d8))
* **render:** template upgrade, install.sh, SETUP.md, and publishing skill ([096cbdd](https://github.com/oxmonty/biscuit/commit/096cbddcaef54c6a7b1f4183d8b17e9fe0a377eb))


### Bug Fixes

* **render:** review fixes — cask man pages, npm wording, doc comments ([4890031](https://github.com/oxmonty/biscuit/commit/48900316cede8f115460a9d55a00e57e58140b7e))

## [0.1.0-alpha.5](https://github.com/oxmonty/biscuit/compare/v0.1.0-alpha.4...v0.1.0-alpha.5) (2026-07-19)


### Features

* biscuit generate --dry-run atop the plan/write split ([314ddaf](https://github.com/oxmonty/biscuit/commit/314ddafa879601024f9b82b72853c7d509e13eec))
* **cli:** biscuit init scaffolds config from doctor gap analysis ([5702f5c](https://github.com/oxmonty/biscuit/commit/5702f5c6f3e2849d0ad89b98158e0fa3fa2be3ca))
* **cli:** show a welcome splash on bare TTY invocation ([e276f87](https://github.com/oxmonty/biscuit/commit/e276f87b08e8ed045102a5052b19e1b9f7b6414d))
* **config:** schema-validated biscuit.yaml with x-biscuit-* overrides ([71e0bce](https://github.com/oxmonty/biscuit/commit/71e0bcef7e0947e9f91d6b42e85dbbcdc366dc22))
* **doctor:** humane diagnostics, folded counts, TTY colors, --format json ([d38bf9c](https://github.com/oxmonty/biscuit/commit/d38bf9c4279733ba3aa2daedebfab8af5eb9a1b3))
* E3 mapping and config — dry-run command surface for any spec ([38497c8](https://github.com/oxmonty/biscuit/commit/38497c8b3b9186a7d3ff0b3d26cc5b05fdeb6129))
* **install:** add curl installer as a third distribution channel ([d690574](https://github.com/oxmonty/biscuit/commit/d690574a94a1cd4edf0aa92af519762daaf9e4b1))
* **mapping:** derive the resource/verb command tree ([705593b](https://github.com/oxmonty/biscuit/commit/705593babf502c91038aff93efde446eec549767))
* **mapping:** flatten request schemas into static flags ([97230b1](https://github.com/oxmonty/biscuit/commit/97230b194856aa5b52e67337fdd512bc17858df7))
* **mapping:** infer oneOf discriminators via the ogen cascade ([a3915f7](https://github.com/oxmonty/biscuit/commit/a3915f703b4808b651c37837e229e9de20340922))


### Bug Fixes

* **cli:** align quickstart command columns with computed padding ([0548475](https://github.com/oxmonty/biscuit/commit/0548475d8667d85e3ef8e3a372205001177d0387))
* **cli:** let init regenerate a config that only caches spec.path ([087411d](https://github.com/oxmonty/biscuit/commit/087411db37aad69d6c5d55eb634dd5f0b15752da))
* **cli:** tighten quickstart column gap for narrow terminals ([4ce2e3d](https://github.com/oxmonty/biscuit/commit/4ce2e3d898c61f8776bd712e0ccab4e38a66d2f4))
* **doctor:** rank findings by severity, label blank severities, add summary footer ([df60050](https://github.com/oxmonty/biscuit/commit/df60050b7b03f3d6b7974b5745059776b5b83b2a))
* **mapping:** dedupe properties redeclared across allOf members ([6d87966](https://github.com/oxmonty/biscuit/commit/6d87966336d05a5175ec1718fd48edb93fa01486))
* **spec:** keep required-chain circular references advisory ([3043249](https://github.com/oxmonty/biscuit/commit/304324918cf8a8131bd87e19f4f7a53a58f52098))


### Reverts

* **cli:** drop quickstart-in-help prototype, keep it scoped for E4 ([954d8d3](https://github.com/oxmonty/biscuit/commit/954d8d38fd8c6f12e32a93a0f646c0739e4977f6))

## [0.1.0-alpha.4](https://github.com/oxmonty/biscuit/compare/v0.1.0-alpha.3...v0.1.0-alpha.4) (2026-07-17)


### Features

* add setup-publishing skill guiding the one-time release setup ([e4a069f](https://github.com/oxmonty/biscuit/commit/e4a069fff353759bc725a2f11dc4721e4c622c18))
* **doctor:** grade specs with vacuum and generation-impact notes ([7554d1e](https://github.com/oxmonty/biscuit/commit/7554d1e5aef3bd469d5e9abaf1120506bcc12880))
* **ir:** define the IR and map specs into it deterministically ([a93c035](https://github.com/oxmonty/biscuit/commit/a93c0354948497b67be67b567fbbf432d57fa8f0))
* **release:** add stable and alpha Homebrew channels mirroring npm dist-tags ([1edcafd](https://github.com/oxmonty/biscuit/commit/1edcafd54c784de5865cbc5ec6d03f9b28ff6755))
* **spec:** discover the spec when --spec is absent ([7629ae4](https://github.com/oxmonty/biscuit/commit/7629ae4eafc9b650f50fb8a20dfb147f8b4e9ba3))
* **spec:** load OpenAPI 3.x specs with the exit-code contract ([a5c249d](https://github.com/oxmonty/biscuit/commit/a5c249d922e77831a03d2e5f6ccf3d2b8cfaa5d5))


### Bug Fixes

* **lint:** check or discard writer error returns flagged by errcheck ([ca1ca7c](https://github.com/oxmonty/biscuit/commit/ca1ca7c655d89086c08332254f00a6fe2263f223))

## [0.1.0-alpha.3](https://github.com/oxmonty/biscuit/compare/v0.1.0-alpha.2...v0.1.0-alpha.3) (2026-07-15)


### Bug Fixes

* **cli:** print bare version for --version ([5d9b824](https://github.com/oxmonty/biscuit/commit/5d9b824483f5bec52a257ebb9e38d21bd98d1e71))
* **release:** tell users the installed command is biscuit in cask caveats ([0bdd249](https://github.com/oxmonty/biscuit/commit/0bdd249f33a6e6515b1d5d844b94fdb152ef1571))

## [0.1.0-alpha.2](https://github.com/oxmonty/biscuit/compare/v0.1.0-alpha.1...v0.1.0-alpha.2) (2026-07-15)


### Bug Fixes

* **npm:** publish prereleases under their prerelease dist-tag ([1632265](https://github.com/oxmonty/biscuit/commit/1632265ff1b03df19d05af26acaac759cfa3c810))
* **release:** point the cask binary stanza at the biscuit binary ([257bd59](https://github.com/oxmonty/biscuit/commit/257bd59b888e681cbdce9958ce78fb2bea1ea52d))

## 0.1.0-alpha.1 (2026-07-15)


### Features

* **npm:** move platform packages under the [@monthy](https://github.com/monthy) scope ([1a0e1ef](https://github.com/oxmonty/biscuit/commit/1a0e1efd8c45ccc5bff108054566e3a1ecc5ec3e))
* scaffold walking skeleton with release pipeline ([59eb27f](https://github.com/oxmonty/biscuit/commit/59eb27fb8e79217605f8436fc892defe52208418))


### Bug Fixes

* **ci:** run goreleaser from release-please workflow ([723d50b](https://github.com/oxmonty/biscuit/commit/723d50b4ac847abd934bb7f5880133fe5e5952e5))
