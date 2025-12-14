#include "vmlinux.h"
#include <bpf/bpf_helpers.h>
#include <bpf/bpf_tracing.h>
#include <bpf/bpf_core_read.h>

char __license[] SEC("license") = "GPL";

// TASK_UNINTERRUPTIBLE is the state where a process sleeps waiting for IO.
// NOTE: The value 0x0002 is standard for many kernels, but CO-RE might handle this better in the future.
#define TASK_UNINTERRUPTIBLE 0x0002

// Key structure for the result map
struct iowait_key_t {
    __u32 pid;
};

// Value structure containing total wait time and process name
struct iowait_val_t {
    __u64 total_ns;
    char comm[16];
};

// Map to record the timestamp when a process goes to sleep (start of wait)
struct {
    __uint(type, BPF_MAP_TYPE_HASH);
    __uint(max_entries, 10240);
    __type(key, __u32);
    __type(value, __u64);
} start_map SEC(".maps");

// Map to store accumulated IO wait time statistics
struct {
    __uint(type, BPF_MAP_TYPE_HASH);
    __uint(max_entries, 10240);
    __type(key, struct iowait_key_t);
    __type(value, struct iowait_val_t);
} result_map SEC(".maps");

// Tracepoint for process switching (sched_switch)
// This captures when a process is switched out (sleep) or switched in (wake up).
SEC("tp_btf/sched_switch")
int BPF_PROG(sched_switch, bool preempt, struct task_struct *prev, struct task_struct *next) {
    __u64 ts = bpf_ktime_get_ns();
    __u32 prev_pid = prev->tgid;
    __u32 next_pid = next->tgid;

    // If the previous process is entering TASK_UNINTERRUPTIBLE state,
    // it effectively means it is waiting for IO.
    long state = BPF_CORE_READ(prev, __state);
    
    // We check if the state matches TASK_UNINTERRUPTIBLE.
    // Note: Use bitwise AND (&) if you want to catch states that include this flag,
    // but strict equality (==) is also common for simple checks.
    if (state & TASK_UNINTERRUPTIBLE) {
        bpf_map_update_elem(&start_map, &prev_pid, &ts, BPF_ANY);
    }

    // If the next process (the one waking up) was in our start_map,
    // it means it just finished waiting.
    __u64 *start_ts = bpf_map_lookup_elem(&start_map, &next_pid);
    if (start_ts) {
        __u64 delta = ts - *start_ts;

        struct iowait_key_t key = { .pid = next_pid };
        struct iowait_val_t *val = bpf_map_lookup_elem(&result_map, &key);
        
        if (!val) {
            // If no entry exists, create a new one
            struct iowait_val_t new_val = {};
            new_val.total_ns = delta;
            
            bpf_probe_read_kernel_str(new_val.comm, sizeof(new_val.comm), next->comm);
            bpf_map_update_elem(&result_map, &key, &new_val, BPF_ANY);
        } else {
            val->total_ns += delta;
            bpf_probe_read_kernel_str(val->comm, sizeof(val->comm), next->comm);
        }

        // Clean up the start time to prevent memory leaks in the map
        bpf_map_delete_elem(&start_map, &next_pid);
    }

    return 0;
}