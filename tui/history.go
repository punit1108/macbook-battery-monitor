package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/ppunit/volt/store"
)

func renderHistory(m Model) string {
	var sb strings.Builder
	sb.WriteString(styleHeader.Render("  BATTERY HISTORY") +
		styleMuted.Render("  (from daemon log — last 24h)") + "\n\n")

	records := m.history
	if len(records) == 0 {
		sb.WriteString("  " + styleMuted.Render("No history data yet.") + "\n")
		sb.WriteString("  " + styleMuted.Render("Start collecting: volt install") + "\n")
		sb.WriteString("  " + styleMuted.Render("Data updates every 60 seconds.") + "\n")
		return sb.String()
	}

	w := m.width - 10
	if w < 30 {
		w = 30
	}
	if w > 120 {
		w = 120
	}
	chartH := 10

	sb.WriteString(renderASCIIChart(records, w, chartH))
	sb.WriteString("\n")
	sb.WriteString(renderHistorySummary(records))
	return sb.String()
}

// renderASCIIChart draws a simple line chart of battery % over time.
func renderASCIIChart(records []store.Record, width, height int) string {
	// Take at most `width` most-recent points.
	pts := records
	if len(pts) > width {
		pts = pts[len(pts)-width:]
	}
	n := len(pts)

	// Build a grid [height][width] of runes.
	grid := make([][]rune, height)
	for i := range grid {
		grid[i] = make([]rune, width)
		for j := range grid[i] {
			grid[i][j] = ' '
		}
	}

	// Plot each point. pct 0=bottom row, 100=top row.
	for i, pt := range pts {
		col := i * width / n
		if col >= width {
			col = width - 1
		}
		rowIdx := height - 1 - int(float64(pt.Pct)/100.0*float64(height-1))
		if rowIdx < 0 {
			rowIdx = 0
		}
		if rowIdx >= height {
			rowIdx = height - 1
		}
		if pt.Charging {
			grid[rowIdx][col] = '▲'
		} else {
			grid[rowIdx][col] = '●'
		}
	}

	// Connect adjacent plotted points with dashes.
	for i := 1; i < n; i++ {
		c1 := (i - 1) * width / n
		c2 := i * width / n
		if c2 >= width {
			c2 = width - 1
		}
		r1 := height - 1 - int(float64(pts[i-1].Pct)/100.0*float64(height-1))
		r2 := height - 1 - int(float64(pts[i].Pct)/100.0*float64(height-1))
		if r1 < 0 {
			r1 = 0
		}
		if r2 < 0 {
			r2 = 0
		}
		if r1 >= height {
			r1 = height - 1
		}
		if r2 >= height {
			r2 = height - 1
		}
		// Fill intermediate columns at interpolated rows.
		if c2 > c1+1 {
			for c := c1 + 1; c < c2; c++ {
				t := float64(c-c1) / float64(c2-c1)
				r := r1 + int(t*float64(r2-r1))
				if r >= 0 && r < height && grid[r][c] == ' ' {
					grid[r][c] = '·'
				}
			}
		}
	}

	// Render with Y-axis labels.
	var sb strings.Builder
	for row := 0; row < height; row++ {
		pct := 100 - int(float64(row)/float64(height-1)*100)
		label := fmt.Sprintf("%3d%%│", pct)
		sb.WriteString("  " + styleMuted.Render(label))
		for col := 0; col < width; col++ {
			ch := string(grid[row][col])
			switch grid[row][col] {
			case '▲':
				sb.WriteString(styleGood.Render(ch))
			case '●':
				sb.WriteString(styleWarn.Render(ch))
			case '·':
				sb.WriteString(styleMuted.Render(ch))
			default:
				sb.WriteString(ch)
			}
		}
		sb.WriteString("\n")
	}

	// X-axis.
	xAxis := "     └" + strings.Repeat("─", width)
	sb.WriteString("  " + styleMuted.Render(xAxis) + "\n")

	// X-axis time labels.
	if len(pts) >= 2 {
		oldest := pts[0].Ts.Local().Format("15:04")
		newest := pts[len(pts)-1].Ts.Local().Format("15:04")
		space := width - len(oldest) - len(newest)
		if space < 1 {
			space = 1
		}
		sb.WriteString("       " + styleMuted.Render(oldest) +
			strings.Repeat(" ", space) +
			styleMuted.Render(newest) + "\n")
	}

	// Legend.
	sb.WriteString("\n  " +
		styleGood.Render("▲ charging") + "  " +
		styleWarn.Render("● discharging") + "\n")

	return sb.String()
}

func renderHistorySummary(records []store.Record) string {
	if len(records) == 0 {
		return ""
	}

	first := records[0]
	last := records[len(records)-1]

	// Find min/max pct.
	minPct, maxPct := first.Pct, first.Pct
	totalChargeEvents := 0
	for i, r := range records {
		if r.Pct < minPct {
			minPct = r.Pct
		}
		if r.Pct > maxPct {
			maxPct = r.Pct
		}
		if i > 0 && r.Charging && !records[i-1].Charging {
			totalChargeEvents++
		}
	}

	duration := last.Ts.Sub(first.Ts).Round(time.Minute)

	var sb strings.Builder
	sb.WriteString("  " + styleHeader.Render("SESSION SUMMARY") + "\n\n")

	colW := 30
	leftLines := []string{
		row("Duration", styleValue.Render(formatDuration(duration))),
		row("Data Points", styleValue.Render(fmt.Sprintf("%d", len(records)))),
		row("Min Charge", battStyle(minPct).Render(fmt.Sprintf("%d%%", minPct))),
	}
	rightLines := []string{
		row("From", styleMuted.Render(first.Ts.Local().Format("Jan 2 15:04"))),
		row("To", styleMuted.Render(last.Ts.Local().Format("Jan 2 15:04"))),
		row("Max Charge", battStyle(maxPct).Render(fmt.Sprintf("%d%%", maxPct))),
	}
	for i := 0; i < len(leftLines); i++ {
		l := lipgloss.NewStyle().Width(colW).Render(leftLines[i])
		r := ""
		if i < len(rightLines) {
			r = rightLines[i]
		}
		sb.WriteString("  " + l + "  " + r + "\n")
	}
	return sb.String()
}

func formatDuration(d time.Duration) string {
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	if h > 0 {
		return fmt.Sprintf("%dh %dm", h, m)
	}
	return fmt.Sprintf("%dm", m)
}
