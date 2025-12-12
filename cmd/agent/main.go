// Package main is the main package for the agent.
package main

import (
	"fmt"
	"os"

	"github.com/bpfstack/bpfstack/internal/agent"
	"github.com/bpfstack/bpfstack/pkg/agent/core"
	"github.com/bpfstack/bpfstack/pkg/probes/fileopen"
	"github.com/bpfstack/bpfstack/pkg/probes/helloworld"
)

func main() {
    factories := map[string]core.ProbeFactoryFunc{
        "helloworld": helloworld.New,
        "fileopen": fileopen.New,
    }
	if err := agent.Run(factories); err != nil {
		fmt.Printf("Error: %v", err)
		os.Exit(1)
	}
}
