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

- [x] **E1: Walking skeleton** — biscuit itself installs via Homebrew and npm and runs end-to-end, doing almost nothing yet. → [CI/CD](PRD.md#cicd), [Distribution](PRD.md#distribution) `v0.1.0-alpha.3` → [stories](docs/epics/E01-walking-skeleton.md)
- [x] **E2: Spec ingestion and IR** — a released biscuit parses any OpenAPI 3.x spec into a deterministic, immutable IR, and `biscuit doctor` grades it. `v0.1.0-alpha.4` → [Project structure](PRD.md#project-structure-the-generator), [Generation pipeline](PRD.md#generation-pipeline-and-concurrency-model), [Spec quality gate](PRD.md#spec-quality-gate-biscuit-doctor), [Spec discovery](PRD.md#spec-discovery) → [stories](docs/epics/E02-spec-ingestion-and-ir.md)
- [x] **E3: Mapping and config** — a released `biscuit generate --dry-run` prints the derived command surface for any spec, overridable via `biscuit.yaml`. `v0.1.0-alpha.5` → [Command grammar](PRD.md#command-grammar), [Argument parsing](PRD.md#argument-parsing) → [stories](docs/epics/E03-mapping-and-config.md)
- [x] **E4: Repo scaffolding** — `biscuit generate` emits a complete repo that builds and releases. `v0.1.0-alpha.7` → [Generated repo structure](PRD.md#generated-repo-structure), [Distribution](PRD.md#distribution), [Regeneration safety](PRD.md#regeneration-safety) → [stories](docs/epics/E04-repo-scaffolding.md)
- [ ] **E5: Execution layer** — generated CLIs make correct, ergonomic API calls, proven by golden requests against a spec-generated mock in a released biscuit. → [Output control](PRD.md#output-control), [API semantics](PRD.md#api-semantics-handled-automatically), [Protocol scope](PRD.md#protocol-scope), [Additional design considerations](PRD.md#additional-design-considerations) → [stories](docs/epics/E05-execution-layer.md)
- [ ] **E6: Bench harness and test ladder** — parity vs openai-cli is measured across three tiers — command surface (~40%), behavior (~50%), structure (~10%) — by `biscuit bench` and published in biscuit's README, atop a graded integration suite. → [Validation strategy](PRD.md#validation-strategy-reverse-engineering-stainless) → [stories](docs/epics/E06-bench-harness-and-test-ladder.md)

---

_MVP line — E1–E6 ship as v0.1: an installable biscuit that generates a production-ready CLI from any OpenAPI spec, verified against Stainless output with a published parity number. Migration tooling (`adopt`, the update pipeline, npm for generated CLIs) is the v0.2 arc._

**v0.1 release gates** (calendar work, runs parallel with E2–E6, not owned by any epic):

- [ ] OSS solicitor signs off GPLv2-or-later + the generated-output exception (plain GPL-2.0 LICENSE ships as fallback since v0.1.0-alpha.1) → [License](PRD.md#license)
- [ ] Decide "biscuit" trademark registration → [License](PRD.md#license)

- [ ] **E7: MCP serve** — every generated CLI is an MCP server. → [MCP subcommand](PRD.md#mcp-subcommand) → [stories](docs/epics/E07-mcp-serve.md)
- [ ] **E8: Chat TUI** — one Bubble Tea interface backs `mcp chat`, `{binary} chat`, and interactive SSE. → [Protocol scope](PRD.md#protocol-scope), [MCP subcommand](PRD.md#mcp-subcommand), [Spec discovery](PRD.md#spec-discovery) → [stories](docs/epics/E08-chat-tui.md)
- [ ] **E9: Update pipeline** — spec changes open reviewable PRs on the CLI repo automatically. → [Update pipeline](PRD.md#update-pipeline) → [stories](docs/epics/E09-update-pipeline.md)
- [ ] **E10: npm distribution for generated CLIs** — generated CLIs install via `npm`/`npx`. → [Distribution](PRD.md#distribution) → [stories](docs/epics/E10-npm-distribution.md)
- [ ] **E11: Adoption** — Stainless-generated repos migrate to biscuit in one command. → [Competitive landscape](PRD.md#competitive-landscape) → [stories](docs/epics/E11-adoption.md)
- [ ] **E12: Registry reach** — installs drop the tap prefix and trust prompt. → [Distribution](PRD.md#distribution) → [stories](docs/epics/E12-registry-reach.md)
- [ ] **E13: Doctor deepening** — every advisory speaks in generation impact, and `doctor --fix` repairs what needs no invention. Schedulable any time after E3; natural slot before E9 (the update-PR doctor delta) and E11 (`adopt` leans on the gap analysis). → [Spec quality gate](PRD.md#spec-quality-gate-biscuit-doctor) → [stories](docs/epics/E13-doctor-deepening.md)

**Future (considered, unscheduled)**: hosted generation API ([here](PRD.md#future-hosted-generation-api)); multi-API MCP toolsets + spec-acquisition skill ([here](PRD.md#future-multi-api-toolsets-and-spec-acquisition)); gRPC/proto frontend ([here](PRD.md#protocol-scope)); keychain auth UX (`auth login`, named profiles) ([here](PRD.md#additional-design-considerations)).
