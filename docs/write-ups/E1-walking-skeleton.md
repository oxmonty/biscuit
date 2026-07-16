# E1 — Walking skeleton

_Completed 2026-07; written up 2026-07-16. Append-only narrative — the spec lives in PRD.md, current state in ROADMAP.md._

## What shipped

`v0.1.0-alpha.3`: biscuit installs through every planned channel and runs end-to-end while doing almost nothing.

- Repo scaffold: Go module, cobra root, `biscuit version` / `--help`, CLAUDE.md, npm shim package layout under `npm/biscuit-cli/`.
- Release pipeline: Conventional Commits → release-please release PR → merge tags `vX.Y.Z-alpha.N` → goreleaser builds darwin/linux/windows (arm64+amd64) → GitHub Release.
- Homebrew: `oxmonty/homebrew-tap` with two casks — stable `biscuit-cli` (`skip_upload: auto`, untouched by prereleases) and `biscuit-cli@alpha` (tracks every release).
- npm: `biscuit-cli` shim + `@oxmonty/biscuit-<platform>-<arch>` platform packages via `optionalDependencies`, published by `scripts/publish_npm.mjs` under the `alpha` dist-tag, OIDC trusted publishing (zero npm tokens).
- `.claude/skills/setup-publishing`: agent-guided wizard for the one-time human setup, including the later alpha→stable graduation steps.

## Evidence

- `brew install oxmonty/tap/biscuit-cli@alpha` → `biscuit version` prints the release version.
- `npx biscuit-cli version` resolves the platform package and runs.
- Release run publishes GitHub Release + casks + npm packages with no manual steps and no stored npm credentials.

## Decisions made along the way

- **`biscuit-cli` on both registries, command stays `biscuit`** — bare `biscuit` is npm-squatted and a browser cask in homebrew/cask; misspellings were rejected outright. Dispute/homebrew-core routes deferred to E12.
- **Two-channel prerelease policy** — stable cask + `@alpha` cask mirroring npm `latest` + `alpha` dist-tags, so prereleases never contaminate the stable channel. Graduation is a deliberate switch: remove `"prerelease": true` from `release-please-config.json`; everything downstream keys off the version string.
- **OIDC-only npm publishing** — no long-lived tokens anywhere; org-level `HOMEBREW_TAP_TOKEN` fine-grained PAT scoped to contents-write on the tap repo only.

## Surprises

- npm points `latest` at a package's **first** publish even for a prerelease — the manual `npm dist-tag add` promotion dance exists until the first stable release.
- A brand-new npm package cannot use OIDC on its first publish; the bootstrap publish is local (`npm login` + `scripts/publish_npm.mjs`), then the trusted publisher gets configured per package.
- New GitHub orgs block Actions from opening PRs by default — release-please fails until "Allow GitHub Actions to create and approve pull requests" is enabled org-side.
- `main` later gained a changes-must-be-PRs branch rule; direct pushes currently bypass it with a warning.

## What this proved

The exact release mechanics (goreleaser + release-please + tap casks + npm shim/platform packages + OIDC) that E4/E10 template into generated repos, exercised end-to-end on biscuit itself before any generator feature existed.
