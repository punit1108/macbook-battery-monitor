package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ppunit/volt/agent"
	"github.com/ppunit/volt/daemon"
	"github.com/ppunit/volt/tui"
)

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "daemon":
			daemon.Run()
		case "install":
			if err := agent.Install(); err != nil {
				fmt.Fprintln(os.Stderr, "install failed:", err)
				os.Exit(1)
			}
		case "uninstall":
			if err := agent.Uninstall(); err != nil {
				fmt.Fprintln(os.Stderr, "uninstall failed:", err)
				os.Exit(1)
			}
		default:
			fmt.Fprintf(os.Stderr, "unknown command: %s\n\n", os.Args[1])
			printUsage()
			os.Exit(1)
		}
		return
	}

	p := tea.NewProgram(
		tui.InitialModel(),
		tea.WithAltScreen(),
	)
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`volt — macOS battery analytics TUI

Usage:
  volt            Launch the interactive TUI
  volt daemon     Run the background data collector
  volt install    Install daemon as a LaunchAgent (auto-start on login)
  volt uninstall  Remove the LaunchAgent and binary`)
}
