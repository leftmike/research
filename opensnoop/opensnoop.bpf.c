//go:build ignore

/*
 * opensnoop - trace open/openat/openat2 syscalls.
 *
 * Attach to the syscalls/sys_{enter,exit}_{open,openat,openat2} tracepoints.
 * On enter we save the filename pointer and flags; on exit we emit an event
 * to the ring buffer with the resolved filename, return value, and comm.
 */

#include <linux/bpf.h>
#include <bpf/bpf_helpers.h>
#include <bpf/bpf_tracing.h>

typedef unsigned int       __u32;
typedef int                __s32;
typedef unsigned long long __u64;
typedef long long          __s64;

#define TASK_COMM_LEN 16
#define NAME_MAX      255

/* Matches struct event in main.go */
struct event {
	__u32 pid;
	__s32 ret;
	__s32 flags;
	char  comm[TASK_COMM_LEN];
	char  fname[NAME_MAX + 1];
};

/* Temporary per-thread state saved between enter and exit.
 * fname stores the user-space pointer as a plain integer to avoid
 * BTF pointer types that bpf2go cannot translate to Go. */
struct enter_args {
	__u64 fname; /* user-space pointer cast to integer */
	__s32 flags;
};

/* tid → enter_args */
struct {
	__uint(type, BPF_MAP_TYPE_HASH);
	__uint(max_entries, 10240);
	__type(key, __u32);
	__type(value, struct enter_args);
} starts SEC(".maps");

/* Events delivered to userspace */
struct {
	__uint(type, BPF_MAP_TYPE_RINGBUF);
	__uint(max_entries, 256 * 1024);
} events SEC(".maps");

/* Configurable filter: trace only this PID; 0 = all */
const volatile __u32 target_pid = 0;

/* Tracepoint raw-format structs (no vmlinux.h needed) */
struct trace_entry {
	__u16 type;
	__u8  flags;
	__u8  preempt_count;
	__s32 pid;
};

struct trace_event_raw_sys_enter {
	struct trace_entry ent;
	long               id;
	unsigned long      args[6];
	char               __data[0];
};

struct trace_event_raw_sys_exit {
	struct trace_entry ent;
	long               id;
	long               ret;
	char               __data[0];
};

/* openat2 uses struct open_how for flags */
struct open_how {
	__u64 flags;
	__u64 mode;
	__u64 resolve;
};

/* ------------------------------------------------------------------ */

static __always_inline int
handle_enter(const char *fname, __s32 flags)
{
	__u64 pid_tgid = bpf_get_current_pid_tgid();
	__u32 tgid     = pid_tgid >> 32;
	__u32 tid      = (__u32)pid_tgid;

	if (target_pid && tgid != target_pid)
		return 0;

	struct enter_args args = { .fname = (__u64)(unsigned long)fname, .flags = flags };
	bpf_map_update_elem(&starts, &tid, &args, BPF_ANY);
	return 0;
}

static __always_inline int
handle_exit(long ret)
{
	__u64 pid_tgid = bpf_get_current_pid_tgid();
	__u32 tgid     = pid_tgid >> 32;
	__u32 tid      = (__u32)pid_tgid;

	struct enter_args *args = bpf_map_lookup_elem(&starts, &tid);
	if (!args)
		return 0;

	struct event *e = bpf_ringbuf_reserve(&events, sizeof(*e), 0);
	if (e) {
		e->pid   = tgid;
		e->ret   = (__s32)ret;
		e->flags = args->flags;
		bpf_get_current_comm(&e->comm, sizeof(e->comm));
		bpf_probe_read_user_str(&e->fname, sizeof(e->fname),
					(const char *)(unsigned long)args->fname);
		bpf_ringbuf_submit(e, 0);
	}

	bpf_map_delete_elem(&starts, &tid);
	return 0;
}

/* ------------------------------------------------------------------ */
/* open (args[0]=filename, args[1]=flags)                             */

SEC("tracepoint/syscalls/sys_enter_open")
int trace_enter_open(struct trace_event_raw_sys_enter *ctx)
{
	return handle_enter((const char *)ctx->args[0], (int)ctx->args[1]);
}

SEC("tracepoint/syscalls/sys_exit_open")
int trace_exit_open(struct trace_event_raw_sys_exit *ctx)
{
	return handle_exit(ctx->ret);
}

/* ------------------------------------------------------------------ */
/* openat (args[0]=dfd, args[1]=filename, args[2]=flags)             */

SEC("tracepoint/syscalls/sys_enter_openat")
int trace_enter_openat(struct trace_event_raw_sys_enter *ctx)
{
	return handle_enter((const char *)ctx->args[1], (int)ctx->args[2]);
}

SEC("tracepoint/syscalls/sys_exit_openat")
int trace_exit_openat(struct trace_event_raw_sys_exit *ctx)
{
	return handle_exit(ctx->ret);
}

/* ------------------------------------------------------------------ */
/* openat2 (args[0]=dfd, args[1]=filename, args[2]=*open_how)        */

SEC("tracepoint/syscalls/sys_enter_openat2")
int trace_enter_openat2(struct trace_event_raw_sys_enter *ctx)
{
	struct open_how how = {};
	bpf_probe_read_user(&how, sizeof(how), (const void *)ctx->args[2]);
	return handle_enter((const char *)ctx->args[1], (int)how.flags);
}

SEC("tracepoint/syscalls/sys_exit_openat2")
int trace_exit_openat2(struct trace_event_raw_sys_exit *ctx)
{
	return handle_exit(ctx->ret);
}

char LICENSE[] SEC("license") = "GPL";
