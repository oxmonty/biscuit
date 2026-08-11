# E14: Rust target

`biscuit generate --language rust` emits a second production-ready CLI repo from the same IR, and TypeScript/Python are scoped from what building it costs. Scheduled after v0.2 (E9–E11): a second renderer must not compete with the migration arc. → [Target languages](../../PRD.md#target-languages)

Settled going in ([Open questions](../../PRD.md#open-questions), 2026-08): the IR needs no change — a throwaway TypeScript renderer found 0 gaps — and Rust goes first because it is the only other target preserving the static-binary [Distribution](../../PRD.md#distribution) spine. The cost is downstream of the renderer: ~3,850 LOC per target, concentrated in logic (pagination walks, multipart, SSE) rather than templates.

## Stories

- [ ] Split `internal/render` at the target boundary: extract the Go-emitting leaf functions (`flagShapes`, `securityLit`, `goStringSliceLit`, `pathExpr`, `upperSnake`/`lowerCamel`, `goExported`/`pkgName`) behind a renderer interface, leaving the view-building in `buildModel` shared. Go output must be byte-identical afterward — the golden corpus is the proof, and a refactor that churns goldens has failed.
- [ ] Add a `--language` flag and `output.language` config key, defaulting to `go`, rejecting unknown values. One target exists until the next story lands, so the flag ships *with* Rust, not before it — a flag with one valid value is a lie in `--help`.
- [ ] Guard the seam: a test in `internal/ir` that fails when a target-specific field reaches the IR. The 0-gap result is a property to keep, not a fact that stays true on its own.
- [ ] Render the Rust CLI to E4 scope (clap command tree, flags, client with path/query/body assembly, auth from the spec's security schemes) and put it under compile-the-output CI — `cargo build` on the golden repos, the Rust equivalent of the check that has caught the most.
- [ ] Carry E5's execution semantics across: pagination walks (cursor/offset/page × has_more/cursor/url/link_header/step), retries with `Retry-After`, output control, multipart file parts, SSE streaming. This is the epic's real weight — port the behavior, and reuse E5's mock and golden-request corpus so both targets are held to one contract.
- [ ] Extend the Rust repo's release pipeline to distribution parity: cross-compiled static binaries across the same 7 platforms, Homebrew cask, npm platform packages, `install.sh`, channel-aware `upgrade`. Prove the spine is genuinely target-agnostic, or find out where it was Go-shaped after all.
- [ ] Run `biscuit bench` against the Rust output and publish the number beside Go's. Tiers 1–2 are black-box, so parity should hold across targets; a gap is a renderer bug, and tier 3 (structure, ~10%) needs a Rust-shaped reference or an explicit exemption.
- [ ] Write up what the second target actually cost — LOC, wall-clock, which parts were logic vs templating, what broke — and scope TypeScript and Python against it, including the distribution design each needs (interpreter runtime vs Bun/Deno/PyInstaller bundling). The write-up is the epic's decision artifact; it settles whether a third target follows.

## Demo

`biscuit generate --spec testdata/specs/petstore.yaml --language rust --out ./petstore-cli`, then `cargo build` and run the same command sequence the Go golden repo runs, against the same mock, producing the same requests.
