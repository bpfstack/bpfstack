package core

import "context"

// Prober is the contract every plugin must implement.
type Prober interface {
    Name() string 
    Load() error 
    Run(ctx context.Context, outCh chan <- TelemetryEvent) error
    Close() error 
}