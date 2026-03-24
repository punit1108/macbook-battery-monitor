package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ppunit/mac_battery/collect"
	"github.com/ppunit/mac_battery/store"
)

type view int

const (
	viewDashboard view = iota
	viewHealth
	viewProcesses
	viewHistory
	viewAppDrain
	viewCount // must be last
)

// Model is the root bubbletea model.
type Model struct {
	width  int
	height int

	activeView view

	battery   collect.BatteryData
	processes []collect.Process
	history   []store.Record

	procTable      table.Model
	appDrainTable  table.Model
	spinner        spinner.Model

	loading    bool
	lastErr    error
	tickCount  int // increments every 3s
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

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		cmdTick(),
		cmdFetchBattery,
		cmdFetchProcesses,
		cmdFetchHistory,
	)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if len(m.processes) > 0 {
			m.procTable = buildProcessTable(m.processes, m.width)
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
		// Refresh processes every 3 ticks (9s) or when on processes view.
		if m.tickCount%3 == 0 || m.activeView == viewProcesses {
			cmds = append(cmds, cmdFetchProcesses)
		}
		// Reload history every 20 ticks (60s).
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
		}
		return m, nil

	case historyMsg:
		if msg.err == nil {
			m.history = msg.records
			entries := store.AggregateAppDrain(msg.records)
			m.appDrainTable = buildAppDrainTable(entries, m.width)
		}
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "tab":
			m.activeView = (m.activeView + 1) % viewCount
			if m.activeView == viewHistory || m.activeView == viewAppDrain {
				return m, cmdFetchHistory
			}
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
			return m, cmdFetchHistory
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
		}
		return m, nil
	}

	return m, nil
}

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

	return lipgloss.JoinVertical(lipgloss.Left, header, body, footer)
}

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
	help := styleMuted.Render("  1-5/tab: switch  ·  q: quit  ·  ↑↓: scroll in Processes/App Drain")
	sep := styleMuted.Render(strings.Repeat("─", m.width))
	lines := []string{"", sep, help}
	if errStr != "" {
		lines = append(lines, errStr)
	}
	return strings.Join(lines, "\n")
}
