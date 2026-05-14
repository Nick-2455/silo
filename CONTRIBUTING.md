# Contributing

Marrow is a community knowledge graph tool. Contributions that make it work for more people are welcome.

## Principles

- **Zero-config for basic use.** `go install` + `marrow` should work.
- **User-defined taxonomy.** No hardcoded domains. Everyone organizes knowledge differently.
- **Graph-first.** Everything is a node or an edge. New features should connect to the graph.
- **Engram is the source of truth.** Content lives in Engram. SQLite is the local cache.

## Developing

```bash
git clone https://github.com/Nick-2455/marrow
cd marrow
go build ./cmd/marrow
go test ./...
```

Requires Engram installed and in PATH for integration tests.

## Code structure

| Package | Role |
|---------|------|
| `cmd/marrow` | Entry point — TUI mode or `--server` MCP mode |
| `internal/domain` | Types, interfaces, errors — no dependencies |
| `internal/store` | SQLite — graph topology (`graph_nodes`, `graph_edges`) and resource triage |
| `internal/engram` | MCP stdio client to Engram — content CRUD |
| `internal/tui` | Bubble Tea TUI — screens, router, styles |
| `internal/mcp` | MCP server — tools exposed to external agents |
| `internal/obsidian` | Vault export — markdown + wikilinks |
| `internal/app` | Dependency bootstrap |
| `internal/config` | YAML config loading |

## Adding a screen

1. Add `ScreenXxx` to the enum in `internal/tui/router.go`
2. Add routes (forward/backward)
3. Create `internal/tui/screens/xxx.go` with `RenderXxx()` function
4. Add state fields to `Model` in `model.go`
5. Add keyboard shortcut in `handleKey()`
6. Wire `View()` to new screen

## Adding an MCP tool

1. Add handler in `internal/mcp/handlers_graph.go`
2. Register tool with JSON schema in `internal/mcp/server.go`

## Adding a node type

1. Add to `NodeType` constants in `internal/domain/models.go`
2. Add to `CHECK` constraint in `internal/store/migrations.go`
3. Add struct if needed
4. Implement GraphStore queries
