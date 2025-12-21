// Package iowait is a probe that monitors IO wait statistics.
package iowait

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/bpfstack/bpfstack/pkg/agent/core"
	"github.com/bpfstack/bpfstack/pkg/probes/common"
	"github.com/cilium/ebpf/link"
)

//go:generate go run github.com/cilium/ebpf/cmd/bpf2go -cflags "-I../../../headers" -target bpfel iowait bpf.c

// ProcessStats holds the statistics for the top iowait processes.
// The format is JSON.
type ProcessStats struct {
	PID      uint32 `json:"pid"`
	Comm     string `json:"comm"`
	WaitTime uint64 `json:"wait_time_ns"`
}

// Probe is the probe for collecting iowait statistics.
type Probe struct {
    common.CommonProbe
    // objs is the eBPF objects.
    objs *iowaitObjects
    // link is the link to the tracing program.
    link link.Link
}

// New creates a new Probe instance.
func New() core.Prober {
	return &Probe{
		CommonProbe: common.CommonProbe{
			Name: "iowait_top5",
			Description: "Collects the top 5 processes by IO wait time.",
			Version: "1.0.0",
		},
	}
}

// Load loads the eBPF objects and attaches the tracing program.
func (p *Probe) Load() error {
	// Initialize the objects struct
	p.objs = &iowaitObjects{}

	// Load the eBPF objects into the struct.
	// We pass 'p.objs' to be filled, and 'nil' for options.
	if err := loadIowaitObjects(p.objs, nil); err != nil {
		return fmt.Errorf("loading objects: %w", err)
	}

	// Attach the BPF program (tp_btf/sched_switch)
	l, err := link.AttachTracing(link.TracingOptions{
		Program: p.objs.SchedSwitch,
	})
	if err != nil {
		return fmt.Errorf("attaching tracepoint: %w", err)
	}
	p.link = l
	return nil
}

// Run collects iowait statistics and sends them to the output channel.
func (p *Probe) Run(ctx context.Context, outCh chan<- core.TelemetryEvent) error {
	// Collect statistics every 30 seconds
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			top5, err := p.collectAndSort()
			if err != nil {
				fmt.Printf("Error collecting stats: %v\n", err)
				continue
			}

			// If there is no data, skip sending
			if len(top5) == 0 {
				continue
			}

			// Convert the stats to JSON
			jsonData, err := json.Marshal(top5)
			if err != nil {
				fmt.Printf("Error marshaling json: %v\n", err)
				continue
			}

			outCh <- core.TelemetryEvent{
				ProbeName: "iowait_top5",
				Data:      string(jsonData),
			}
		}
	}
}

// collectAndSort iterates over the BPF map, collects stats, and returns the top 5 processes.
func (p *Probe) collectAndSort() ([]ProcessStats, error) {
	var stats []ProcessStats

	// Variables to store map key and value
	var key iowaitIowaitKeyT
	var val iowaitIowaitValT

	iter := p.objs.ResultMap.Iterate()
	for iter.Next(&key, &val) {
		stats = append(stats, ProcessStats{
			PID: key.Pid,
			// Use the local goString function to convert C string to Go string
			Comm:     goString(val.Comm[:]),
			WaitTime: val.TotalNs,
		})

		// IMPORTANT: Read data must be deleted to get new statistics for the next interval.
		// If cumulative statistics are needed (forever increasing), do not delete it.
		// Ignore deletion failures (e.g., race conditions where the entry is already gone)
		if err := p.objs.ResultMap.Delete(key); err != nil {
			return nil, fmt.Errorf("deleting map entry: %w", err)
		}
	}

	if err := iter.Err(); err != nil {
		return nil, err
	}

	// Sort by WaitTime in descending order
	sort.Slice(stats, func(i, j int) bool {
		return stats[i].WaitTime > stats[j].WaitTime
	})

	// Return only the top 5
	if len(stats) > 5 {
		return stats[:5], nil
	}
	return stats, nil
}

// Close closes the probe and cleans up resources.
func (p *Probe) Close() error {
	if p.link != nil {
		if err := p.link.Close(); err != nil {
			return fmt.Errorf("closing link: %w", err)
		}
	}
	if p.objs != nil {
		if err := p.objs.Close(); err != nil {
			return fmt.Errorf("closing objects: %w", err)
		}
	}
	return nil
}

// goString converts a C-style null-terminated int8 array to a Go string.
func goString(cBytes []int8) string {
	b := make([]byte, len(cBytes))
	for i, v := range cBytes {
		if v == 0 {
			return string(b[:i])
		}
		b[i] = byte(v)
	}
	return string(b)
}