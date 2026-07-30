# E9: Update pipeline

spec changes open reviewable PRs on the CLI repo automatically. → [Update pipeline](../../PRD.md#update-pipeline)

## Stories

- [ ] Ship the pull-topology workflow with `.biscuit-state.yml` and App-token PRs.
- [ ] Classify spec diffs into semver bumps feeding release-please.
- [ ] Make `biscuit generate` fetch a remote `spec.source` before regenerating — the CI loop run locally, no separate update verb (`fern generate`/`speakeasy run` precedent); the templated workflow invokes the same command.
- [ ] Ship the biscuit-upgrade PR flow so tool bumps arrive as a separate PR species from spec updates.
- [ ] Document the push topology (`repository_dispatch`) as a snippet.
