# Changelog

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
