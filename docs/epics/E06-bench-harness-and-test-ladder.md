# E6: Bench harness and test ladder

parity vs openai-cli is measured across three tiers — command surface (~40%), behavior (~50%), structure (~10%) — by `biscuit bench` and published in biscuit's README, atop a graded integration suite. → [Validation strategy](../../PRD.md#validation-strategy-reverse-engineering-stainless)

## Stories

- [ ] Ship the easy/medium integration rungs: generate → build → golden requests vs mock, on every commit.
- [ ] Implement help-tree diffing of command surfaces — tier 1 (verify openai-cli's help output parses first; per-target adapter if it isn't stock cobra).
- [ ] Run golden-request comparison against openai-cli on the spec-generated mock — tier 2, the hard rung (PRs touching mapping/templates); tier 3 file-tree similarity rides the same run.
- [ ] Ship `biscuit bench --against <repo>` emitting the parity report: per-tier scores with the `--min-parity` CI ratchet and `expected: ours|theirs|either` corpus annotations; publish the dated, spec/CLI-SHA-paired scores as a per-tier bar chart in biscuit's README (SVG rendered by the bench harness itself, no Python dependency).
- [ ] Stand up the cross-generator benchmark against [fern-api/petstore-cli](https://github.com/fern-api/petstore-cli) (Fern's published CLI generator output, same petstore spec as our easy rung); Speakeasy ships no CLI generator — charted as zero, footnoted, MCP comparison deferred to E7. Score all output against the same spec-generated mock on the six absolute metrics, plus the optional read-only live-API smoke tier. → [Bench metrics](../../PRD.md#bench-metrics-cross-generator)
- [ ] Write biscuit's README quickstart and commit `examples/` (petstore-cli plus one real-world spec) as browsable generated output, leading with the biscuit-vs-Fern-vs-Speakeasy six-metric bar chart above the Stainless parity chart.
- [x] Add the passive download metrics: badgen.net badges on the README querying GitHub release-asset counts and the npm downloads API on demand — registry-side only, zero infrastructure, the binary never phones home. (A committed-JSON history workflow only if trend archaeology is ever wanted.) → [Additional design considerations](../../PRD.md#additional-design-considerations)
- [ ] Bump the README downloads badge's pinned release tag on each release (automate in the release workflow): badgen has no total-across-releases endpoint, so the badge pins one tag and the pin must move with releases to stay honest.
