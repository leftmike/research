//go:build ignore

/*
 * execsnoop - trace execve/execveat syscalls.
 *
 * Attach to the syscalls/sys_{enter,exit}_{execve,execveat} tracepoints.
 * On enter we save the filename and argv pointers; on exit we emit an event
 * to the ring buffer with the resolved command line, return value, and comm.
 * A sched_process_fork tracepoint maintains a pid→ppid map so each event
 * can include the parent PID without requiring vmlinux BTF.
 */

#include <linux/bpf.h>
#include "bpf_helpers.h"

typedef unsigned int       __u32;
typedef int                __s32;
typedef unsigned long long __u64;

#define TASK_COMM_LEN 16
#define NAME_MAX      255
#define ARGSIZE       128
#define MAXARGS       20

/* Matches struct event in main.go exactly. */
struct event {
	__u32 pid;
	__u32 ppid;
	__u32 uid;
	__s32 ret;
	__u32 args_count;
	char  comm[TASK_COMM_LEN];
	char  filename[NAME_MAX + 1];
	char  args[ARGSIZE * MAXARGS];
};

/* Temporary per-thread state saved between enter and exit. */
struct enter_args {
	__u64 fname; /* user-space filename pointer as integer */
	__u64 argv;  /* user-space argv pointer as integer */
};

/* tid → enter_args */
struct {
	__uint(type, BPF_MAP_TYPE_HASH);
	__uint(max_entries, 10240);
	__type(key, __u32);
	__type(value, struct enter_args);
} starts SEC(".maps");

/* pid → ppid (maintained via sched_process_fork) */
struct {
	__uint(type, BPF_MAP_TYPE_HASH);
	__uint(max_entries, 10240);
	__type(key, __u32);
	__type(value, __u32);
} child_map SEC(".maps");

/* Events delivered to userspace */
struct {
	__uint(type, BPF_MAP_TYPE_RINGBUF);
	__uint(max_entries, 1 << 20);
} events SEC(".maps");

/* Configurable filters; 0 / 0xFFFFFFFF means "all". */
const volatile __u32 target_pid = 0;
const volatile __u32 target_uid = 0xFFFFFFFF;

/* Tracepoint raw-format structs (no vmlinux.h needed). */
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

struct trace_event_raw_sched_process_fork {
	struct trace_entry ent;
	char  parent_comm[TASK_COMM_LEN];
	__s32 parent_pid;
	char  child_comm[TASK_COMM_LEN];
	__s32 child_pid;
};

struct trace_event_raw_sched_process_template {
	struct trace_entry ent;
	char  comm[TASK_COMM_LEN];
	__s32 pid;
	__s32 prio;
};

/* ------------------------------------------------------------------ */

static __always_inline int
handle_enter(const char *filename, const char *const *argv)
{
	__u64 pid_tgid = bpf_get_current_pid_tgid();
	__u32 tgid     = pid_tgid >> 32;
	__u32 tid      = (__u32)pid_tgid;

	if (target_pid && tgid != target_pid)
		return 0;

	__u64 uid_gid = bpf_get_current_uid_gid();
	__u32 uid = (__u32)uid_gid;
	if (target_uid != 0xFFFFFFFF && uid != target_uid)
		return 0;

	struct enter_args ea = {
		.fname = (__u64)(unsigned long)filename,
		.argv  = (__u64)(unsigned long)argv,
	};
	bpf_map_update_elem(&starts, &tid, &ea, BPF_ANY);
	return 0;
}

static __always_inline int
handle_exit(long ret)
{
	__u64 pid_tgid = bpf_get_current_pid_tgid();
	__u32 tgid     = pid_tgid >> 32;
	__u32 tid      = (__u32)pid_tgid;

	struct enter_args *ea = bpf_map_lookup_elem(&starts, &tid);
	if (!ea)
		return 0;

	struct event *e = bpf_ringbuf_reserve(&events, sizeof(*e), 0);
	if (!e)
		goto cleanup;

	__u64 uid_gid = bpf_get_current_uid_gid();
	e->pid = tgid;
	e->uid = (__u32)uid_gid;
	e->ret = (__s32)ret;

	__u32 *ppidp = bpf_map_lookup_elem(&child_map, &tgid);
	e->ppid = ppidp ? *ppidp : 0;

	bpf_get_current_comm(&e->comm, sizeof(e->comm));
	bpf_probe_read_user_str(&e->filename, sizeof(e->filename),
				(const char *)(unsigned long)ea->fname);

	/* Read up to MAXARGS argv entries into fixed-size slots. */
	const char *argp;
	e->args_count = 0;

#pragma unroll
	for (int i = 0; i < MAXARGS; i++) {
		if (bpf_probe_read_user(&argp, sizeof(argp),
				(void *)(ea->argv + (__u64)i * sizeof(__u64))) < 0)
			break;
		if (!argp)
			break;
		if (bpf_probe_read_user_str(e->args + i * ARGSIZE, ARGSIZE, argp) < 0)
			break;
		e->args_count = i + 1;
	}

	bpf_ringbuf_submit(e, 0);

cleanup:
	bpf_map_delete_elem(&starts, &tid);
	return 0;
}

/* ------------------------------------------------------------------ */

SEC("tracepoint/sched/sched_process_fork")
int trace_process_fork(struct trace_event_raw_sched_process_fork *ctx)
{
	__u32 child  = (__u32)ctx->child_pid;
	__u32 parent = (__u32)ctx->parent_pid;
	bpf_map_update_elem(&child_map, &child, &parent, BPF_ANY);
	return 0;
}

SEC("tracepoint/sched/sched_process_exit")
int trace_process_exit(struct trace_event_raw_sched_process_template *ctx)
{
	__u32 pid = (__u32)ctx->pid;
	bpf_map_delete_elem(&child_map, &pid);
	return 0;
}

/* execve(filename, argv, envp): args[0]=filename, args[1]=argv */

SEC("tracepoint/syscalls/sys_enter_execve")
int trace_enter_execve(struct trace_event_raw_sys_enter *ctx)
{
	return handle_enter((const char *)ctx->args[0],
			    (const char *const *)ctx->args[1]);
}

SEC("tracepoint/syscalls/sys_exit_execve")
int trace_exit_execve(struct trace_event_raw_sys_exit *ctx)
{
	return handle_exit(ctx->ret);
}

/* execveat(dirfd, filename, argv, envp, flags): args[1]=filename, args[2]=argv */

SEC("tracepoint/syscalls/sys_enter_execveat")
int trace_enter_execveat(struct trace_event_raw_sys_enter *ctx)
{
	return handle_enter((const char *)ctx->args[1],
			    (const char *const *)ctx->args[2]);
}

SEC("tracepoint/syscalls/sys_exit_execveat")
int trace_exit_execveat(struct trace_event_raw_sys_exit *ctx)
{
	return handle_exit(ctx->ret);
}

char LICENSE[] SEC("license") = "GPL";
