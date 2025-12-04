package compute

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/bpfstack/bpfstack/pkg/action"
	"github.com/bpfstack/bpfstack/pkg/logger"
)

// CPUMetricsAction collects CPU metrics
type CPUMetricsAction struct {
	*action.BaseAction
	stopCh      chan struct{}
	wg          sync.WaitGroup
	prevStats   *CPUStats
	statsMutex  sync.Mutex
}

// NewCPUMetricsAction creates a new CPU metrics action
func NewCPUMetricsAction() *CPUMetricsAction {
	return &CPUMetricsAction{
		BaseAction: action.NewBaseAction("cpu_metrics"),
	}
}

// Init initializes the CPU metrics action
func (c *CPUMetricsAction) Init() error {
	c.stopCh = make(chan struct{})
	c.LogInfo("initialized CPU metrics collector")
	return nil
}

func (c *CPUMetricsAction) Start() error {
	c.LogInfo("starting CPU metrics collector")

	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				c.collectCPUMetrics()
			case <-c.stopCh:
				return
			}
		}
	}()

	return nil
}

// Stop stops the CPU metrics collector
func (c *CPUMetricsAction) Stop() error {
	c.LogInfo("stopping CPU metrics collector")
	close(c.stopCh)
	c.wg.Wait()
	return nil
}

// collectCPUMetrics collects and logs CPU usage metrics
func (c *CPUMetricsAction) collectCPUMetrics() {
	stats, err := readCPUStats()
	if err != nil {
		c.LogError("failed to read CPU stats", logger.Fields{
			"error": err.Error(),
		})
		return
	}

	c.statsMutex.Lock()
	prevStats := c.prevStats
	c.prevStats = stats
	c.statsMutex.Unlock()

	// If this is the first reading, we can't calculate usage yet
	if prevStats == nil {
		return
	}

	// Calculate CPU usage based on the difference
	usage := calculateCPUUsage(prevStats, stats)
	
	c.LogInfo("collected CPU metrics", logger.Fields{
		"cpu_usage": fmt.Sprintf("%.2f%%", usage),
	})
}

// readCPUStats reads CPU statistics from /proc/stat
func readCPUStats() (*CPUStats, error) {
	file, err := os.Open("/proc/stat")
	if err != nil {
		return nil, fmt.Errorf("failed to open /proc/stat: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "cpu ") {
			// Parse the "cpu" line which represents total CPU usage
			fields := strings.Fields(line)
			if len(fields) < 8 {
				return nil, fmt.Errorf("invalid cpu line format")
			}

			stats := &CPUStats{Name: "total"}
			var err error

			// Parse fields: cpu user nice system idle iowait irq softirq steal
			if stats.User, err = parseUint64(fields[1]); err != nil {
				return nil, fmt.Errorf("failed to parse user: %w", err)
			}
			if stats.Nice, err = parseUint64(fields[2]); err != nil {
				return nil, fmt.Errorf("failed to parse nice: %w", err)
			}
			if stats.System, err = parseUint64(fields[3]); err != nil {
				return nil, fmt.Errorf("failed to parse system: %w", err)
			}
			if stats.Idle, err = parseUint64(fields[4]); err != nil {
				return nil, fmt.Errorf("failed to parse idle: %w", err)
			}
			if stats.IOWait, err = parseUint64(fields[5]); err != nil {
				return nil, fmt.Errorf("failed to parse iowait: %w", err)
			}
			if stats.IRQ, err = parseUint64(fields[6]); err != nil {
				return nil, fmt.Errorf("failed to parse irq: %w", err)
			}
			if stats.SoftIRQ, err = parseUint64(fields[7]); err != nil {
				return nil, fmt.Errorf("failed to parse softirq: %w", err)
			}
			if len(fields) > 8 {
				if stats.Steal, err = parseUint64(fields[8]); err != nil {
					return nil, fmt.Errorf("failed to parse steal: %w", err)
				}
			}

			return stats, nil
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read /proc/stat: %w", err)
	}

	return nil, fmt.Errorf("cpu line not found in /proc/stat")
}

// parseUint64 parses a string to uint64
func parseUint64(s string) (Nanoseconds, error) {
	val, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return 0, err
	}
	return Nanoseconds(val), nil
}

// calculateCPUUsage calculates CPU usage percentage between two CPUStats readings
func calculateCPUUsage(prev, curr *CPUStats) float64 {
	prevTotal := prev.Total()
	currTotal := curr.Total()
	
	prevActive := prev.Active()
	currActive := curr.Active()

	// Calculate the difference
	totalDiff := currTotal - prevTotal
	activeDiff := currActive - prevActive

	if totalDiff == 0 {
		return 0.0
	}

	// CPU usage is the percentage of active time over total time
	return float64(activeDiff) / float64(totalDiff) * 100.0
}

