package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ppunit/mac_battery/collect"
	"github.com/ppunit/mac_battery/store"
)

// ── view ──────────────────────────────────────────────────────────────────────

type view int

const (
	viewDashboard view = iota
	viewHealth
	viewProcesses
	viewHistory
	viewAppDrain
	viewCount // must be last
)

// ── drainPeriod ───────────────────────────────────────────────────────────────

type drainPeriod int

const (
	period1d drainPeriod = iota
	period2d
	period3d
	period1w
	period1m
	period3m
	periodCount
)

var periodLabels = [periodCount]string{"1d", "2d", "3d", "1w", "1m", "3m"}

func (p drainPeriod) duration() time.Duration {
	switch p {
	case period2d:
		return 48 * time.Hour
	case period3d:
		return 72 * time.Hour
	case period1w:
		return 7 * 24 * time.Hour
	case period1m:
		return 30 * 24 * time.Hour
	case period3m:
		return 90 * 24 * time.Hour
	default: // period1d
		return 24 * time.Hour
	}
}

// ── Model ─────────────────────────────────────────────────────────────────────

// Model is the root bubbletea model.
type Model struct {
	width  int
	height int

	activeView view

	// Live data
	battery   collect.BatteryData
	processes []collect.Process

	// History view data (battery chart)
	history []store.Record

	// App Drain view data
	drainPeriod      drainPeriod
	appDrainEntries  []store.AppDrainEntry
	appDrainRecords  []store.Record
	appDrainLoading  bool
	showAppPopup     bool

	// Tables
	procTable     table.Model
	appDrainTable table.Model

	spinner spinner.Model

	loading      bool
	lastErr      error
	tickCount    int       // increments every 3s
	lastStoredAt time.Time // last time TUI wrote a record to the store
}

// InitialModel returns the starting model.
func InitialModel() Model {
	sp := spinner.New(spinner.WithSpinner(spinner.Dot))
	sp.Style = lipgloss.NewStyle().Foreground(colorCyan)

	return Model{
		width:   120,
		height:  30,
		loading: true,
		spinner: sp,
	}
}

// ── Init ──────────────────────────────────────────────────────────────────────

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		cmdTick(),
		cmdFetchBattery,
		cmdFetchProcesses,
		cmdFetchHistory,
	)
}

// ── Update ────────────────────────────────────────────────────────────────────

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if len(m.processes) > 0 {
			m.procTable = buildProcessTable(m.processes, m.width)
		}
		if len(m.appDrainEntries) > 0 {
			m.appDrainTable = buildAppDrainTable(m.appDrainEntries, m.width)
		}
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case tickMsg:
		m.tickCount++
		cmds := []tea.Cmd{
			cmdTick(),
			cmdFetchBattery,
		}
		if m.tickCount%3 == 0 || m.activeView == viewProcesses {
			cmds = append(cmds, cmdFetchProcesses)
		}
		if m.tickCount%20 == 0 {
			cmds = append(cmds, cmdFetchHistory)
		}
		return m, tea.Batch(cmds...)

	case batteryMsg:
		m.loading = false
		if msg.err != nil {
			m.lastErr = msg.err
		} else {
			m.battery = msg.data
			m.lastErr = nil
		}
		return m, nil

	case processMsg:
		if msg.err == nil && len(msg.procs) > 0 {
			m.processes = msg.procs
			m.procTable = buildProcessTable(m.processes, m.width)
			// Write to store so App Drain works without a running daemon.
			if time.Since(m.lastStoredAt) >= 60*time.Second {
				m.lastStoredAt = time.Now()
				batt := m.battery
				procs := msg.procs
				period := m.drainPeriod
				return m, func() tea.Msg {
					_ = store.Append(batt, procs)
					// Refresh App Drain data if that view is active.
					return cmdFetchAppDrain(period)()
				}
			}
		}
		return m, nil

	case historyMsg:
		if msg.err == nil {
			m.history = msg.records
			// Seed App Drain with history data on first load.
			if len(m.appDrainEntries) == 0 && len(msg.records) > 0 {
				entries := store.AggregateAppDrain(msg.records, 30)
				m.appDrainEntries = entries
				m.appDrainRecords = msg.records
				m.appDrainTable = buildAppDrainTable(entries, m.width)
			}
		}
		return m, nil

	case appDrainMsg:
		m.appDrainLoading = false
		if msg.err == nil {
			m.appDrainEntries = msg.entries
			m.appDrainRecords = msg.records
			m.appDrainTable = buildAppDrainTable(msg.entries, m.width)
		}
		return m, nil

	case tea.KeyMsg:
		// Popup intercepts all keys when open.
		if m.showAppPopup {
			switch msg.String() {
			case "esc", "enter", "q", " ":
				m.showAppPopup = false
			}
			return m, nil
		}

		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

		case "tab":
			m.activeView = (m.activeView + 1) % viewCount
			return m, m.onViewSwitch()

		case "1":
			m.activeView = viewDashboard
		case "2":
			m.activeView = viewHealth
		case "3":
			m.activeView = viewProcesses
		case "4":
			m.activeView = viewHistory
			return m, cmdFetchHistory
		case "5":
			m.activeView = viewAppDrain
			return m, m.onViewSwitch()

		// Period selector in App Drain view.
		case "[", "left":
			if m.activeView == viewAppDrain {
				if m.drainPeriod > 0 {
					m.drainPeriod--
					m.appDrainLoading = true
					return m, cmdFetchAppDrain(m.drainPeriod)
				}
			}
		case "]", "right":
			if m.activeView == viewAppDrain {
				if m.drainPeriod < periodCount-1 {
					m.drainPeriod++
					m.appDrainLoading = true
					return m, cmdFetchAppDrain(m.drainPeriod)
				}
			}

		// Table navigation.
		case "up", "k", "down", "j":
			if m.activeView == viewProcesses {
				var cmd tea.Cmd
				m.procTable, cmd = m.procTable.Update(msg)
				return m, cmd
			}
			if m.activeView == viewAppDrain {
				var cmd tea.Cmd
				m.appDrainTable, cmd = m.appDrainTable.Update(msg)
				return m, cmd
			}

		// Open popup on the selected app.
		case "enter", " ":
			if m.activeView == viewAppDrain && len(m.appDrainEntries) > 0 {
				m.showAppPopup = true
			}
		}
		return m, nil
	}

	return m, nil
}

