# E7: MCP serve

every generated CLI is an MCP server. → [MCP subcommand](../../PRD.md#mcp-subcommand)

## Stories

- [ ] Map operations to MCP tools and serve over stdio, then Streamable HTTP, on the official `modelcontextprotocol/go-sdk`, pinning the targeted MCP protocol revision.
- [ ] Named toolsets in `biscuit.yaml` + `mcp serve --toolset` — serve-time allowlists over operations/tags (admin vs public), seeded from tags/`x-internal`; the HTTP sidecar defaults to the public toolset. → [MCP subcommand](../../PRD.md#mcp-subcommand)
- [ ] Template a project-scope `.mcp.json` into generated repos so opening one in Claude Code wires the CLI's tools automatically — zero-command team onboarding. → [MCP subcommand](../../PRD.md#mcp-subcommand)
- [ ] Generated README documents the HTTP-transport sidecar deployment for API owners (run `{binary} mcp serve --transport http` next to the API, reverse-proxy `/mcp` to it, env-var auth); public-facing endpoint auth is deferred to the MCP-gateway future item. → [MCP subcommand](../../PRD.md#mcp-subcommand)
