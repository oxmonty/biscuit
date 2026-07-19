# Test specs

| File | Source URL | Retrieved | License | Ladder rung |
|---|---|---|---|---|
| `petstore.yaml` | https://raw.githubusercontent.com/OAI/OpenAPI-Specification/3.0.4/examples/v3.0/petstore.yaml | 2026-07-17 | Apache License 2.0 | easy — small, canonical, no edge cases |
| `train-travel.yaml` | https://raw.githubusercontent.com/bump-sh-examples/train-travel-api/main/openapi.yaml | 2026-07-17 | CC-BY-NC-SA-4.0 | medium — OpenAPI 3.1, oneOf, multiple auth schemes, links |
| `openai.yaml` | https://raw.githubusercontent.com/openai/openai-openapi/main/openapi.yaml | 2026-07-17 | MIT License | hard — very large, real-world spec |
| `museum.yaml` | https://raw.githubusercontent.com/Redocly/museum-openapi-example/main/openapi.yaml | 2026-07-17 | MIT License | medium — 3.1, contrasting modeling style, binary image responses |
| `galaxy.yaml` | https://cdn.jsdelivr.net/npm/@scalar/galaxy/dist/3.1.yaml | 2026-07-17 | MIT License (scalar monorepo) | medium — 3.1.1, deliberate edge-case gauntlet: multi-auth, file upload, webhooks, a real circular ref |
| `pokeapi.yml` | https://raw.githubusercontent.com/PokeAPI/pokeapi/master/openapi.yml | 2026-07-17 | BSD-3-Clause | mapping scale — 98 GET-only nested resource operations; doctor grades it 10/100 (the override-rescue demo) |
| `stripe.yaml` | https://raw.githubusercontent.com/stripe/openapi/master/openapi/spec3.yaml | 2026-07-19 | MIT License | tree-derivation stress test — large real-world 3.x spec, deeply nested resources, polymorphic oneOf on nearly every object; a distinct shape from openai.yaml |
| `pathological/cyclic-refs.yaml` | hand-written | 2026-07-17 | n/a | pathological — cyclic and self-referencing $refs |
| `pathological/unresolvable-ref.yaml` | hand-written | 2026-07-17 | n/a | pathological — missing local and external $refs |
| `pathological/duplicate-operation-ids.yaml` | hand-written | 2026-07-17 | n/a | pathological — two operations share one operationId |
