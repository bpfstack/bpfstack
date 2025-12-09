package core

import "context"

// Probe is the contract every plugin must implement.
type Probe interface {
    Name() string 
    Load() error 
    Run(ctx context.Context, outCh chan <- TelemetryEvent) error
    Close() error 
}