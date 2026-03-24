package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/lipgloss"
	"github.com/ppunit/volt/collect"
)

func buildProcessTable(procs []collect.Process, width int) table.Model {
	if width < 60 {
		width = 60
	}
	cmdW := width - 42
	if cmdW < 10 {
		cmdW = 10
	}

	cols := []table.Column{
		{Title: "PID", Width: 7},
		{Title: "Process", Width: cmdW},
		{Title: "%CPU", Width: 7},
		{Title: "Mem%", Width: 7},
		{Title: "Energy", Width: 8},
		{Title: "User", Width: 10},
	}

	rows := make([]table.Row, 0, len(procs))
	for _, p := range procs {
		rows = append(rows, table.Row{
			fmt.Sprintf("%d", p.PID),
			truncate(p.Command, cmdW),
			fmt.Sprintf("%.1f", p.CPU),
			fmt.Sprintf("%.1f", p.Mem),
			fmt.Sprintf("%.0f", p.Power),
			p.User,
		})
	}

	t := table.New(
		table.WithColumns(cols),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(20),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(colorBorder).
		BorderBottom(true).
		Foreground(colorCyan).
		Bold(true)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("#0D1117")).
		Background(colorCyan).
		Bold(true)
	t.SetStyles(s)

	return t
}

func renderProcesses(m Model) string {
	var sb strings.Builder
	sb.WriteString(styleHeader.Render("  TOP PROCESSES") +
		styleMuted.Render("  (refreshes every 3s)") + "\n\n")

	if len(m.processes) == 0 {
		sb.WriteString("  " + styleMuted.Render("No process data available") + "\n")
		return sb.String()
	}

	tableView := lipgloss.NewStyle().MarginLeft(2).Render(m.procTable.View())
	sb.WriteString(tableView + "\n")
	sb.WriteString("\n  " + styleMuted.Render("↑/↓ or j/k to scroll"))
	return sb.String()
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-1] + "…"
}
