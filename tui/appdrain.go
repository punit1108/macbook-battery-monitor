package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/lipgloss"
	"github.com/ppunit/volt/store"
)

// ── Table ─────────────────────────────────────────────────────────────────────

func buildAppDrainTable(entries []store.AppDrainEntry, width int) table.Model {
	if width < 60 {
		width = 60
	}
	// Fixed column widths: 3+13+11+9+9+9 = 54, plus separators ~12 = 66
	cmdW := width - 68
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
		table.WithHeight(14),
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

// ── Main view ─────────────────────────────────────────────────────────────────

func renderAppDrain(m Model) string {
	var sb strings.Builder

	sb.WriteString(styleHeader.Render("  APP BATTERY DRAIN") + "\n\n")

	// Period selector.
	sb.WriteString(renderPeriodSelector(m.drainPeriod, m.appDrainLoading) + "\n\n")

	if m.appDrainLoading {
		sb.WriteString("  " + m.spinner.View() + " Loading...\n")
		return sb.String()
	}

	entries := m.appDrainEntries
	records := m.appDrainRecords

	if len(records) == 0 {
		sb.WriteString("  " + styleMuted.Render("No data for this period.") + "\n")
		sb.WriteString("  " + styleMuted.Render("Run  volt install  to start the background daemon.") + "\n")
		return sb.String()
	}

	if len(entries) == 0 {
		sb.WriteString("  " + styleMuted.Render("No process data in this period's records yet.") + "\n")
		sb.WriteString("  " + styleMuted.Render("Process snapshots are collected every 5 minutes by the daemon.") + "\n")
		return sb.String()
	}

	// Summary.
	recordsWithProcs := 0
	for _, r := range records {
		if len(r.TopProcs) > 0 {
			recordsWithProcs++
		}
	}
	summary := fmt.Sprintf("%d apps  ·  %d process snapshots  ·  %d total records",
		len(entries), recordsWithProcs, len(records))
	sb.WriteString("  " + styleMuted.Render(summary) + "\n\n")

	// Top-3 bars.
	sb.WriteString("  " + styleHeader.Render("TOP OFFENDERS") + "\n\n")
	barWidth := m.width - 34
	if barWidth < 10 {
		barWidth = 10
	}
	for i, e := range entries {
		if i >= 3 {
			break
		}
		bar := drainBar(e.TotalPower, entries[0].TotalPower, barWidth)
		rankStyles := []lipgloss.Style{styleCrit, styleWarn, styleGood}
		sb.WriteString(fmt.Sprintf("  %s  %-20s  %s  %.0f\n",
			rankStyles[i].Render(fmt.Sprintf("#%d", i+1)),
			truncate(e.Command, 20),
			bar,
			e.TotalPower,
		))
	}
	sb.WriteString("\n")

	// Table.
	sb.WriteString("  " + styleHeader.Render("ALL APPS") + "\n\n")
	tableView := lipgloss.NewStyle().MarginLeft(2).Render(m.appDrainTable.View())
	sb.WriteString(tableView + "\n")
	sb.WriteString("\n  " + styleMuted.Render("↑/↓: scroll  ·  Enter/Space: view app graph  ·  [/]: change period"))

	return sb.String()
}

// renderPeriodSelector renders the period toggle row.
func renderPeriodSelector(active drainPeriod, loading bool) string {
	var parts []string
	parts = append(parts, styleLabel.Render("  Period: "))
	for i := drainPeriod(0); i < periodCount; i++ {
		label := periodLabels[i]
		if i == active {
			parts = append(parts, styleActiveTab.Render(label))
		} else {
			parts = append(parts, styleInactiveTab.Render(label))
		}
		parts = append(parts, " ")
	}
	if loading {
		parts = append(parts, styleMuted.Render(" loading…"))
	} else {
		parts = append(parts, styleMuted.Render(" ← [ and ] to change →"))
	}
	return strings.Join(parts, "")
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

// ── Popup ─────────────────────────────────────────────────────────────────────

func renderAppPopup(entry store.AppDrainEntry, period drainPeriod, termWidth int) string {
	popupW := termWidth - 20
	if popupW < 50 {
		popupW = 50
	}
	if popupW > 100 {
		popupW = 100
	}
	innerW := popupW - 4 // subtract border + padding

	var sb strings.Builder

	// Stats row.
	sb.WriteString(styleLabel.Render("Total Impact: ") +
		styleCrit.Render(fmt.Sprintf("%.0f", entry.TotalPower)) + "   ")
	sb.WriteString(styleLabel.Render("Avg/Snapshot: ") +
		styleValue.Render(fmt.Sprintf("%.1f", entry.AvgPower)) + "   ")
	sb.WriteString(styleLabel.Render("Avg CPU: ") +
		styleValue.Render(fmt.Sprintf("%.1f%%", entry.AvgCPU)) + "   ")
	sb.WriteString(styleLabel.Render("Presence: ") +
		styleValue.Render(fmt.Sprintf("%.0f%%", entry.SharePct)))
	sb.WriteString("\n\n")

	chartW := innerW - 8 // leave room for Y-axis label
	if chartW < 10 {
		chartW = 10
	}
	if len(entry.BucketPower) > 0 {
		sb.WriteString(renderBucketChart(
			"ENERGY IMPACT OVER TIME",
			fmt.Sprintf("last %s", periodLabels[period]),
			entry.BucketPower, chartW, 8, period,
		))
	} else {
		sb.WriteString(styleMuted.Render("  No time-series data available.\n"))
	}

	sb.WriteString("\n" + styleMuted.Render("Esc · Enter · Space  to close"))

	// Wrap in a rounded border box titled with the app name.
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorCyan).
		Padding(1, 2).
		Width(popupW)

	title := lipgloss.NewStyle().
		Foreground(colorCyan).
		Bold(true).
		Render("  " + entry.Command + "  ")

	// Place title in the top border via a join trick.
	box := boxStyle.Render(sb.String())
	// Overwrite the first border line's center with the title.
	lines := strings.Split(box, "\n")
	if len(lines) > 0 {
		topLine := lines[0]
		titleRunes := []rune(title)
		topRunes := []rune(topLine)
		start := (len(topRunes) - len(titleRunes)) / 2
		if start > 1 && start+len(titleRunes) < len(topRunes) {
			copy(topRunes[start:], titleRunes)
			lines[0] = string(topRunes)
		}
	}
	return strings.Join(lines, "\n")
}

// renderBucketChart draws an ASCII bar chart from pre-bucketed power values.
// The title and subtitle are rendered indented to align with the bar area.
func renderBucketChart(title, subtitle string, buckets []float64, width, height int, period drainPeriod) string {
	resampled := resampleBuckets(buckets, width)

	maxVal := 0.0
	for _, v := range resampled {
		if v > maxVal {
			maxVal = v
		}
	}

	blocks := []string{" ", "▁", "▂", "▃", "▄", "▅", "▆", "▇", "█"}

	// Fixed Y-axis label width so the title indent is predictable.
	maxLabel := fmt.Sprintf("%.0f", maxVal)
	yLabelW := len(maxLabel)
	if yLabelW < 5 {
		yLabelW = 5
	}
	// Bars begin immediately after the Y-axis: yLabelW chars + "│" = yLabelW+1 cols.
	indent := strings.Repeat(" ", yLabelW+1)

	var sb strings.Builder

	// Title + subtitle, then a separator that spans the full bar width.
	sb.WriteString(indent + styleHeader.Render(title))
	if subtitle != "" {
		sb.WriteString("  " + styleMuted.Render(subtitle))
	}
	sb.WriteString("\n")
	sb.WriteString(indent + styleMuted.Render(strings.Repeat("─", width)) + "\n\n")

	// Chart rows.
	for row := 0; row < height; row++ {
		switch row {
		case 0:
			sb.WriteString(styleMuted.Render(fmt.Sprintf("%*s│", yLabelW, maxLabel)))
		case height / 2:
			mid := fmt.Sprintf("%.0f", maxVal/2)
			sb.WriteString(styleMuted.Render(fmt.Sprintf("%*s│", yLabelW, mid)))
		default:
			sb.WriteString(styleMuted.Render(fmt.Sprintf("%*s│", yLabelW, "")))
		}

		for _, v := range resampled {
			ratio := 0.0
			if maxVal > 0 {
				ratio = v / maxVal
			}
			barHeight := ratio * float64(height)
			rowsFromBottom := float64(height - row)

			var ch string
			if barHeight >= rowsFromBottom {
				ch = blocks[len(blocks)-1]
			} else if barHeight > rowsFromBottom-1 {
				frac := barHeight - (rowsFromBottom - 1)
				idx := int(frac * float64(len(blocks)-1))
				if idx < 0 {
					idx = 0
				}
				if idx >= len(blocks) {
					idx = len(blocks) - 1
				}
				ch = blocks[idx]
			} else {
				ch = " "
			}

			switch {
			case ratio > 0.6:
				sb.WriteString(styleCrit.Render(ch))
			case ratio > 0.3:
				sb.WriteString(styleWarn.Render(ch))
			default:
				sb.WriteString(styleGood.Render(ch))
			}
		}
		sb.WriteString("\n")
	}

	// X-axis baseline.
	sb.WriteString(styleMuted.Render(strings.Repeat(" ", yLabelW+1)+"└"+strings.Repeat("─", width)) + "\n")

	// Time labels aligned under the bar area.
	since := time.Now().Add(-period.duration())
	startLabel := since.Local().Format("Jan 2 15:04")
	endLabel := time.Now().Local().Format("Jan 2 15:04")
	gap := width - len(startLabel) - len(endLabel)
	if gap < 1 {
		gap = 1
	}
	sb.WriteString(indent +
		styleMuted.Render(startLabel) +
		strings.Repeat(" ", gap) +
		styleMuted.Render(endLabel) + "\n")

	return sb.String()
}

// resampleBuckets resamples src to exactly targetLen buckets using averaging.
func resampleBuckets(src []float64, targetLen int) []float64 {
	if len(src) == 0 || targetLen == 0 {
		return make([]float64, targetLen)
	}
	if len(src) == targetLen {
		return src
	}
	out := make([]float64, targetLen)
	for i := range out {
		// Map output index to a range in src.
		lo := float64(i) * float64(len(src)) / float64(targetLen)
		hi := float64(i+1) * float64(len(src)) / float64(targetLen)
		sum, count := 0.0, 0.0
		for j := int(lo); float64(j) < hi && j < len(src); j++ {
			sum += src[j]
			count++
		}
		if count > 0 {
			out[i] = sum / count
		}
	}
	return out
}
