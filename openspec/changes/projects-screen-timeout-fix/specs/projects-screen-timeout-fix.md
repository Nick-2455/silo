# Delta Spec — Projects screen timeout fix

## Requirements

### R1

The application MUST list active projects from the SQLite cache on the TUI Projects screen when active projects exist.

### R2

The application MUST avoid N+1 project-subarea queries in the normal Projects screen load path.

### R3

The application MUST keep the Projects screen usable when the SQLite store is contended by another running `silo --server` process.

### R4

If project subarea enrichment cannot complete in time, the application MUST degrade gracefully by showing the project list without blocking the screen, and it MUST recover on a later refresh or reload.

### R5

The application MUST NOT surface `context deadline exceeded` during normal Projects screen use.

### R6

The repository MUST continue to pass `go test ./...`.

## Scenarios

### Scenario: populated cache renders projects

Given a SQLite cache containing active projects such as `blacksight`
When I open the TUI Projects screen
Then the screen lists those active projects
And the footer does not show a load error
And the empty-state message `No projects yet` is not shown.

### Scenario: concurrent server does not blank the screen

Given the TUI is open and `./silo --server` is running in another terminal
When I open or refresh the Projects screen
Then the app still shows the available projects if the base cache query succeeds
And if subarea data is unavailable, the screen degrades gracefully instead of timing out
And the user can retry and recover on a later refresh.

### Scenario: timeout text is not part of normal use

Given a healthy local cache with active projects
When I navigate to the Projects screen repeatedly
Then the UI does not display `context deadline exceeded` in the footer or status area.

### Scenario: test suite remains green

Given the fix is merged
When I run `go test ./...`
Then the full test suite passes.
