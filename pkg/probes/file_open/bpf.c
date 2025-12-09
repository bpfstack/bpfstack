//go:build ignore
#include "vmlinux.h"
#include <bpf/bpf_helpers.h>

char __license[] SEC("license") = "Dual MIT/GPL";

struct event_data {
    // pid is the process ID of the process that opened the file.
    u32 pid;
    // comm is the name of the command that opened the file.
    // e.g) ls, cat, etc.
    char comm[16];
};

struct {
    // events is a ring buffer map to store the events.
    __uint(type, BPF_MAP_TYPE_RINGBUF);
    // max_entries is the maximum number of entries in the ring buffer.
    __uint(max_entries, 1 << 24);
} events SEC(".maps");

SEC("tracepoint/syscalls/sys_enter_openat")
int handle_openat(void *ctx) {
    struct event_data *data;

    data = bpf_ringbuf_reserve(&events, sizeof(*data), 0);
    if (!data) {
        return 0;
    }

    data->pid = bpf_get_current_pid_tgid() >> 32;
    bpf_get_current_comm(&data->comm, sizeof(data->comm));
    bpf_ringbuf_submit(data, 0);
    
    return 0;
}