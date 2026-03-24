package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/lipgloss"
	"github.com/ppunit/mac_battery/collect"
)

func renderDashboard(m Model) string {
	b := m.battery
	w := m.width
	if w < 40 {
		w = 40
	}

	var sb strings.Builder

	// Battery gauge.
	sb.WriteString(styleHeader.Render("  BATTERY STATUS") + "\n\n")

	pct := float64(b.CurrentCapacity) / 100.0
	gaugeWidth := w - 20
	if gaugeWidth < 20 {
		gaugeWidth = 20
	}
	gauge := newBatteryGauge(gaugeWidth, b.CurrentCapacity)
	sb.WriteString("  " + gauge.ViewAs(pct))

	pctStr := battStyle(b.CurrentCapacity).Render(fmt.Sprintf(" %d%%", b.CurrentCapacity))
	timeStr := formatTime(b.TimeRemaining, b.IsCharging)
	sb.WriteString(pctStr + "  " + styleMuted.Render(timeStr) + "\n\n")

	// Status line.
	if b.IsCharging {
		adapter := b.AdapterName
		if adapter == "" {
			adapter = fmt.Sprintf("%dW adapter", b.AdapterWatts)
		}
		sb.WriteString("  " + styleLabel.Render("Status:  ") +
			styleGood.Render("⚡ CHARGING") +
			styleMuted.Render(fmt.Sprintf("  via %s", adapter)) + "\n")
	} else if b.ExternalConnected {
		sb.WriteString("  " + styleLabel.Render("Status:  ") +
			styleWarn.Render("⏸ NOT CHARGING") + "\n")
	} else {
		sb.WriteString("  " + styleLabel.Render("Status:  ") +
			styleValue.Render("🔋 ON BATTERY") + "\n")
	}

	sb.WriteString("\n")

	// Two-column metrics.
	leftLines := []string{
		row("Voltage", fmt.Sprintf("%.2f V", float64(b.Voltage)/1000.0)),
		row("Current", formatAmperage(b)),
		row("Temp", tempStyle(b.TemperatureC).Render(fmt.Sprintf("%.1f °C", b.TemperatureC))),
		row("Condition", conditionStr(b.Condition)),
	}
	rightLines := []string{
		row("Power In", formatWatts(b.SystemPowerInMW)),
		row("Sys Load", formatWatts(b.SystemLoadMW)),
		row("Eff Loss", formatWatts(b.AdapterEfficiencyLossMW)),
		row("Adapter", adapterWattsStr(b)),
	}

	colW := (w - 6) / 2
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

func newBatteryGauge(width, pct int) progress.Model {
	var gradFull, gradEmpty lipgloss.Color
	switch {
	case pct > 40:
		gradFull = lipgloss.Color("#00FF87")
		gradEmpty = lipgloss.Color("#00A550")
	case pct > 20:
		gradFull = lipgloss.Color("#FFD700")
		gradEmpty = lipgloss.Color("#B8860B")
	default:
		gradFull = lipgloss.Color("#FF4136")
		gradEmpty = lipgloss.Color("#8B0000")
	}
	p := progress.New(
		progress.WithGradient(string(gradEmpty), string(gradFull)),
		progress.WithWidth(width),
		progress.WithoutPercentage(),
	)
	return p
}

func formatTime(minutes int, charging bool) string {
	if minutes == 65535 || minutes == 0 {
		if charging {
			return "calculating..."
		}
		return ""
	}
	h := minutes / 60
	m := minutes % 60
	if charging {
		return fmt.Sprintf("%dh %dm to full", h, m)
	}
	return fmt.Sprintf("%dh %dm remaining", h, m)
}

func formatAmperage(b collect.BatteryData) string {
	a := float64(b.Amperage) / 1000.0
	if a < 0 {
		return fmt.Sprintf("%.2f A ↓", -a)
	}
	return fmt.Sprintf("%.2f A ↑", a)
}

func formatWatts(mw int64) string {
	if mw == 0 {
		return styleMuted.Render("—")
	}
	w := float64(mw) / 1000.0
	if w < 0 {
		w = -w
	}
	return styleValue.Render(fmt.Sprintf("%.1f W", w))
}

func adapterWattsStr(b collect.BatteryData) string {
	if b.AdapterWatts == 0 {
		return styleMuted.Render("—")
	}
	return styleValue.Render(fmt.Sprintf("%d W", b.AdapterWatts))
}

func conditionStr(c string) string {
	if c == "Normal" || c == "" {
		return styleGood.Render("Normal")
	}
	return styleCrit.Render(c)
}

func row(label, value string) string {
	return styleLabel.Render(fmt.Sprintf("%-10s", label+":")) + "  " + value
}
