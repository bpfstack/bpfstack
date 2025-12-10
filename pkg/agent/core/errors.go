package core

import (
	"errors"
	"fmt"
	"github.com/cilium/ebpf"
)

func ProbeNotFound(name string) error {
    return fmt.Errorf("probe %s not found", name)
}

func ConfigInvalid(err error) error {
    return fmt.Errorf("config invalid: %w", err)
}

func ProbeLoadFailed(name string,err error) error {
    var ve *ebpf.VerifierError
    if errors.As(err, &ve) {
        return fmt.Errorf("probe %s load failed(eBPF verifier error): %w", name, ve)
    }
    return fmt.Errorf("probe %s load failed: %w", name, err)
}

func ProbeRunFailed(name string,err error) error {
    return fmt.Errorf("probe %s run failed: %w", name, err)
}

func FailedToStartProbe(name string, err error) error {
    return fmt.Errorf("failed to start probe %s: %w", name, err)
}

func FailedToStopProbe(name string, err error) error {
    return fmt.Errorf("failed to stop probe %s: %w", name, err)
}

func FailedToCloseProbe(name string, err error) error {
    return fmt.Errorf("failed to close probe %s: %w", name, err)
}

func ProbePanic(name string, r any) error {
    return fmt.Errorf("probe %s panicked: %v", name, r)
}

func ProbeContextCanceled(name string) error {
    return fmt.Errorf("probe %s context canceled", name)
}