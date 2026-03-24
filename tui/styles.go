package tui

import "github.com/charmbracelet/lipgloss"

// Dark pastel palette — desaturated, easy on the eyes, dark-terminal friendly.
var (
	colorAccent   = lipgloss.Color("#5B8DB8") // slate blue  — headers, active tabs
	colorGood     = lipgloss.Color("#5F9E78") // sage green  — healthy / charging
	colorWarn     = lipgloss.Color("#B89650") // dark amber  — warnings
	colorCrit     = lipgloss.Color("#B86060") // muted brick — critical
	colorBgCard   = lipgloss.Color("#141C26") // dark navy   — card backgrounds
	colorBorder   = lipgloss.Color("#253040") // steel       — borders
	colorText     = lipgloss.Color("#B8C8D8") // soft silver — primary text
	colorTextDim  = lipgloss.Color("#4A5A6A") // dark steel  — secondary text
	colorTabBg    = lipgloss.Color("#1A2840") // deep navy   — active tab bg

	// Keep colorCyan as an alias used in a few places outside styles.go.
	colorCyan = colorAccent
)

var (
	styleHeader = lipgloss.NewStyle().
			Foreground(colorAccent).
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
			Foreground(colorGood).
			Bold(true)

	styleWarn = lipgloss.NewStyle().
			Foreground(colorWarn).
			Bold(true)

	styleCrit = lipgloss.NewStyle().
			Foreground(colorCrit).
			Bold(true)

	styleMuted = lipgloss.NewStyle().
			Foreground(colorTextDim)

	styleActiveTab = lipgloss.NewStyle().
			Foreground(colorText).
			Background(colorTabBg).
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
