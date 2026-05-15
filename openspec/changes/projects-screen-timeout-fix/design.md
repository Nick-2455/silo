# Design — Projects screen timeout fix

## Approach

Replace the current per-project subarea lookup path with a single set-based SQLite query that returns all active projects and their linked subarea IDs in one cursor, then group rows in Go.

## Why this approach

- Removes the nested query pattern that can hold the outer cursor open and trigger contention.
- Keeps the read path short and predictable.
- Preserves existing `domain.Project` shape so the Projects detail screen can still resolve linked subareas.

## Implementation outline

1. Rewrite `ListActiveProjects` to use one query over `graph_nodes` joined to `graph_edges` on `applies_to`.
2. Group rows by project ID in Go so each project appears once with a slice of `SubareaIDs`.
3. Keep ordering deterministic by project title and subarea ID.
4. Add tests for:
   - active project list with linked subareas,
   - empty store,
   - contention-safe behavior or partial fallback if the query path is split.
5. If a fallback is needed, prefer retaining the last successful list or showing the list without subarea enrichment rather than failing the screen.

## Review notes

This should stay small, reviewable, and focused. Avoid widening the change into unrelated sessions or learnings loading unless a shared bug is proven.
