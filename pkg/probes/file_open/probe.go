package file_open

//go:generate sh -c "go run github.com/cilium/ebpf/cmd/bpf2go file_open bpf.c -- -I${HEADER_DIR}"

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

type FileOpenProbe struct {
    objs file_openObjects
    link link.Link
    reader *ringbuf.Reader
}

func New() core.Probe {
    return &FileOpenProbe{}
}

func (p *FileOpenProbe) Name() string {
    return "file_open"
}

func (p *FileOpenProbe) Load() error {
    if err := rlimit.RemoveMemlock(); err != nil {
        return err 
    }

    p.objs = file_openObjects{}
    if err := loadFile_openObjects(&p.objs, nil); err != nil {
        return err
    }
    return nil
}

func (p *FileOpenProbe) Run(ctx context.Context, outCh chan <- core.TelemetryEvent) error {
    // Attach to hook
    link, err := link.Tracepoint("syscalls", "sys_enter_openat", p.objs.HandleOpenat, nil)
    if err != nil {
        return err
    }
    p.link = link
    
    // Open the ring buffer
    reader, err := ringbuf.NewReader(p.objs.Events)
    if err != nil {
        return err
    }
    p.reader = reader

    // Start the event reader - read every 30 seconds
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
                // Try to read one event from ring buffer
                record, err := p.reader.Read()
                if err != nil {
                    if errors.Is(err, ringbuf.ErrClosed) {
                        return
                    }
                    log.Printf("failed to read ring buffer: %v", err)
                    continue
                }
                // Parse Binary Data
                if err := binary.Read(bytes.NewReader(record.RawSample), binary.LittleEndian, &event); err != nil {
                    log.Printf("failed to read event: %v", err)
                    continue
                }

                comm := string(bytes.TrimRight(event.Comm[:], "\x00"))
                data := fmt.Sprintf("PID: %d, Command: %s", event.Pid, comm)
                outCh <- core.TelemetryEvent{
                    ProbeName: "file_open",
                    Timestamp: time.Now().Unix(),
                    Data: data,
                }
            }
        }
    }()
    <- ctx.Done()
    return nil
}

func (p *FileOpenProbe) Close() error {
	if p.reader != nil { p.reader.Close() }
	if p.link != nil { p.link.Close() }
	p.objs.Close()
	return nil
}