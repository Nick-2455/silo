# Proposal — Projects screen timeout fix

## Intent

Fix the TUI Projects screen so active projects from the local SQLite cache render reliably, even when `silo --server` is running in another terminal.

## Scope

- Rework project loading in the SQLite graph store to avoid nested read queries / long-lived read cursors.
- Preserve or restore project detail behavior for linked subareas.
- Add regression tests for project listing and timeout-safe behavior.

## Affected areas

- `internal/store/graph.go`
- `internal/store/graph_test.go`
- `internal/tui/model.go`
- `internal/tui/screens/projects.go` only if rendering needs fallback messaging
- `internal/tui/tui_test.go` or a focused model test file

## Risks

- Changing how subarea IDs are loaded could affect project detail rendering.
- A query rewrite could accidentally change ordering or active/inactive filtering.
- SQLite contention may still appear if the fix keeps multiple dependent queries in the hot path.

## Rollback

Revert the store/query rewrite and any TUI fallback changes. The change should be small enough to back out cleanly.

## Success criteria

- Projects from a populated cache render instead of showing the empty state.
- No `context deadline exceeded` appears during normal Projects screen use.
- Contention degrades gracefully rather than blanking the screen.
- `go test ./...` passes.
