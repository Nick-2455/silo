package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Nick-2455/silo/internal/app"
	"github.com/Nick-2455/silo/internal/mcp"
	"github.com/Nick-2455/silo/internal/tui"
)

func main() {
	serverMode := flag.Bool("server", false, "Start as MCP stdio server")
	flag.Parse()

	// Bootstrap application dependencies
	deps, err := app.Bootstrap()
	if err != nil {
		fmt.Fprintf(os.Stderr, "bootstrap error: %v\n", err)
		os.Exit(1)
	}
	defer func() { _ = deps.Store.Close() }()

	if *serverMode {
		ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
		defer cancel()
		if err := runServer(ctx, deps); err != nil {
			fmt.Fprintf(os.Stderr, "server error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// TUI mode — requires a terminal
	if !isTerminal() {
		fmt.Fprintln(os.Stderr, "silo: TUI requires a terminal. Use --server for MCP stdio mode or --help for usage.")
		os.Exit(1)
	}

	model := tui.NewModel(deps)
	p := tea.NewProgram(model, tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "tui error: %v\n", err)
		os.Exit(1)
	}
}

func isTerminal() bool {
	stat, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (stat.Mode() & os.ModeCharDevice) != 0
}

func runServer(ctx context.Context, deps *app.Deps) error {
	handlerDeps := mcp.NewDeps(deps)

	// Run server in a goroutine so we can listen for shutdown
	errCh := make(chan error, 1)
	go func() {
		errCh <- mcp.StartServer(handlerDeps)
	}()

	select {
	case <-ctx.Done():
		// Graceful shutdown on SIGINT/SIGTERM
		fmt.Fprintln(os.Stderr, "shutting down MCP server...")
		return nil
	case err := <-errCh:
		return err
	}
}
