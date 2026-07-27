# E4: Repo scaffolding — write-up

2026-07-27

## What shipped

`v0.1.0-alpha.7`: `biscuit generate` stopped being a dry-run printer and became the product — one command turns any OpenAPI 3.x spec into a complete CLI repository that compiles immediately, makes real API calls, and ships itself.

For a developer running biscuit: the generated repo builds with zero fixup (`go build ./...`, no `go mod tidy` — go.sum ships pinned), the command tree carries typed flags with dot-notation bodies, and regeneration is safe by construction — every generated file carries a `DO NOT EDIT` marker, `internal/custom/` is emitted once and never overwritten, and stripping a marker takes ownership of a file permanently. The client's exported per-operation signatures are the frozen contract custom code may depend on; biscuit's own CI compile-checks that contract on every commit.

For the end user of a generated CLI: `brew install` (two-channel casks with man pages and shell completions installed), `curl | sh` (opt-in hardened installer), `{binary} upgrade` that knows how it was installed and upgrades through the right channel, quickstart help on bare invocation, and a README documenting the Homebrew 6 tap-trust step. The generated SETUP.md and bundled `.claude/skills/setup-publishing` wizard cover the one-time human publishing setup — including creating the GitHub repo itself.

Every mechanic templated into generated repos was proven on biscuit's own release pipeline first: this release is the first where `man biscuit` works straight from the cask.

## Try it yourself

```sh
brew upgrade biscuit-cli@next        # or: npx biscuit-cli@next --version  → 0.1.0-alpha.7
man biscuit                          # first release with man pages in the cask

cd $(mktemp -d)
curl -fsSLo openapi.yaml https://raw.githubusercontent.com/oxmonty/biscuit/main/testdata/specs/petstore.yaml
biscuit generate                     # discovers the spec, writes ./swagger-petstore-cli
cd swagger-petstore-cli && go build ./... && make help
go run ./cmd/swagger-petstore        # bare invocation → quickstart
go run ./cmd/swagger-petstore pets list --limit 5 --base-url https://httpbin.org/anything
```

The last command sends a real `GET /anything?limit=5` and prints the response — a working CLI from a spec in under a minute. Edit `internal/custom/`, rerun `biscuit generate --out .`, and watch your file survive.

## Validation notes

- `go test ./...` — the whole suite in one command (~35 s): golden repos (petstore + galaxy, `-update` to refresh), FilePlan sha256 pins for the five big specs, regeneration-safety and marker tests, 11 upgrade tests, install.sh's offline shell suite, compile-the-output on all seven ladder specs (galaxy's golden repo carries a hand-written `internal/custom/` fixture; openai's ephemeral build injects one), and the e2e that generates, builds, and runs a CLI against a live server.
- `golangci-lint run` — 0 issues, and generated output itself lints clean.
- `go test ./internal/render/ -bench BenchmarkRender -run XXX` — budget: openai ≤ 5 s.
- `sh scripts/test_install.sh` — the installer suite standalone (also rides go test).

## Evidence

- Release run 30276702325 green; 8 assets + checksums on [v0.1.0-alpha.7](https://github.com/oxmonty/biscuit/releases/tag/v0.1.0-alpha.7); darwin-arm64 archive verified to contain `man/*.1` and 4 completion scripts; `biscuit-cli@next` cask at 0.1.0-alpha.7 with `manpage`/`*_completion` stanzas.
- [PR #8](https://github.com/oxmonty/biscuit/pull/8) (10 commits, CI green) + release-fix commit `f6680a8` direct to main.
- Two-axis review (standards + spec) ran pre-merge; all four actionable findings fixed in `4890031` (nested cask man pages, npm doc wording, kebab doc comments) with the brew-graduation deferral recorded as a `ponytail:` ceiling.
- Regression exit audit (`c564dd4`): every kickoff-named check built and observed failing once before being trusted.
- Bench baseline (Apple M2, parse→render→FilePlan): petstore 73 ms, train-travel 94 ms, openai 1.23 s, stripe 2.45 s — all inside the 5 s budget; the numbers later epics regress against.
- Live e2e evidence: generated petstore CLI sent `GET /pets?limit=5`, substituted `/pets/42`, enforced required flags, and relayed response bodies against a local echo server.

## Decisions made along the way

- **E4 client scope: naive HTTP client with final signatures** (decision log). Per-operation methods over plain `net/http`; E5 layers auth/retries/pagination behind unchanged signatures. Stub-only rejected (the custom-contract gate needs a real surface and an erroring CLI demos nothing); deferring the contract rejected (ships regeneration safety half-done).
- **Golden corpus: petstore + galaxy committed; big specs hash-pinned + ephemerally built** (decision log). Whole-ladder commits rejected for repo bloat and unreviewable diffs. Mid-epic user steer confirmed the openai shape: generate the entire repo in CI, build it, throw it away.
- **Repo creation delegated to `gh`** (decision log): no native `--push`/`--url`; SETUP.md opens with `gh repo create --source=. --push` and the setup-publishing skill runs it when no origin exists. Generation stays pure.
- **Release-please manifest is emit-once** — regenerating a released repo must never reset `.release-please-manifest.json` to 0.0.0 (caught reviewing a subagent's report, fixed before merge).
- **Markers double as an ownership protocol**: overwrite only when the existing file still carries the marker; both-sides-unmarked (go.sum, JSON) stays machine-owned. Falls out of one write-time rule, no per-file config.
- **`imports.Process` over conditional template imports**: templates carry a superset import list, goimports prunes — killed a whole class of template bugs for one sanctioned dependency (the PRD already named the goimports pass).

## What surprised

- **The ladder caught four naming bugs in one afternoon**: pokeapi's `type` resource became a Go-keyword import alias; openai operationIds leaked `?` into identifiers; PokéAPI's é broke the derived module path (Go module paths are ASCII-only); pokeapi's `version` resource was shadowed by `NewRootCmd`'s `version` parameter. Each produced a reserved-word or sanitization rule plus a regression test.
- **goreleaser before-hooks may not write into `dist/`** — it re-checks dist is empty *after* hooks run, which failed the alpha.6 release outright. Docs now render to `docsgen/` (gitignored). Release-please had already tagged, leaving an asset-less release that would have 404'd the `next` channel; alpha.6 was deleted (release + tag) after alpha.7 superseded it.
- **The release cache "fix" from E2 was never really tested**: caches evict after 7 idle days, and the alpha cadence crossed the line — alpha.7 ran cold (~28 min) and only *saved* the cache. Recorded in the PRD with the weekly-refresh upgrade path.
- **The manual E3 bench preview paid off again in reverse**: the spec-axis review caught that cask manpage lists covered only top-level commands — `man galaxy-auth-token` would have silently 404'd post-install. The list is now computed recursively from the model.
- **A subagent's honest self-report beat the diff**: the release-templates agent flagged its own manifest-reset hazard in its report, which is what turned into the emit-once fix.

## Left open

- Brew graduation (announced `@next`→stable cask swap when stable overtakes prereleases) is a `ponytail:` ceiling in both upgrade implementations — implement when the first stable release nears.
- The generated e2e asserts request wiring against an echo server; schema-valid response data arrives with E5's spec-generated mock, which replaces the echo server in the same test.
- Cold-release duration (~28 min on >7-day gaps) — weekly cache-refresh job if the cadence stays slow.
- npm distribution for generated CLIs is E10; generated SETUP.md's npm section is expressly forward-looking until then.
