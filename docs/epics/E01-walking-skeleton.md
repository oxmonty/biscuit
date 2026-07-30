# E1: Walking skeleton

biscuit itself installs via Homebrew and npm and runs end-to-end, doing almost nothing yet. → [CI/CD](../../PRD.md#cicd), [Distribution](../../PRD.md#distribution) `v0.1.0-alpha.3`

Write-up: [docs/write-ups/E1-walking-skeleton.md](../write-ups/E1-walking-skeleton.md)

## Stories

- [x] Scaffold the generator repo: module layout, cobra root, `biscuit version` and `--help` + CLAUDE.md file
- [x] Wire CI and releases: release-please + goreleaser cross-platform builds to GitHub Releases.
- [x] Publish the Homebrew tap so `brew install` works.
- [x] Publish `biscuit-cli` to npm (shim + platform optionalDependencies) so `npx biscuit-cli` works.
- [x] _(Same mechanics later templated into generated CLIs in E4/E10 — this epic proves them on biscuit itself.)_
