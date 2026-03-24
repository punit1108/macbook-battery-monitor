package collect

import (
	"os/exec"
	"regexp"
	"strconv"
)

// BatteryData holds all battery metrics collected from ioreg.
type BatteryData struct {
	CurrentCapacity         int     // percent (0-100)
	RawMaxCapacity          int     // mAh (AppleRawMaxCapacity)
	DesignCapacity          int     // mAh
	CycleCount              int
	DesignCycleCount        int     // typically 1000
	IsCharging              bool
	ExternalConnected       bool
	Condition               string  // "Normal", "Check Battery", etc.
	Voltage                 int     // millivolts
	Amperage                int     // milliamps; negative = discharging
	TemperatureRaw          int     // raw value in 0.1 K increments
	TemperatureC            float64 // (raw/10.0) - 273.15
	TimeRemaining           int     // minutes; 65535 = still calculating
	SystemPowerInMW         int64   // milliwatts from PowerTelemetryData
	SystemLoadMW            int64
	AdapterEfficiencyLossMW int64
	AdapterWatts            int
	AdapterName             string
	HealthPercent           float64 // RawMaxCapacity / DesignCapacity * 100
}

// Pre-compiled regexes for top-level keys.
var (
	reCurrentCap  = regexp.MustCompile(`"CurrentCapacity"\s*=\s*(\d+)`)
	reRawMaxCap   = regexp.MustCompile(`"AppleRawMaxCapacity"\s*=\s*(\d+)`)
	reDesignCap   = regexp.MustCompile(`"DesignCapacity"\s*=\s*(\d+)`)
	reCycleCount  = regexp.MustCompile(`"CycleCount"\s*=\s*(\d+)`)
	reDesignCycle = regexp.MustCompile(`"DesignCycleCount9C"\s*=\s*(\d+)`)
	reIsCharging  = regexp.MustCompile(`"IsCharging"\s*=\s*(Yes|No)`)
	reExtConn     = regexp.MustCompile(`"ExternalConnected"\s*=\s*(Yes|No)`)
	reCondition   = regexp.MustCompile(`"Condition"\s*=\s*"([^"]+)"`)
	reVoltage     = regexp.MustCompile(`"Voltage"\s*=\s*(\d+)`)
	reAmperage    = regexp.MustCompile(`"InstantAmperage"\s*=\s*(\d+)`)
	reTemp        = regexp.MustCompile(`"Temperature"\s*=\s*(\d+)`)
	reTimeRemain  = regexp.MustCompile(`"TimeRemaining"\s*=\s*(\d+)`)

	// PowerTelemetryData nested dict (inline on one line or multi-line).
	rePowerBlock = regexp.MustCompile(`"PowerTelemetryData"\s*=\s*\{([^}]+)\}`)
	reSysPowerIn = regexp.MustCompile(`"SystemPowerIn"\s*=\s*(\d+)`)
	reSysLoad    = regexp.MustCompile(`"SystemLoad"\s*=\s*(\d+)`)
	reEffLoss    = regexp.MustCompile(`"AdapterEfficiencyLoss"\s*=\s*(\d+)`)

	// AdapterDetails nested dict.
	reAdapterBlock = regexp.MustCompile(`"AdapterDetails"\s*=\s*\{([^}]+)\}`)
	reAdapterWatts = regexp.MustCompile(`"Watts"\s*=\s*(\d+)`)
	reAdapterName  = regexp.MustCompile(`"Name"\s*=\s*"([^"]+)"`)
)

// FetchBattery runs ioreg and parses battery data.
func FetchBattery() (BatteryData, error) {
	out, err := exec.Command("ioreg", "-rn", "AppleSmartBattery").Output()
	if err != nil {
		return BatteryData{}, err
	}
	return ParseIoreg(string(out)), nil
}

// ParseIoreg parses the ioreg output string into BatteryData.
func ParseIoreg(s string) BatteryData {
	var d BatteryData

	d.CurrentCapacity = matchInt(reCurrentCap, s)
	d.RawMaxCapacity = matchInt(reRawMaxCap, s)
	d.DesignCapacity = matchInt(reDesignCap, s)
	d.CycleCount = matchInt(reCycleCount, s)
	d.DesignCycleCount = matchInt(reDesignCycle, s)
	if d.DesignCycleCount == 0 {
		d.DesignCycleCount = 1000
	}

	d.IsCharging = matchBool(reIsCharging, s)
	d.ExternalConnected = matchBool(reExtConn, s)
	d.Condition = matchStr(reCondition, s)
	if d.Condition == "" {
		d.Condition = "Normal"
	}

	d.Voltage = matchInt(reVoltage, s)

	// InstantAmperage is stored as uint64 in the plist when negative (wraps around 2^64).
	if m := reAmperage.FindStringSubmatch(s); len(m) >= 2 {
		u, _ := strconv.ParseUint(m[1], 10, 64)
		d.Amperage = int(int64(u)) // correct sign via re-interpretation
	}

	d.TemperatureRaw = matchInt(reTemp, s)
	if d.TemperatureRaw > 0 {
		d.TemperatureC = float64(d.TemperatureRaw)/10.0 - 273.15
	}

	d.TimeRemaining = matchInt(reTimeRemain, s)

	// PowerTelemetryData nested block.
	if m := rePowerBlock.FindStringSubmatch(s); len(m) >= 2 {
		block := m[1]
		d.SystemPowerInMW = matchInt64(reSysPowerIn, block)
		d.SystemLoadMW = matchInt64(reSysLoad, block)
		d.AdapterEfficiencyLossMW = matchInt64(reEffLoss, block)
	}

	// AdapterDetails nested block.
	if m := reAdapterBlock.FindStringSubmatch(s); len(m) >= 2 {
		block := m[1]
		d.AdapterWatts = matchInt(reAdapterWatts, block)
		d.AdapterName = matchStr(reAdapterName, block)
	}

	if d.DesignCapacity > 0 && d.RawMaxCapacity > 0 {
		d.HealthPercent = float64(d.RawMaxCapacity) / float64(d.DesignCapacity) * 100.0
	}

	return d
}

func matchInt(re *regexp.Regexp, s string) int {
	m := re.FindStringSubmatch(s)
	if len(m) < 2 {
		return 0
	}
	v, _ := strconv.Atoi(m[1])
	return v
}

func matchInt64(re *regexp.Regexp, s string) int64 {
	m := re.FindStringSubmatch(s)
	if len(m) < 2 {
		return 0
	}
	v, _ := strconv.ParseInt(m[1], 10, 64)
	return v
}

func matchBool(re *regexp.Regexp, s string) bool {
	m := re.FindStringSubmatch(s)
	if len(m) < 2 {
		return false
	}
	return m[1] == "Yes"
}

func matchStr(re *regexp.Regexp, s string) string {
	m := re.FindStringSubmatch(s)
	if len(m) < 2 {
		return ""
	}
	return m[1]
}
