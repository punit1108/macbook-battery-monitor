package daemon

import (
	"fmt"
	"log"
	"os"
	"runtime"
	"time"

	"github.com/ppunit/volt/collect"
	"github.com/ppunit/volt/store"
)

// Run starts the background collection loop.
// Battery data is collected every 60 seconds.
// Process data is collected every 5 minutes (more expensive).
func Run() {
	// Limit to one OS thread to minimise power draw.
	runtime.GOMAXPROCS(1)

	// Ensure the data directory exists.
	if _, err := store.DataDir(); err != nil {
		fmt.Fprintln(os.Stderr, "volt daemon: cannot create data dir:", err)
		os.Exit(1)
	}

	log.SetPrefix("[volt daemon] ")
	log.SetFlags(log.Ldate | log.Ltime)
	log.Println("started")

	battTicker := time.NewTicker(60 * time.Second)
	procTicker := time.NewTicker(5 * time.Minute)
	defer battTicker.Stop()
	defer procTicker.Stop()

	var lastProcs []collect.Process

	// Collect immediately on start.
	collectBattery(&lastProcs)
	collectProcesses(&lastProcs)

	for {
		select {
		case <-battTicker.C:
			collectBattery(&lastProcs)
		case <-procTicker.C:
			collectProcesses(&lastProcs)
		}
	}
}

func collectBattery(procs *[]collect.Process) {
	batt, err := collect.FetchBattery()
	if err != nil {
		log.Println("battery fetch error:", err)
		return
	}
	if err := store.Append(batt, *procs); err != nil {
		log.Println("store write error:", err)
	}
}

func collectProcesses(procs *[]collect.Process) {
	p, err := collect.FetchProcesses()
	if err != nil {
		log.Println("process fetch error:", err)
		return
	}
	*procs = p
}
