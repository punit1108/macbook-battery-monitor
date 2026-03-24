package tui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ppunit/mac_battery/collect"
	"github.com/ppunit/mac_battery/store"
)

func cmdTick() tea.Cmd {
	return tea.Tick(3*time.Second, func(_ time.Time) tea.Msg {
		return tickMsg{}
	})
}

func cmdFetchBattery() tea.Msg {
	data, err := collect.FetchBattery()
	return batteryMsg{data: data, err: err}
}

func cmdFetchProcesses() tea.Msg {
	procs, err := collect.FetchProcesses()
	return processMsg{procs: procs, err: err}
}

func cmdFetchHistory() tea.Msg {
	records, err := store.ReadLast(200)
	return historyMsg{records: records, err: err}
}

func cmdFetchAppDrain(period drainPeriod) tea.Cmd {
	return func() tea.Msg {
		since := time.Now().Add(-period.duration())
		records, err := store.ReadSince(since)
		if err != nil {
			return appDrainMsg{err: err}
		}
		entries := store.AggregateAppDrain(records, 30)
		return appDrainMsg{entries: entries, records: records}
	}
}
