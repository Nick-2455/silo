# SDD Init — Projects screen timeout fix

## Context

The TUI Projects screen is failing with `No projects yet` and a footer error similar to `load projects: store: query project subareas: context deadline exceeded` even when the SQLite cache contains active projects.

## Project snapshot

- Language/build: Go, `go build ./cmd/silo`
- Test runner: `go test ./...`
- SDD mode: auto
- Artifact store: openspec
- Review budget: 400 changed lines
- Strict TDD: false in `openspec/config.yaml`

## Repository signals

- `.atl/skill-registry.md` exists.
- Existing SDD config is present and should not be rewritten destructively.
- The bug path is in SQLite graph-store project loading and the TUI Projects screen.

## Likely root cause

`ListActiveProjects` appears to load each project's subareas with nested queries while the outer rows cursor is still open, which can trigger contention/timeouts under SQLite/WAL plus the app's single-connection setup.

## Planning direction

Plan a fix that removes the N+1 query pattern, keeps reads short, and preserves graceful degradation when the cache is contended.
