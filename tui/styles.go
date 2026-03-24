package tui

import "github.com/charmbracelet/lipgloss"

var (
	colorGreen    = lipgloss.Color("#00FF87")
	colorYellow   = lipgloss.Color("#FFD700")
	colorRed      = lipgloss.Color("#FF4136")
	colorCyan     = lipgloss.Color("#00CFFF")
	colorBgCard   = lipgloss.Color("#161B22")
	colorBorder   = lipgloss.Color("#30363D")
	colorText     = lipgloss.Color("#E6EDF3")
	colorTextDim  = lipgloss.Color("#8B949E")

	styleHeader = lipgloss.NewStyle().
			Foreground(colorCyan).
			Bold(true)

	styleCard = lipgloss.NewStyle().
			Background(colorBgCard).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorBorder).
			Padding(0, 2)

	styleLabel = lipgloss.NewStyle().
			Foreground(colorTextDim)

	styleValue = lipgloss.NewStyle().
			Foreground(colorText).
			Bold(true)

	styleGood = lipgloss.NewStyle().
			Foreground(colorGreen).
			Bold(true)

	styleWarn = lipgloss.NewStyle().
			Foreground(colorYellow).
			Bold(true)

	styleCrit = lipgloss.NewStyle().
			Foreground(colorRed).
			Bold(true)

	styleMuted = lipgloss.NewStyle().
			Foreground(colorTextDim)

	styleActiveTab = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#0D1117")).
			Background(colorCyan).
			Padding(0, 2).
			Bold(true)

	styleInactiveTab = lipgloss.NewStyle().
				Foreground(colorTextDim).
				Background(colorBgCard).
				Padding(0, 2)
)

// battStyle returns a status style based on battery percentage.
func battStyle(pct int) lipgloss.Style {
	switch {
	case pct > 40:
		return styleGood
	case pct > 20:
		return styleWarn
	default:
		return styleCrit
	}
}

// tempStyle returns a status style based on temperature.
func tempStyle(c float64) lipgloss.Style {
	switch {
	case c < 35:
		return styleGood
	case c < 45:
		return styleWarn
	default:
		return styleCrit
	}
}

// cycleStyle returns a status style based on cycle count fraction.
func cycleStyle(cycles, max int) lipgloss.Style {
	if max == 0 {
		return styleGood
	}
	pct := float64(cycles) / float64(max)
	switch {
	case pct < 0.6:
		return styleGood
	case pct < 0.85:
		return styleWarn
	default:
		return styleCrit
	}
}

// healthStyle returns a status style based on health percentage.
func healthStyle(h float64) lipgloss.Style {
	switch {
	case h >= 80:
		return styleGood
	case h >= 60:
		return styleWarn
	default:
		return styleCrit
	}
}
