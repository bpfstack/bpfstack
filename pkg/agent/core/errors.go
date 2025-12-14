// Package core contains the core logic for the agent.
package core

import (
	"errors"
	"fmt"
	"github.com/cilium/ebpf"
)

// ProbeNotFound returns an error if the probe is not found.
func ProbeNotFound(name string) error {
    return fmt.Errorf("probe %s not found", name)
}

// ConfigInvalid returns an error if the config is invalid.
func ConfigInvalid(err error) error {
    return fmt.Errorf("config invalid: %w", err)
}

// ProbeLoadFailed returns an error if the probe load failed.
func ProbeLoadFailed(name string,err error) error {
    var ve *ebpf.VerifierError
    if errors.As(err, &ve) {
        return fmt.Errorf("probe %s load failed(eBPF verifier error): %w", name, ve)
    }
    return fmt.Errorf("probe %s load failed: %w", name, err)
}

// ProbeRunFailed returns an error if the probe run failed.
func ProbeRunFailed(name string,err error) error {
    return fmt.Errorf("probe %s run failed: %w", name, err)
}

// FailedToStartProbe returns an error if the probe start failed.
func FailedToStartProbe(name string, err error) error {
    return fmt.Errorf("failed to start probe %s: %w", name, err)
}

// FailedToStopProbe returns an error if the probe stop failed.
func FailedToStopProbe(name string, err error) error {
    return fmt.Errorf("failed to stop probe %s: %w", name, err)
}

// FailedToCloseProbe returns an error if the probe close failed.
func FailedToCloseProbe(name string, err error) error {
    return fmt.Errorf("failed to close probe %s: %w", name, err)
}

// ProbePanic returns an error if the probe panicked.
func ProbePanic(name string, r any) error {
    return fmt.Errorf("probe %s panicked: %v", name, r)
}

// ProbeContextCanceled returns an error if the probe context canceled.
func ProbeContextCanceled(name string) error {
    return fmt.Errorf("probe %s context canceled", name)
}