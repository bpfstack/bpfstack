package helloworld

import (
	"context"
	"time"

	"github.com/bpfstack/bpfstack/pkg/agent/core"
)

type HelloWorldProbe struct {}

func New() core.Probe {
    return &HelloWorldProbe{}
}

func (p *HelloWorldProbe) Name() string {
    return "helloworld"
}

func (p *HelloWorldProbe) Load() error {
    return nil
}

func (p *HelloWorldProbe) Run(ctx context.Context, outCh chan<- core.TelemetryEvent) error {
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

func (p *HelloWorldProbe) Close() error {
	return nil
}