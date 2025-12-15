// Package main is the main package for the controller.
package main

import (
	"fmt"
	"log"
	"net"
	"time"

	pb "github.com/bpfstack/bpfstack/api/proto/v1"
	"google.golang.org/grpc"
)

// server is the server for the controller.
type server struct {
    // UnimplementedBpfstackServer is the unimplemented server for the Bpfstack service.
	pb.UnimplementedBpfstackServer
}

// StreamConnect is a bidirectional stream of telemetry events.
func (s *server) StreamConnect(stream pb.Bpfstack_StreamConnectServer) error {
	fmt.Println("New Agent Connected")

	// If the agent sends data, print it.
	go func() {
		for {
			data, err := stream.Recv()
			if err != nil {
				return
			}
			fmt.Printf("📊 [Controller DB] Received telemetry event: Agent=%s, Probe=%s, Payload=%s\n", 
				data.AgentId, data.ProbeName, data.Data)
		}
	}()

	// Send a command to the agent to stop the iowait probe.
	time.Sleep(10 * time.Second)
	
    // TODO: bpfstack-cli will request by cli with REST API.
	fmt.Println("[Controller Ops] User requested to START iowait probe!")
	err := stream.Send(&pb.Command{
		ProbeName: "iowait_top5",
		Enabled:   true,
	})
	if err != nil {
		fmt.Println("Failed to send command:", err)
	}

	// TODO: To keep the stream alive by waiting indefinitely
    // TODO: We need to manage the context properly.
	select {}
}

func main() {
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	pb.RegisterBpfstackServer(s, &server{})

	fmt.Println("Controller listening on :50051...")
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}