# Changelog

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
