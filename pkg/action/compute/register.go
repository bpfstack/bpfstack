package compute

import (
	"github.com/bpfstack/bpfstack/pkg/action"
)

// RegisterActions registers all compute-related actions
func RegisterActions(registry *action.Registry) {
	registry.Register(NewCPUMetricsAction())
}
