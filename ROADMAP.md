# biscuit

- Repo: https://github.com/oxmonty/biscuit.git
- Design: [PRD.md](PRD.md) — every epic links into it; completed epics get a write-up in [docs/write-ups/](docs/write-ups/)

> A Go package and CLI that converts an OpenAPI 3.x spec into a complete, production-ready CLI repository (`{project}-cli`) — the open, self-hostable alternative to the wound-down Stainless CLI generator and today's account-gated successors (Speakeasy, Fern).

```
biscuit generate --spec openapi.yaml --config biscuit.yaml --out ./foo-cli
```

Usable two ways:

- **Library**: `import "github.com/oxmonty/biscuit"` → `biscuit.Generate(ctx, spec, cfg)` returns a file plan
- **CLI**: `biscuit generate | doctor | bench | init | upgrade`

---

## Roadmap

- [x] **E1: Walking skeleton** — biscuit itself installs via Homebrew and npm and runs end-to-end, doing almost nothing yet. → [CI/CD](PRD.md#cicd), [Distribution](PRD.md#distribution) `v0.1.0-alpha.3`
    - [x] Scaffold the generator repo: module layout, cobra root, `biscuit version` and `--help` + CLAUDE.md file
    - [x] Wire CI and releases: release-please + goreleaser cross-platform builds to GitHub Releases.
    - [x] Publish the Homebrew tap so `brew install` works.
    - [x] Publish `biscuit-cli` to npm (shim + platform optionalDependencies) so `npx biscuit-cli` works.
    - [x] _(Same mechanics later templated into generated CLIs in E4/E10 — this epic proves them on biscuit itself.)_
- [x] **E2: Spec ingestion and IR** — a released biscuit parses any OpenAPI 3.x spec into a deterministic, immutable IR, and `biscuit doctor` grades it. `v0.1.0-alpha.4` → [Project structure](PRD.md#project-structure-the-generator), [Generation pipeline](PRD.md#generation-pipeline-and-concurrency-model), [Spec quality gate](PRD.md#spec-quality-gate-biscuit-doctor), [Spec discovery](PRD.md#spec-discovery)
    - [x] Spike `pb33f/libopenapi` vs `speakeasy-api/openapi` in `spike/`, both parsing openai.yaml, scored against defined metrics (cycle-safe `$ref` resolution, 3.0/3.1 handling, parse time/memory, API ergonomics, governance/bus-factor); the winner and its linter sibling become the parser and doctor engine.
    - [x] Parse and validate specs with the spike-chosen parser, resolving `$ref`s cycle-safely, with biscuit's own exit-code contract so scripts and pipelines get predictable failures.
    - [x] Make `--spec` optional: discover the spec by well-known names (`openapi|swagger.{yaml,yml,json}`) in the current directory (flat scan — deeper enumeration ships with E8's discovery UX), then content-sniff its remaining yaml/json (first ~1 KB) for an `openapi:` root key; on multiple matches list candidates and prompt (plain stderr); persist the choice to `spec.path` in `biscuit.yaml` so discovery runs once — the config is the cache.
    - [x] Define IR types with all collections sorted at mapping time, normalizing 3.0 and 3.1 (`nullable` vs `type` arrays, `example` vs `examples`) into one shape.
    - [x] Integrate the spike-chosen linter (vacuum or Speakeasy's) as `biscuit doctor`: blocking correctness errors, advisory quality report with generation-impact notes, `--strict` / `lint.min_grade` gate.
    - [x] Seed `testdata/specs` as a graded ladder: petstore (easy), a mid-size real-world 3.1 spec with oneOf/multi-auth/SSE (medium, e.g. Train Travel API), openai.yaml (hard), plus pathological cases including cyclic `$ref`s.
    - [x] Add the generation benchmark (`gen_bench_test.go`) from day one.
- [ ] **E3: Mapping and config** — a released `biscuit generate --dry-run` prints the derived command surface for any spec, overridable via `biscuit.yaml`. → [Command grammar](PRD.md#command-grammar), [Argument parsing](PRD.md#argument-parsing)
    - [x] Derive the resource/verb tree from tags and paths, including nested sub-resources and stutter removal.
    - [x] Add [stripe/openapi](https://github.com/stripe/openapi) to `testdata/specs` as the tree-derivation stress test: a large real-world 3.x spec with deeply nested resources and polymorphic `oneOf` on nearly every object, a distinct shape from openai.yaml.
    - [x] Implement flag flattening with the schema-adaptive dot-notation depth policy, cycle detection, and a hard depth bound.
    - [x] Implement the oneOf discriminator-inference cascade.
    - [x] Load and apply `biscuit.yaml` overrides (names, aliases, hidden endpoints, pagination hints), validated against a schema: unknown keys rejected with precise errors, `version` key for forward migration — plus the in-spec `x-biscuit-*` mirror set (name, group, ignore, pagination hints) feeding the same override struct, sidecar winning on conflict.
    - [x] Ship `biscuit init`: scaffold a starter `biscuit.yaml` seeded from `doctor`'s gap analysis.
    - [x] Ship `biscuit generate --dry-run` printing the derived resource/verb tree and the FilePlan — free from the plan/write split, and E3's demo.
    - [x] Polish doctor output: humane one-line resolver diagnostics (no raw rolodex dumps), finding counts folded into the impact phrasing ("718 sites weaken the mock corpus"), severity colors on TTY, and `doctor --format json` for CI pipelines.
- [ ] **E4: Repo scaffolding** — `biscuit generate` emits a complete repo that builds and releases. → [Generated repo structure](PRD.md#generated-repo-structure), [Distribution](PRD.md#distribution), [Regeneration safety](PRD.md#regeneration-safety)
    - [ ] Render the full template tree with generated-file markers and `internal/custom/`, defining the stable surface custom code may depend on.
    - [ ] Emit goreleaser, release-please, and Homebrew tap configuration (proven in E1), including the two-channel prerelease policy: stable cask with `skip_upload: auto` + `{name}@next` cask mirroring the npm `next` dist-tag — one prerelease channel for any maturity (alpha/beta/rc live in the version string) — and the release job's dedicated cross-compile build cache (proven on biscuit; see [CI/CD](PRD.md#cicd)).
    - [ ] Generate README (documenting the Homebrew 6 tap-trust step), shell completions (bash/zsh/fish/PowerShell), and man pages.
    - [ ] Ship biscuit's own man pages and packaged shell completions (cobra's generators, bundled into release archives and casks so `man biscuit` works post-install) — proving the mechanics the story above templates into generated repos.
    - [ ] Template the Makefile into generated repos (proven on biscuit's own): sectioned awk help headed by the binary name with the description on the line beneath it, sourced from the spec (`info.title`/`info.description` via the IR, `biscuit.yaml` overrides winning, lines omitted when the spec has neither), with build/run/check/lint/bench/snapshot/gacp targets mapped to the generated project.
    - [ ] Ship channel-aware `biscuit upgrade` (alias `update` — synonyms in the wild) and template the same command into generated CLIs: detect the install channel (brew/npm/bare binary) and release channel (stable vs next), exec the package manager's own upgrade, self-swap only bare binaries; `--channel` and `--version` for explicit channel crossing and exact pins. → [Distribution](PRD.md#distribution)
    - [ ] Harden biscuit's own `install.sh` ahead of templating: checksum verification against the release's `checksums.txt`, a clear error on a bad `--version`, and a `--binary <path>` local-install override for testing.
    - [ ] Template `install.sh` (proven on biscuit's own, adapted from [opencode's](https://github.com/sst/opencode), MIT) into generated repos as a third distribution channel alongside brew/npm — installs to `~/.{binary}/bin`, `--channel`/`--version` flags mirroring `upgrade`'s, invoked as `curl -fsSL https://raw.githubusercontent.com/<org>/<repo>/main/install.sh | sh`. → [Distribution](PRD.md#distribution)
    - [ ] Add the quickstart-in-help pattern to generated CLIs via `root.Long`: spec-derived description (same `biscuit.yaml`-overrides-first rule as the Makefile) plus quickstart lines specific to the generated command tree, shown on `{binary} help`/`--help`/`-h` and bare invocation alike through cobra's default template. → [Distribution](PRD.md#distribution)
    - [ ] Generate SETUP.md documenting the one-time human publishing steps proven on biscuit itself: tap repo + contents-write PAT, org "allow Actions to create PRs" setting, npm 2FA, first-publish-is-local (OIDC needs an existing package), `npm trust` for the trusted-publisher config.
    - [ ] Template a Claude Code skill (.claude/skills/setup-publishing) into generated repos: agent verifies the checkable setup via gh/npm (tap repo, secrets, org Actions setting, trusted publishers), runs the local bootstrap publish, and hands the user only the true browser steps with exact URLs and field values. SETUP.md stays as the human-readable fallback. (v1 lives in biscuit's own .claude/skills/setup-publishing — template it from there.)
    - [ ] Add compile-the-output CI (`go build ./...` on generated golden repos), including one repo with a representative `internal/custom/` file so contract drift fails in biscuit's CI.
- [ ] **E5: Execution layer** — generated CLIs make correct, ergonomic API calls, proven by golden requests against a spec-generated mock in a released biscuit. → [Output control](PRD.md#output-control), [API semantics](PRD.md#api-semantics-handled-automatically), [Protocol scope](PRD.md#protocol-scope), [Additional design considerations](PRD.md#additional-design-considerations)
    - [ ] Generate a mock server from any spec (routes + schema-valid canned responses + request recording) — shared by the golden harness, the parity bench, and the smoke tests templated into generated repos.
    - [ ] Map `securitySchemes` to auth flags and env vars.
    - [ ] Ship `--format` (incl. jsonl) and `--transform`/`--transform-error` via gjson.
    - [ ] Implement `@file` argument handling with sniffing and explicit prefixes.
    - [ ] Implement pagination (`--all`/`--max-pages` or transparent — see Open questions) and stream SSE responses as JSONL when piped.
    - [ ] Add retries/backoff with `Retry-After`, the exit-code contract, and `--debug` with secret redaction.
- [ ] **E6: Bench harness and test ladder** — parity vs openai-cli is measured across three tiers — command surface (~40%), behavior (~50%), structure (~10%) — by `biscuit bench` and published in biscuit's README, atop a graded integration suite. → [Validation strategy](PRD.md#validation-strategy-reverse-engineering-stainless)
    - [ ] Ship the easy/medium integration rungs: generate → build → golden requests vs mock, on every commit.
    - [ ] Implement help-tree diffing of command surfaces — tier 1 (verify openai-cli's help output parses first; per-target adapter if it isn't stock cobra).
    - [ ] Run golden-request comparison against openai-cli on the spec-generated mock — tier 2, the hard rung (PRs touching mapping/templates); tier 3 file-tree similarity rides the same run.
    - [ ] Ship `biscuit bench --against <repo>` emitting the parity report: per-tier scores with the `--min-parity` CI ratchet and `expected: ours|theirs|either` corpus annotations; publish the dated, spec/CLI-SHA-paired scores as a per-tier bar chart in biscuit's README (SVG rendered by the bench harness itself, no Python dependency).
    - [ ] Stand up the cross-generator benchmark against [fern-api/petstore-cli](https://github.com/fern-api/petstore-cli) (Fern's published CLI generator output, same petstore spec as our easy rung); Speakeasy ships no CLI generator — charted as zero, footnoted, MCP comparison deferred to E7. Score all output against the same spec-generated mock on the six absolute metrics, plus the optional read-only live-API smoke tier. → [Bench metrics](PRD.md#bench-metrics-cross-generator)
    - [ ] Write biscuit's README quickstart and commit `examples/` (petstore-cli plus one real-world spec) as browsable generated output, leading with the biscuit-vs-Fern-vs-Speakeasy six-metric bar chart above the Stainless parity chart.
    - [x] Add the passive download metrics: shields.io badges on the README querying GitHub release-asset counts and the npm downloads API on demand — registry-side only, zero infrastructure, the binary never phones home. (A committed-JSON history workflow only if trend archaeology is ever wanted.) → [Additional design considerations](PRD.md#additional-design-considerations)

---

_MVP line — E1–E6 ship as v0.1: an installable biscuit that generates a production-ready CLI from any OpenAPI spec, verified against Stainless output with a published parity number. Migration tooling (`adopt`, the update pipeline, npm for generated CLIs) is the v0.2 arc._

**v0.1 release gates** (calendar work, runs parallel with E2–E6, not owned by any epic):

- [ ] OSS solicitor signs off GPLv2-or-later + the generated-output exception (plain GPL-2.0 LICENSE ships as fallback since v0.1.0-alpha.1) → [License](PRD.md#license)
- [ ] Decide "biscuit" trademark registration → [License](PRD.md#license)

- [ ] **E7: MCP serve** — every generated CLI is an MCP server. → [MCP subcommand](PRD.md#mcp-subcommand)
    - [ ] Map operations to MCP tools and serve over stdio, then Streamable HTTP, on the official `modelcontextprotocol/go-sdk`, pinning the targeted MCP protocol revision.
    - [ ] Template a project-scope `.mcp.json` into generated repos so opening one in Claude Code wires the CLI's tools automatically — zero-command team onboarding. → [MCP subcommand](PRD.md#mcp-subcommand)
- [ ] **E8: Chat TUI** — one Bubble Tea interface backs `mcp chat`, `{binary} chat`, and interactive SSE. → [Protocol scope](PRD.md#protocol-scope), [MCP subcommand](PRD.md#mcp-subcommand), [Spec discovery](PRD.md#spec-discovery)
    - [ ] Spike the MCP-client integration that the chat strategy leans on: drive a generated `{binary} mcp serve` from Claude Code, Warp, and [pi](https://github.com/earendil-works/pi) end to end (tool discovery, streaming, env auth, stdio and Streamable HTTP) — rich chat UX belongs to these clients, not an owned TUI; if they can't carry it, the pi-port question reopens with evidence. → [MCP subcommand](PRD.md#mcp-subcommand)
    - [ ] Build the minimal built-in TUI — serviceable, not spectacular — with streaming and tool-call display, stealing the UX decisions of [pi](https://github.com/earendil-works/pi) on Bubble Tea.
    - [ ] Add Anthropic and OpenAI providers behind a two-provider interface.
    - [ ] Detect chat-shaped endpoints and emit the `{binary} chat` REPL.
    - [ ] Route interactive-TTY SSE responses into the TUI.
    - [ ] Upgrade spec discovery to the full UX: git-index enumeration with the gitignore-blind `WalkDir` fallback, the delayed stderr spinner, and a Bubble Tea countdown selector auto-picking the best-ranked candidate (non-TTY prints its pick).
- [ ] **E9: Update pipeline** — spec changes open reviewable PRs on the CLI repo automatically. → [Update pipeline](PRD.md#update-pipeline)
    - [ ] Ship the pull-topology workflow with `.biscuit-state.yml` and App-token PRs.
    - [ ] Classify spec diffs into semver bumps feeding release-please.
    - [ ] Make `biscuit generate` fetch a remote `spec.source` before regenerating — the CI loop run locally, no separate update verb (`fern generate`/`speakeasy run` precedent); the templated workflow invokes the same command.
    - [ ] Ship the biscuit-upgrade PR flow so tool bumps arrive as a separate PR species from spec updates.
    - [ ] Document the push topology (`repository_dispatch`) as a snippet.
- [ ] **E10: npm distribution for generated CLIs** — generated CLIs install via `npm`/`npx`. → [Distribution](PRD.md#distribution)
    - [ ] Template the shim, per-platform packages, and ordered OIDC publish job, with prereleases published under the `next` dist-tag (never `latest`).
- [ ] **E11: Adoption** — Stainless-generated repos migrate to biscuit in one command. → [Competitive landscape](PRD.md#competitive-landscape)
    - [ ] Ship `biscuit adopt --repo --spec` proposing a parity-maximizing config and taking over the release pipeline.
- [ ] **E12: Registry reach** — installs drop the tap prefix and trust prompt. → [Distribution](PRD.md#distribution)
    - [ ] Submit biscuit-cli to homebrew/core (or homebrew/cask) once notability criteria are met; revisit the npm bare-name dispute at the same time.

**Future (considered, unscheduled)**: hosted generation API ([here](PRD.md#future-hosted-generation-api)); gRPC/proto frontend ([here](PRD.md#protocol-scope)); keychain auth UX (`auth login`, named profiles) ([here](PRD.md#additional-design-considerations)).
