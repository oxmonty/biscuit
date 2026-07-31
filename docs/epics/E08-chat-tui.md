# E8: Chat TUI

one Bubble Tea interface backs `mcp chat`, `{binary} chat`, and interactive SSE. → [Protocol scope](../../PRD.md#protocol-scope), [MCP subcommand](../../PRD.md#mcp-subcommand), [Spec discovery](../../PRD.md#spec-discovery)

## Stories

- [ ] Spike the MCP-client integration that the chat strategy leans on: drive a generated `{binary} mcp serve` from Claude Code, Warp, and [pi](https://github.com/earendil-works/pi) end to end (tool discovery, streaming, env auth, stdio and Streamable HTTP) — rich chat UX belongs to these clients, not an owned TUI; if they can't carry it, the pi-port question reopens with evidence. → [MCP subcommand](../../PRD.md#mcp-subcommand)
- [ ] Build the minimal built-in TUI — serviceable, not spectacular — with streaming and tool-call display, stealing the UX decisions of [pi](https://github.com/earendil-works/pi) on Bubble Tea.
- [ ] Add Anthropic and OpenAI providers behind a two-provider interface.
- [ ] Detect chat-shaped endpoints and emit the `{binary} chat` REPL.
- [ ] Route interactive-TTY SSE responses into the TUI.
- [ ] Upgrade spec discovery to the full UX: git-index enumeration with the gitignore-blind `WalkDir` fallback, the delayed stderr spinner, and a Bubble Tea countdown selector auto-picking the best-ranked candidate (non-TTY prints its pick).
