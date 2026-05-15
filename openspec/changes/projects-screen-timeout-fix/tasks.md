# Tasks — Projects screen timeout fix

1. Rewrite `internal/store/graph.go::ListActiveProjects` to remove the nested per-project subarea query and return grouped project/subarea data from a single read path.
2. Add or update store tests in `internal/store/graph_test.go` to verify active projects still include linked subarea IDs and inactive projects remain excluded.
3. Add a TUI regression test that opening the Projects screen with cached data renders project names and does not surface the empty-state/error path.
4. If needed, add a graceful-fallback test for the contention path so the screen keeps prior data or degrades without a hard timeout.
5. Run `go test ./...` and confirm the suite passes.

## Workload forecast

Estimated change size: ~150-250 LOC. This should fit in a single review slice and stay below the 400-line split threshold.

## Delivery note

Keep the fix limited to the project-loading path unless another hotspot is proven by tests.
