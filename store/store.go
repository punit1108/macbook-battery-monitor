package store

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/ppunit/mac_battery/collect"
)

// Record is a single data point written to the JSONL store.
type Record struct {
	Ts          time.Time         `json:"ts"`
	Pct         int               `json:"pct"`
	Charging    bool              `json:"charging"`
	TempC       float64           `json:"temp_c"`
	VoltageV    float64           `json:"voltage_v"`
	AmperageA   float64           `json:"amperage_a"`
	PowerInW    float64           `json:"power_in_w"`
	SystemLoadW float64           `json:"system_load_w"`
	TimeRemain  int               `json:"time_remaining"`
	AdapterW    int               `json:"adapter_watts"`
	TopProcs    []ProcSnapshot    `json:"top_procs,omitempty"`
}

// ProcSnapshot is a lightweight process entry stored in each Record.
type ProcSnapshot struct {
	PID   int     `json:"pid"`
	Cmd   string  `json:"cmd"`
	CPU   float64 `json:"cpu"`
	Mem   float64 `json:"mem"`
	Power float64 `json:"power"`
}

// DataDir returns the path to the data directory, creating it if needed.
func DataDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".mac_battery", "data")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	return dir, nil
}

// todayFile returns the path to today's JSONL file.
func todayFile() (string, error) {
	dir, err := DataDir()
	if err != nil {
		return "", err
	}
	name := time.Now().Format("2006-01-02") + ".jsonl"
	return filepath.Join(dir, name), nil
}

// Append writes a new record to today's JSONL log file.
func Append(batt collect.BatteryData, procs []collect.Process) error {
	r := Record{
		Ts:          time.Now().UTC(),
		Pct:         batt.CurrentCapacity,
		Charging:    batt.IsCharging,
		TempC:       batt.TemperatureC,
		VoltageV:    float64(batt.Voltage) / 1000.0,
		AmperageA:   float64(batt.Amperage) / 1000.0,
		PowerInW:    float64(batt.SystemPowerInMW) / 1000.0,
		SystemLoadW: float64(batt.SystemLoadMW) / 1000.0,
		TimeRemain:  batt.TimeRemaining,
		AdapterW:    batt.AdapterWatts,
	}
	for _, p := range procs {
		r.TopProcs = append(r.TopProcs, ProcSnapshot{
			PID:   p.PID,
			Cmd:   p.Command,
			CPU:   p.CPU,
			Mem:   p.Mem,
			Power: p.Power,
		})
	}

	path, err := todayFile()
	if err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	line, err := json.Marshal(r)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(f, "%s\n", line)
	return err
}

// ReadLast reads up to the last maxPoints records across today and yesterday.
func ReadLast(maxPoints int) ([]Record, error) {
	dir, err := DataDir()
	if err != nil {
		return nil, err
	}

	// Read today and yesterday.
	dates := []string{
		time.Now().Format("2006-01-02"),
		time.Now().AddDate(0, 0, -1).Format("2006-01-02"),
	}

	var all []Record
	for _, date := range dates {
		path := filepath.Join(dir, date+".jsonl")
		records, err := readFile(path)
		if err != nil {
			continue // file may not exist yet
		}
		all = append(records, all...) // prepend older records
	}

	if len(all) > maxPoints {
		all = all[len(all)-maxPoints:]
	}
	return all, nil
}

// AppDrainEntry holds aggregated per-app battery drain data.
type AppDrainEntry struct {
	Command      string
	TotalPower   float64 // sum of energy-impact scores across all snapshots
	TotalCPU     float64 // sum of CPU % samples
	Appearances  int     // number of snapshots the app appeared in
	RecordCount  int     // total records scanned (for percentage)
	AvgCPU       float64 // TotalCPU / Appearances
	AvgPower     float64 // TotalPower / Appearances
	SharePct     float64 // Appearances / RecordCount * 100
}

// AggregateAppDrain aggregates per-app drain from a slice of records.
// Returns entries sorted by TotalPower descending.
func AggregateAppDrain(records []Record) []AppDrainEntry {
	type agg struct {
		totalPower  float64
		totalCPU    float64
		appearances int
	}
	m := make(map[string]*agg)

	recordsWithProcs := 0
	for _, r := range records {
		if len(r.TopProcs) == 0 {
			continue
		}
		recordsWithProcs++
		seen := make(map[string]bool)
		for _, p := range r.TopProcs {
			cmd := p.Cmd
			if cmd == "" {
				continue
			}
			if !seen[cmd] {
				seen[cmd] = true
				if m[cmd] == nil {
					m[cmd] = &agg{}
				}
				m[cmd].appearances++
			}
			m[cmd].totalPower += p.Power
			m[cmd].totalCPU += p.CPU
		}
	}

	entries := make([]AppDrainEntry, 0, len(m))
	for cmd, a := range m {
		e := AppDrainEntry{
			Command:     cmd,
			TotalPower:  a.totalPower,
			TotalCPU:    a.totalCPU,
			Appearances: a.appearances,
			RecordCount: recordsWithProcs,
		}
		if a.appearances > 0 {
			e.AvgCPU = a.totalCPU / float64(a.appearances)
			e.AvgPower = a.totalPower / float64(a.appearances)
		}
		if recordsWithProcs > 0 {
			e.SharePct = float64(a.appearances) / float64(recordsWithProcs) * 100
		}
		entries = append(entries, e)
	}

	// Sort by TotalPower descending.
	for i := 1; i < len(entries); i++ {
		for j := i; j > 0 && entries[j].TotalPower > entries[j-1].TotalPower; j-- {
			entries[j], entries[j-1] = entries[j-1], entries[j]
		}
	}
	return entries
}

func readFile(path string) ([]Record, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var records []Record
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var r Record
		if err := json.Unmarshal(line, &r); err != nil {
			continue
		}
		records = append(records, r)
	}
	return records, scanner.Err()
}
