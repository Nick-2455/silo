# Verify Report — Projects screen timeout fix

Date: 2026-05-15

## What changed

- Rewrote `ListActiveProjects` to be a single set-based query (no nested per-project subarea queries).
- Ensured SQLite `busy_timeout` is set via `PRAGMA busy_timeout = 15000` on open.

## Automated verification

```bash
go test ./...
```

Result: ✅ PASS

```bash
go build -o silo ./cmd/silo
```

Result: ✅ PASS

## Manual verification checklist

1. Ensure no stray `silo` processes hold `~/.local/share/silo/state.db`.
2. Run `./silo` and press `p` → Projects should list active projects (e.g., `blacksight`).
3. Optional concurrency check: run `./silo --server` in another terminal and re-open Projects; it should remain usable (busy_timeout should prevent spurious timeouts).
