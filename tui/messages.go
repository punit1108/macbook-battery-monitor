package tui

import (
	"github.com/ppunit/mac_battery/collect"
	"github.com/ppunit/mac_battery/store"
)

// tickMsg fires on the 3-second refresh timer.
type tickMsg struct{}

// batteryMsg carries freshly fetched battery data.
type batteryMsg struct {
	data collect.BatteryData
	err  error
}

// processMsg carries freshly fetched process data.
type processMsg struct {
	procs []collect.Process
	err   error
}

// historyMsg carries historical records loaded from the store.
type historyMsg struct {
	records []store.Record
	err     error
}

// appDrainMsg carries aggregated app-drain data for a selected time period.
type appDrainMsg struct {
	entries []store.AppDrainEntry
	records []store.Record
	err     error
}
