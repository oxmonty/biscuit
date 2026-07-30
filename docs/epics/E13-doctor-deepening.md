# E13: Doctor deepening

every advisory speaks in generation impact, and `doctor --fix` repairs what needs no invention. Schedulable any time after E3; natural slot before E9 (the update-PR doctor delta) and E11 (`adopt` leans on the gap analysis). → [Spec quality gate](../../PRD.md#spec-quality-gate-biscuit-doctor)

## Stories

- [ ] Tune the advisory set on the test-ladder specs and give every surviving biscuit-ruleset rule an impact + remediation template (today only six rules carry one; the rest print raw counts). A rule that can't be phrased as generation impact is noise — drop it from the ruleset rather than print it.
- [ ] Implement `lint.ruleset` (custom vacuum ruleset file), the documented-but-unbuilt config key.
- [ ] Ship `doctor --fix`: deterministic, reviewable spec repairs written in place — operationIds and tags derived exactly as generate would derive them, so the spec converges on the CLI it already produces. Never invents prose: descriptions/examples stay with the spec-acquisition skill ([Future](../../PRD.md#future-multi-api-toolsets-and-spec-acquisition)). Opt-in write, clean diff, grade printed before/after — the epic's demo.
