package collect

import (
	"os/exec"
	"strconv"
	"strings"
)

// Process holds per-process analytics data.
type Process struct {
	PID     int
	Command string
	CPU     float64 // percentage
	Mem     float64 // percentage
	Power   float64 // energy impact score from top
	User    string
}

// FetchProcesses runs top and ps to collect per-process data.
func FetchProcesses() ([]Process, error) {
	// top -l 1: one snapshot, sorted by power/energy impact
	out, err := exec.Command("top", "-l", "1", "-s", "0", "-n", "20", "-o", "power").Output()
	if err != nil {
		return fetchProcessesPS()
	}
	procs := ParseTop(string(out))
	if len(procs) == 0 {
		return fetchProcessesPS()
	}
	return procs, nil
}

// fetchProcessesPS is a fallback using ps when top parsing fails.
func fetchProcessesPS() ([]Process, error) {
	out, err := exec.Command("ps", "aux").Output()
	if err != nil {
		return nil, err
	}
	return parsePS(string(out)), nil
}

// ParseTop parses the output of top -l 1.
func ParseTop(output string) []Process {
	lines := strings.Split(output, "\n")

	// Find the header line (starts with "PID").
	headerIdx := -1
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "PID") {
			headerIdx = i
			break
		}
	}
	if headerIdx < 0 || headerIdx+1 >= len(lines) {
		return nil
	}

	headers := strings.Fields(lines[headerIdx])
	pidCol, cmdCol, cpuCol, powerCol := -1, -1, -1, -1

	for i, h := range headers {
		switch strings.ToUpper(h) {
		case "PID":
			pidCol = i
		case "COMMAND":
			cmdCol = i
		case "%CPU":
			cpuCol = i
		case "POWER":
			powerCol = i
		}
	}
	if pidCol < 0 || cmdCol < 0 {
		return nil
	}

	maxRequired := maxOf(pidCol, cmdCol, cpuCol)

	var procs []Process
	for _, line := range lines[headerIdx+1:] {
		if strings.TrimSpace(line) == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) <= maxRequired {
			continue
		}

		p := Process{}
		p.PID, _ = strconv.Atoi(fields[pidCol])
		p.Command = fields[cmdCol]
		if cpuCol >= 0 && cpuCol < len(fields) {
			p.CPU, _ = strconv.ParseFloat(fields[cpuCol], 64)
		}
		if powerCol >= 0 && powerCol < len(fields) {
			p.Power, _ = strconv.ParseFloat(fields[powerCol], 64)
		}
		procs = append(procs, p)
	}
	return procs
}

// parsePS parses `ps aux` output.
func parsePS(output string) []Process {
	lines := strings.Split(output, "\n")
	if len(lines) < 2 {
		return nil
	}

	var procs []Process
	for _, line := range lines[1:] { // skip header
		if strings.TrimSpace(line) == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 11 {
			continue
		}
		p := Process{}
		p.User = fields[0]
		p.PID, _ = strconv.Atoi(fields[1])
		p.CPU, _ = strconv.ParseFloat(fields[2], 64)
		p.Mem, _ = strconv.ParseFloat(fields[3], 64)
		// Command is everything from field 10 onward.
		p.Command = strings.Join(fields[10:], " ")
		// Trim to basename.
		parts := strings.Split(p.Command, "/")
		p.Command = parts[len(parts)-1]
		if len(p.Command) > 30 {
			p.Command = p.Command[:30]
		}
		procs = append(procs, p)
	}
	return procs
}

func maxOf(vals ...int) int {
	m := 0
	for _, v := range vals {
		if v > m {
			m = v
		}
	}
	return m
}
