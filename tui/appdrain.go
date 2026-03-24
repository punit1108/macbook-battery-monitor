package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/lipgloss"
	"github.com/ppunit/mac_battery/store"
)

func buildAppDrainTable(entries []store.AppDrainEntry, width int) table.Model {
	if width < 60 {
		width = 60
	}
	cmdW := width - 56
	if cmdW < 12 {
		cmdW = 12
	}

	cols := []table.Column{
		{Title: "#", Width: 3},
		{Title: "App", Width: cmdW},
		{Title: "Total Impact", Width: 13},
		{Title: "Avg Impact", Width: 11},
		{Title: "Avg CPU%", Width: 9},
		{Title: "Seen In", Width: 9},
		{Title: "Records%", Width: 9},
	}

	rows := make([]table.Row, 0, len(entries))
	for i, e := range entries {
		rows = append(rows, table.Row{
			fmt.Sprintf("%d", i+1),
			truncate(e.Command, cmdW),
			fmt.Sprintf("%.0f", e.TotalPower),
			fmt.Sprintf("%.1f", e.AvgPower),
			fmt.Sprintf("%.1f%%", e.AvgCPU),
			fmt.Sprintf("%d", e.Appearances),
			fmt.Sprintf("%.0f%%", e.SharePct),
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

func renderAppDrain(m Model) string {
	var sb strings.Builder
	sb.WriteString(styleHeader.Render("  APP BATTERY DRAIN") +
		styleMuted.Render("  (aggregated from daemon logs)") + "\n\n")

	if len(m.history) == 0 {
		sb.WriteString("  " + styleMuted.Render("No history data yet.") + "\n")
		sb.WriteString("  " + styleMuted.Render("Install the daemon to start collecting: mac-battery install") + "\n")
		return sb.String()
	}

	entries := store.AggregateAppDrain(m.history)
	if len(entries) == 0 {
		sb.WriteString("  " + styleMuted.Render("No process data in history records yet.") + "\n")
		sb.WriteString("  " + styleMuted.Render("Process data is sampled every 5 minutes by the daemon.") + "\n")
		return sb.String()
	}

	// Summary line above the table.
	recordsWithProcs := 0
	for _, r := range m.history {
		if len(r.TopProcs) > 0 {
			recordsWithProcs++
		}
	}
	summary := fmt.Sprintf("%d apps tracked across %d snapshots (%d total records)",
		len(entries), recordsWithProcs, len(m.history))
	sb.WriteString("  " + styleMuted.Render(summary) + "\n\n")

	// Top-3 highlight bar.
	if len(entries) >= 1 {
		sb.WriteString("  " + styleHeader.Render("TOP OFFENDERS") + "\n\n")
		for i, e := range entries {
			if i >= 3 {
				break
			}
			bar := drainBar(e.TotalPower, entries[0].TotalPower, m.width-30)
			rank := []lipgloss.Style{styleCrit, styleWarn, styleGood}[i]
			sb.WriteString(fmt.Sprintf("  %s  %-20s  %s  %.0f impact\n",
				rank.Render(fmt.Sprintf("#%d", i+1)),
				truncate(e.Command, 20),
				bar,
				e.TotalPower,
			))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("  " + styleHeader.Render("ALL APPS") + "\n\n")
	tableView := lipgloss.NewStyle().MarginLeft(2).Render(m.appDrainTable.View())
	sb.WriteString(tableView + "\n")
	sb.WriteString("\n  " + styleMuted.Render("↑/↓ or j/k to scroll  ·  Total Impact = sum of energy scores across all snapshots"))

	return sb.String()
}

// drainBar renders a proportional ASCII bar relative to the max value.
func drainBar(value, max float64, width int) string {
	if width < 5 {
		width = 5
	}
	if max == 0 {
		max = 1
	}
	filled := int(value / max * float64(width))
	if filled > width {
		filled = width
	}
	bar := strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
	// Color gradient: high=red, mid=yellow, low=green
	ratio := value / max
	switch {
	case ratio > 0.6:
		return styleCrit.Render(bar)
	case ratio > 0.3:
		return styleWarn.Render(bar)
	default:
		return styleGood.Render(bar)
	}
}