// onViewSwitch returns the appropriate command when the active view changes.
func (m Model) onViewSwitch() tea.Cmd {
	switch m.activeView {
	case viewHistory:
		return cmdFetchHistory
	case viewAppDrain:
		m2 := m // immutable copy
		_ = m2
		return cmdFetchAppDrain(m.drainPeriod)
	}
	return nil
}

// selectedEntry returns the AppDrainEntry for the currently highlighted table row.
func (m Model) selectedEntry() (store.AppDrainEntry, bool) {
	cursor := m.appDrainTable.Cursor()
	if cursor < 0 || cursor >= len(m.appDrainEntries) {
		return store.AppDrainEntry{}, false
	}
	return m.appDrainEntries[cursor], true
}

// ── View ──────────────────────────────────────────────────────────────────────

func (m Model) View() string {
	if m.loading {
		return "\n  " + m.spinner.View() + " Fetching battery data...\n"
	}

	header := renderHeader(m.activeView, m.width)

	var body string
	switch m.activeView {
	case viewDashboard:
		body = renderDashboard(m)
	case viewHealth:
		body = renderHealth(m)
	case viewProcesses:
		body = renderProcesses(m)
	case viewHistory:
		body = renderHistory(m)
	case viewAppDrain:
		body = renderAppDrain(m)
	}

	footer := renderFooter(m)
	base := lipgloss.JoinVertical(lipgloss.Left, header, body, footer)

	// Overlay the popup if active.
	if m.showAppPopup {
		if entry, ok := m.selectedEntry(); ok {
			popup := renderAppPopup(entry, m.drainPeriod, m.width)
			return lipgloss.Place(
				m.width, m.height,
				lipgloss.Center, lipgloss.Center,
				popup,
				lipgloss.WithWhitespaceBackground(lipgloss.Color("#0D1117")),
			)
		}
		m.showAppPopup = false
	}

	return base
}

// ── Header / Footer ───────────────────────────────────────────────────────────

func renderHeader(active view, width int) string {
	tabs := []struct {
		key   string
		label string
		v     view
	}{
		{"1", "Dashboard", viewDashboard},
		{"2", "Health", viewHealth},
		{"3", "Processes", viewProcesses},
		{"4", "History", viewHistory},
		{"5", "App Drain", viewAppDrain},
	}

	var parts []string
	for _, t := range tabs {
		label := fmt.Sprintf("[%s] %s", t.key, t.label)
		if t.v == active {
			parts = append(parts, styleActiveTab.Render(label))
		} else {
			parts = append(parts, styleInactiveTab.Render(label))
		}
		parts = append(parts, " ")
	}

	title := lipgloss.NewStyle().Foreground(colorCyan).Bold(true).Render("MacBattery")
	tabRow := strings.Join(parts, "")
	sep := styleMuted.Render(strings.Repeat("─", width))

	return lipgloss.JoinVertical(lipgloss.Left,
		"  "+title,
		"  "+tabRow,
		sep,
		"",
	)
}

func renderFooter(m Model) string {
	var errStr string
	if m.lastErr != nil {
		errStr = "  " + styleCrit.Render("Error: "+m.lastErr.Error())
	}
	help := styleMuted.Render("  1-5/tab: switch  ·  q: quit  ·  ↑↓: scroll  ·  Enter: app graph (App Drain)")
	sep := styleMuted.Render(strings.Repeat("─", m.width))
	lines := []string{"", sep, help}
	if errStr != "" {
		lines = append(lines, errStr)
	}
	return strings.Join(lines, "\n")
}
