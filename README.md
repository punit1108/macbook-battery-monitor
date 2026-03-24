# volt

A macOS battery analytics tool with a terminal UI. Runs a lightweight background daemon that continuously logs battery and process data, and a bubbletea TUI to explore it.

![Go](https://img.shields.io/badge/Go-1.24-00ADD8?logo=go) ![macOS](https://img.shields.io/badge/macOS-arm64%20%7C%20amd64-black?logo=apple)

## Features

- **Live dashboard** — charge %, temperature, voltage, current draw, power in/out
- **Battery health** — cycle count gauge, capacity vs design capacity, health %
- **Top processes** — real-time table of CPU and energy impact per process
- **Charge history** — ASCII chart of battery % over time (last 24h)
- **App drain** — historical ranking of which apps consumed the most battery, aggregated from daemon logs

## Requirements

- macOS (arm64 or x86_64)
- Git
- Go 1.21+ (installed automatically if Homebrew is present)

## Install

### One-line install (recommended)

```bash
curl -fsSL https://raw.githubusercontent.com/punit1108/volt/main/install.sh | bash
```

The script will:
1. Check for Go and install it via Homebrew if missing
2. Clone the repo, build the binary, and install it to `~/.local/bin/volt`
3. Add `~/.local/bin` to your `$PATH` if needed
4. Prompt to install the background daemon (recommended)

To install to a custom location:

```bash
INSTALL_DIR=/usr/local/bin curl -fsSL https://raw.githubusercontent.com/punit1108/volt/main/install.sh | bash
```

### Manual install

```bash
git clone https://github.com/punit1108/volt.git
cd volt
go build -o volt .
./volt install   # installs daemon + copies binary to ~/.local/bin
```

### Background daemon

The daemon collects battery and process data continuously, even when the TUI is closed. Without it, the App Drain history tab only shows data from sessions when the TUI was open.

```bash
volt install      # register as LaunchAgent, auto-start on login
volt uninstall    # remove daemon (collected data is preserved)
```

### Verify

```bash
launchctl list | grep volt   # should show a PID
tail -f ~/.volt/daemon.log   # live daemon output
```

## Usage

```
volt              Launch the TUI
volt daemon       Run the background collector (foreground)
volt install      Install as a LaunchAgent (auto-start on login)
volt uninstall    Remove the LaunchAgent and binary
```

### TUI key bindings

| Key | Action |
|-----|--------|
| `1` – `5` | Switch views |
| `Tab` | Cycle views |
| `↑` / `↓` / `j` / `k` | Scroll table (Processes, App Drain) |
| `q` / `Ctrl+C` | Quit |

### Views

| # | View | Description |
|---|------|-------------|
| 1 | Dashboard | Live battery status, power metrics, adapter info |
| 2 | Health | Cycle count, capacity health, lifetime stats |
| 3 | Processes | Top processes by CPU and energy impact (live) |
| 4 | History | Battery % chart over time from daemon logs |
| 5 | App Drain | Ranked table of apps by historical battery drain |

## Data collection

The daemon writes one JSON record per minute to `~/.volt/data/YYYY-MM-DD.jsonl`. Process data is sampled every 5 minutes (the more expensive `top` call).

```
~/.volt/
├── data/
│   ├── 2026-03-24.jsonl
│   └── 2026-03-25.jsonl
└── daemon.log
```

Each record looks like:

```json
{
  "ts": "2026-03-24T13:50:04Z",
  "pct": 84,
  "charging": true,
  "temp_c": 38.1,
  "voltage_v": 12.67,
  "amperage_a": 2.40,
  "power_in_w": 47.8,
  "system_load_w": 17.2,
  "time_remaining": 47,
  "adapter_watts": 94,
  "top_procs": [
    { "pid": 1234, "cmd": "Xcode", "cpu": 22.1, "power": 200 }
  ]
}
```

Uninstalling preserves all collected data — only the binary and plist are removed.

## Power consumption

The daemon is designed to have negligible impact on battery life:

- `ioreg` (battery read) runs every **60 seconds** — kernel data read, ~1ms
- `top` (process list) runs every **5 minutes**
- No network, no busy-wait; the process sleeps between ticks
- `GOMAXPROCS=1` limits OS thread usage
- Expected CPU usage: **< 0.1% average**

## Project structure

```
volt/
├── main.go          CLI entry point
├── collect/         ioreg and top parsers
├── store/           JSONL read/write and aggregation
├── daemon/          Background collection loop
├── agent/           LaunchAgent install/uninstall
└── tui/             Bubbletea TUI (5 views)
```

## Dependencies

- [bubbletea](https://github.com/charmbracelet/bubbletea) — TUI framework
- [lipgloss](https://github.com/charmbracelet/lipgloss) — terminal styling
- [bubbles](https://github.com/charmbracelet/bubbles) — table, progress bar, spinner components
