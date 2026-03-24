package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/lipgloss"
)

func renderHealth(m Model) string {
	b := m.battery
	w := m.width
	if w < 40 {
		w = 40
	}

	var sb strings.Builder
	sb.WriteString(styleHeader.Render("  BATTERY HEALTH") + "\n\n")

	gaugeW := w - 20
	if gaugeW < 20 {
		gaugeW = 20
	}

	// Condition.
	sb.WriteString("  " + row("Condition", conditionStr(b.Condition)) + "\n\n")

	// Cycle count gauge.
	cyclePct := 0.0
	if b.DesignCycleCount > 0 {
		cyclePct = float64(b.CycleCount) / float64(b.DesignCycleCount)
		if cyclePct > 1 {
			cyclePct = 1
		}
	}
	cycleGauge := newHealthGauge(gaugeW, cycleStyle(b.CycleCount, b.DesignCycleCount))
	cycleLabel := cycleStyle(b.CycleCount, b.DesignCycleCount).Render(
		fmt.Sprintf("%d / %d cycles (%.0f%%)", b.CycleCount, b.DesignCycleCount, cyclePct*100),
	)
	sb.WriteString("  " + styleLabel.Render("Cycle Count:") + "\n")
	sb.WriteString("  " + cycleGauge.ViewAs(cyclePct) + "\n")
	sb.WriteString("  " + cycleLabel + "\n\n")

	// Health / capacity gauge.
	healthPct := b.HealthPercent / 100.0
	if healthPct > 1 {
		healthPct = 1
	}
	healthGauge := newHealthGauge(gaugeW, healthStyle(b.HealthPercent))
	healthLabel := healthStyle(b.HealthPercent).Render(
		fmt.Sprintf("%.1f%% of original", b.HealthPercent),
	)
	sb.WriteString("  " + styleLabel.Render("Capacity Health:") + "\n")
	sb.WriteString("  " + healthGauge.ViewAs(healthPct) + "\n")
	sb.WriteString("  " + healthLabel + "\n\n")

	// Capacity numbers.
	sb.WriteString("  " + styleHeader.Render("CAPACITY DETAILS") + "\n\n")
	maxCap := styleValue.Render(fmt.Sprintf("%d mAh", b.RawMaxCapacity))
	designCap := styleValue.Render(fmt.Sprintf("%d mAh", b.DesignCapacity))
	colW := (w - 6) / 2
	leftLines := []string{
		row("Max Capacity", maxCap),
		row("Design Cap", designCap),
	}
	rightLines := []string{
		row("Cycles", styleValue.Render(fmt.Sprintf("%d", b.CycleCount))),
		row("Design Cycles", styleValue.Render(fmt.Sprintf("%d", b.DesignCycleCount))),
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

func newHealthGauge(width int, style lipgloss.Style) progress.Model {
	// Extract foreground color from the style to use as gradient.
	fg := style.GetForeground()
	full := lipgloss.Color(string(fg.(lipgloss.Color)))
	p := progress.New(
		progress.WithSolidFill(string(full)),
		progress.WithWidth(width),
		progress.WithoutPercentage(),
	)
	return p
}
