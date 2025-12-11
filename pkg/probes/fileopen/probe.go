// Package fileopen is a probe that monitors file opens.
package fileopen

//go:generate sh -c "go run github.com/cilium/ebpf/cmd/bpf2go fileopen bpf.c -- -I${HEADER_DIR}"

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/cilium/ebpf/link"
	"github.com/cilium/ebpf/ringbuf"
	"github.com/cilium/ebpf/rlimit"
	"github.com/bpfstack/bpfstack/pkg/agent/core"
)

// Probe is a probe that monitors file opens.
type Probe struct {
    // objs is the eBPF objects.
    objs fileopenObjects
    // link is the link to the eBPF program.
    link link.Link
    // reader is the reader for the ring buffer.
    reader *ringbuf.Reader
}

// New is a test implementation of the New method.
func New() core.Prober {
    return &Probe{}
}

// Name returns the name of the probe.
func (p *Probe) Name() string {
    return "fileopen"
}

// Load loads the eBPF program about file opens.
func (p *Probe) Load() error {
    if err := rlimit.RemoveMemlock(); err != nil {
        return err 
    }

    p.objs = fileopenObjects{}
    if err := loadFileopenObjects(&p.objs, nil); err != nil {
        return err
    }
    return nil
}

// Run send the file open events every 30 seconds to the output channel.
func (p *Probe) Run(ctx context.Context, outCh chan <- core.TelemetryEvent) error {
    // Attach to hook
    link, err := link.Tracepoint("syscalls", "sys_enter_openat", p.objs.HandleOpenat, nil)
    if err != nil {
        return err
    }
    p.link = link
    
    // Open the ring buffer for reading events.
    reader, err := ringbuf.NewReader(p.objs.Events)
    if err != nil {
        return err
    }
    p.reader = reader

    // Start the event reader - read events every 30 seconds.
    go func() {
        var event struct {
            Pid uint32
            Comm [16]byte
        }

        ticker := time.NewTicker(30 * time.Second)
        defer ticker.Stop()

        for {
            select {
            case <-ctx.Done():
                return
            case <-ticker.C:
                record, err := p.reader.Read()
                if err != nil {
                    if errors.Is(err, ringbuf.ErrClosed) {
                        return
                    }
                    log.Printf("failed to read ring buffer: %v", err)
                    continue
                }
                // Parse binary data.
                if err := binary.Read(bytes.NewReader(record.RawSample), binary.LittleEndian, &event); err != nil {
                    log.Printf("failed to read event: %v", err)
                    continue
                }

                comm := string(bytes.TrimRight(event.Comm[:], "\x00"))
                data := fmt.Sprintf("PID: %d, Command: %s", event.Pid, comm)
                outCh <- core.TelemetryEvent{
                    ProbeName: "fileopen",
                    Timestamp: time.Now().Unix(),
                    Data: data,
                }
            }
        }
    }()
    <- ctx.Done()
    return nil
}

// Close closes the ring buffer and the link to the eBPF program.
func (p *Probe) Close() error {
	if p.reader != nil { 
        if err := p.reader.Close(); err != nil {
            return err
        }
    }

	if p.link != nil { 
        if err := p.link.Close(); err != nil {
            return err
        }
    }

	if err := p.objs.Close(); err != nil {
		return err
	}
	return nil
}