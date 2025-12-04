package compute

import (
	"github.com/bpfstack/bpfstack/pkg/action"
)

// RegisterActions registers all compute-related actions
func RegisterActions(registry *action.Registry) {
	if err := registry.Register(NewCPUMetricsAction()); err != nil {
		// Log error but don't fail - allow partial registration
		_ = err
	}
}
