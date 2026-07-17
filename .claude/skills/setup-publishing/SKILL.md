---
name: setup-publishing
description: Step-by-step wizard for the one-time human setup that makes this repo's release pipeline publish to Homebrew and npm. Use when setting up a freshly generated (or freshly adopted) CLI repo, when a release run fails on missing secrets/permissions/trusted publishing, when the user asks how to set up brew/npm publishing, or when graduating from prerelease (alpha) to the first stable release.
---

# Set up Homebrew + npm publishing

You are walking the user through the one-time setup for this repo's release pipeline
(release-please → goreleaser → Homebrew tap cask + npm packages via OIDC trusted
publishing). Most steps are browser-only account actions you cannot perform — your job
is to verify everything that is verifiable from the CLI, hand the user exact URLs and
field values for the rest, and **gate each step with AskUserQuestion** before moving on.

## Ground rules

- Work the steps in order; each builds on the previous one.
- After presenting a manual step, fire AskUserQuestion: "Have you completed <step>?"
  with options like `Done`, `Skip (already set up)`, `Help — it didn't work`. Do not
  proceed past a step until the user picks Done or Skip. On Help, debug with them
  before continuing.
- Verify before asking: if a CLI check can prove a step is already done, run it and
  skip the question entirely.
- Never ask the user to paste tokens into the chat; they go into `gh secret set`
  prompts or browser forms directly.

## Step 0 — Read the repo's publishing config

Derive every name from the repo instead of assuming:

- `.goreleaser.yml` → cask name, tap `repository.owner`/`name`, token env var name,
  binary name.
- `npm/*/package.json` → the shim package name; the platform-package scope/prefix is in
  `scripts/publish_npm.mjs` (or the npm publish script this repo uses).
- `.github/workflows/` → the release workflow **filename** (trusted publishing matches
  it exactly, extension included).
- `git remote get-url origin` → GitHub org and repo.

## Step 1 — GitHub org allows Actions to create PRs

release-please opens release PRs from a workflow; new orgs block this by default.

- Ask the user to enable: `https://github.com/organizations/<org>/settings/actions`
  → Workflow permissions → check "Allow GitHub Actions to create and approve pull
  requests" (repo-level setting too if overridden).
- Failure signature if skipped: release-please run fails with
  "GitHub Actions is not permitted to create or approve pull requests."
- Gate with AskUserQuestion.

## Step 2 — Homebrew tap repo + token

1. Tap repo: verify with `gh api repos/<org>/<tap-repo> --jq .private` (must exist and
   be **public**; a private tap breaks installs). If missing and the user's `gh` has
   rights, offer to run `gh repo create <org>/<tap-repo> --public`.
2. PAT (browser-only): user creates a fine-grained token at
   `https://github.com/settings/personal-access-tokens/new` — resource owner `<org>`,
   repository access ONLY the tap repo, permissions **Contents: read and write**,
   nothing else. Must be created by an account that is a member of the org.
3. Secret: user runs `gh secret set <TOKEN_ENV_VAR> --org <org> --visibility all`
   (needs `admin:org` scope — `gh auth refresh -s admin:org` — or use the org settings
   UI). On a free org plan, org secrets only reach **public** repos; this repo must be
   public anyway for release downloads to work.
4. Gate with AskUserQuestion.

## Step 3 — npm account, 2FA, and org

- `npm whoami` verifies login; `npm login` if not.
- 2FA is **required to publish**; without it publishes fail with a hard 403
  ("Two-factor authentication ... is required"). User enables it at
  `https://www.npmjs.com/settings/~/tfa` (passkey or authenticator app).
- If the platform packages are scoped (`@<scope>/...`), the scope's org must exist:
  `https://www.npmjs.com/org/create`. Check availability first:
  `curl -s https://registry.npmjs.org/-/user/org.couchdb.user:<scope>`.
- Gate with AskUserQuestion.

## Step 4 — First release and bootstrap npm publish

npm trusted publishing can only be configured on packages that already exist, so the
first npm publish is local; Homebrew needs no bootstrap.

1. Merge the first release-please PR (the user merges, or asks you to). The release
   run publishes the GitHub release + cask; **the npm step failing is expected**.
2. Bootstrap publish from the tag (run these with the user, OTP prompts are theirs):
   ```sh
   git fetch --tags && git checkout <tag>
   goreleaser release --clean --skip=publish,announce   # or: go run github.com/goreleaser/goreleaser/v2@latest ...
   node scripts/publish_npm.mjs <version>
   git checkout -
   ```
3. Note: npm auto-points `latest` at a package's very first publish even for
   prereleases; verify with `npm view <shim-package> dist-tags`.
4. Gate with AskUserQuestion.

## Step 5 — npm trusted publishing (OIDC)

For **each** published package (shim + every platform package), the user fills the
form at `https://www.npmjs.com/package/<name>/settings` → Trusted Publisher:

- Publisher: GitHub Actions
- Organization or user: `<org>` / Repository: `<repo>`
- Workflow filename: exactly the release workflow filename from Step 0
- Environment: leave blank
- Allowed actions: check **Allow npm publish** only

Also under Publishing access, recommend "Require two-factor authentication and
disallow tokens" — trusted publishers are unaffected and token attacks go dead.
List every package URL for the user, then gate with AskUserQuestion.

## Step 6 — Verify end to end

- `brew install <org>/<tap>/<cask-name>` (fully qualified — it bypasses Homebrew 6
  tap-trust; the bare name after `brew tap` requires `brew trust`), then run the
  binary with `--version`.
- `npx <shim-package> version` (or `npm install -g`).
- Optionally cut the next release and confirm the npm job publishes via OIDC with no
  credentials.

Close by summarizing what is now automated (every future release) versus what was
one-time (everything above).

## Later — graduating from prerelease to stable

Not part of setup; do this when the user decides the first stable release is ready.
release-please never graduates on its own — with prerelease versioning enabled it
increments the prerelease counter forever (alpha.4, alpha.5, ...).

1. Remove `"prerelease": true` (and any `"prerelease-type"`) from
   `release-please-config.json` and commit. The next release PR proposes the stable
   version. To force a specific version instead, use a commit footer:
   `Release-As: 0.1.0`.
2. Everything downstream keys off the version string — no other changes needed:
   - goreleaser `release.prerelease: auto` stops marking GitHub releases prerelease.
   - The stable cask (`skip_upload: auto`) resumes updating; the `@next` cask keeps
     tracking every release.
   - The npm publish script keys the dist-tag off the version: any prerelease
     suffix → `next`, no suffix → `latest`. Manual `npm dist-tag add` promotions stop.
3. While still pre-stable, remind the user that npm `latest` only moves manually:
   `npm dist-tag add <shim-package>@<version> latest` after each prerelease they
   want promoted.
