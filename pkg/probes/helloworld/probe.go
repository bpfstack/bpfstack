// Package helloworld is a test implementation of the Prober interface.
package helloworld

import (
	"context"
	"time"

	"github.com/bpfstack/bpfstack/pkg/agent/core"
	"github.com/bpfstack/bpfstack/pkg/probes/common"
)

// Probe is a test implementation of the Prober interface.
type Probe struct {
	common.CommonProbe
}

// New is a test implementation of the New method.
func New() core.Prober {
    return &Probe{
        CommonProbe: common.CommonProbe{
            Name: "helloworld",
            Description: "A test probe that sends a 'helloworld!!!' message every 10 seconds.",
            Version: "1.0.0",
        },
    }
}

// Load is a test implementation of the Load method.
func (p *Probe) Load() error {
    return nil
}

// Run is a test implementation of the Run method.
// It sends a "helloworld!!!" message with the current time every 10 seconds.
func (p *Probe) Run(ctx context.Context, outCh chan<- core.TelemetryEvent) error {
	ticker := time.NewTicker(10 * time.Second)
	for {
		select {
		case <-ctx.Done():
			return nil
		case t := <-ticker.C:

			outCh <- core.TelemetryEvent{
				ProbeName: "helloworld",
				Data:   "helloworld!!!: " + t.Format(time.TimeOnly),
			}
		}
	}
}

// Close is a test implementation of the Close method.
// It does nothing.
func (p *Probe) Close() error {
	return nil
}