// Package agent contains the agent client logic.
package agent

import (
	"context"
	"fmt"
	"log"
	"strconv"

	pb "github.com/bpfstack/bpfstack/api/proto/v1"
	"github.com/bpfstack/bpfstack/pkg/agent/core"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Client is the client for the agent.
type Client struct {
    // serverAddr is the address of the controller.
	serverAddr string
    // agentID is the ID of the agent.
	agentID    string
    // probeMgr is the manager for the probes.
	probeMgr   *core.ProbeManager
    // config is the configuration for the agent.
	config     map[string]bool
}

// NewClient creates a new agent client.
func NewClient(addr, id string, pm *core.ProbeManager, config map[string]bool) *Client {
	return &Client{
		serverAddr: addr,
		agentID:    id,
		probeMgr:   pm,
		config:     config,
	}
}

// Start starts the agent client.
func (c *Client) Start(ctx context.Context) error {
	// Create a gRPC connection.
	conn, err := grpc.NewClient(c.serverAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return err
	}
	defer func() {
		if err := conn.Close(); err != nil {
			log.Printf("Failed to close connection: %v", err)
		}
	}()

	client := pb.NewBpfstackClient(conn)

	// Create a bidirectional stream.
	stream, err := client.StreamConnect(ctx)
	if err != nil {
		return err
	}
	fmt.Println("Connected to Controller via gRPC")

	// If the probe manager make data available, send it to the controller.
	go func() {
		for event := range c.probeMgr.DataChan() {
			err := stream.Send(&pb.TelemetryEvent{
				AgentId:   c.agentID,
				ProbeName: event.ProbeName,
				Data:      event.Data,
				Timestamp: strconv.FormatInt(event.Timestamp, 10),
			})
			if err != nil {
				log.Printf("Failed to send telemetry: %v", err)
				return // 연결 끊김 처리 필요
			}
			// (요구사항) 결과값 출력
			fmt.Printf("[Sent to Controller] %s: %s\n", event.ProbeName, event.Data)
		}
	}()

	// If the controller sends a command, update the probe manager.
	for {
		cmd, err := stream.Recv()
		if err != nil {
			return fmt.Errorf("stream closed: %v", err)
		}

		fmt.Printf("[Command Received] Probe: %s, Enabled: %v\n", cmd.ProbeName, cmd.Enabled)

		c.config[cmd.ProbeName] = cmd.Enabled
		// The Reconcile function will start or stop the probes 
		// based on the configuration.
		if err := c.probeMgr.Reconcile(ctx, c.config); err != nil {
			log.Printf("Failed to reconcile probes: %v", err)
		}
	}
}