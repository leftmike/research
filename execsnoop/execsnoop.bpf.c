//go:build ignore

/*
 * execsnoop - trace execve/execveat syscalls.
 *
 * Attach to the syscalls/sys_{enter,exit}_{execve,execveat} tracepoints.
 * On enter we copy the filename and argv strings into a per-CPU scratch
 * buffer (strings must be read while the caller's address space is still
 * live — after a successful execve the original mappings are gone).  On
 * exit we emit an event to the ring buffer with the saved command line,
 * return value, and comm.
 * A sched_process_fork tracepoint maintains a pid→ppid map so each event
 * can include the parent PID without requiring vmlinux BTF.
 *
 * Inode resolution
 * ----------------
 * Go userspace loads kernel BTF at startup and discovers the byte offsets of
 * task_struct.mm, mm_struct.exe_file, file.f_inode, inode.i_ino, inode.i_sb,
 * and super_block.s_dev.  Those values are written into the const-volatile
 * variables below before the program is loaded.  handle_exit() then walks
 * the chain with bpf_probe_read_kernel().  Kernel BTF is required; if it is
 * unavailable Go exits before loading this program.
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

/*
 * Matches struct event in main.go exactly.
 * inode is first so __u64 sits at offset 0 with no padding.
 * Layout: inode(8)+dev(4)+pid(4)+ppid(4)+uid(4)+ret(4)+args_count(4)+
 *         comm(16)+filename(256)+args(2560) = 2864 bytes.
 */
struct event {
	__u64 inode;
	__u64 cgroup_id;
	__u32 dev;
	__u32 pid;
	__u32 ppid;
	__u32 uid;
	__s32 ret;
	__u32 args_count;
	char  comm[TASK_COMM_LEN];
	char  filename[NAME_MAX + 1];
	char  args[ARGSIZE * MAXARGS];
};

/*
 * Temporary per-thread state saved between enter and exit.
 * Strings are read at sys_enter while the caller's address space is still
 * live; the struct is too large (~2.8 KB) for the 512-byte BPF stack, so
 * it lives in a per-CPU scratch array.
 */
struct enter_args {
	__u32 args_count;
	char  filename[NAME_MAX + 1];
	char  args[ARGSIZE * MAXARGS];
};

/* Per-CPU scratch buffer for building enter_args without hitting the stack limit. */
struct {
	__uint(type, BPF_MAP_TYPE_PERCPU_ARRAY);
	__uint(max_entries, 1);
	__type(key, __u32);
	__type(value, struct enter_args);
} scratch SEC(".maps");

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

/*
 * Byte offsets into kernel structs for the inode walk:
 *   task_struct  → mm       (off_task_mm)
 *   mm_struct    → exe_file (off_mm_exefile)
 *   file         → f_inode  (off_file_inode)
 *   inode        → i_ino    (off_inode_ino)
 *
 * Set by Go userspace from kernel BTF before the program is loaded; Go
 * exits before this program is loaded if any offset cannot be resolved.
 */
const volatile __u32 off_task_mm    = 0;
const volatile __u32 off_mm_exefile = 0;
const volatile __u32 off_file_inode = 0;
const volatile __u32 off_inode_ino  = 0;
const volatile __u32 off_inode_sb   = 0;
const volatile __u32 off_sb_dev     = 0;

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

/*
 * Read filename and argv strings from user space into the per-CPU scratch
 * buffer, then stash it in the starts map keyed by tid.
 *
 * Strings MUST be read here (sys_enter) because after a successful execve
 * the caller's address space is replaced and bpf_probe_read_user would
 * read from the new process's memory or fail entirely.
 */
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

	__u32 zero = 0;
	struct enter_args *ea = bpf_map_lookup_elem(&scratch, &zero);
	if (!ea)
		return 0;

	bpf_probe_read_user_str(ea->filename, sizeof(ea->filename), filename);

	const char *argp;
	ea->args_count = 0;

#pragma unroll
	for (int i = 0; i < MAXARGS; i++) {
		if (bpf_probe_read_user(&argp, sizeof(argp), argv + i) < 0)
			break;
		if (!argp)
			break;
		if (bpf_probe_read_user_str(ea->args + i * ARGSIZE, ARGSIZE, argp) < 0)
			break;
		ea->args_count = i + 1;
	}

	bpf_map_update_elem(&starts, &tid, ea, BPF_ANY);
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
	bpf_probe_read_kernel(e->filename, sizeof(e->filename), ea->filename);
	e->args_count = ea->args_count;
	bpf_probe_read_kernel(e->args, sizeof(e->args), ea->args);

	/*
	 * Walk task_struct → mm_struct → file → inode using the byte offsets
	 * discovered from kernel BTF at load time.
	 */
	e->cgroup_id = bpf_get_current_cgroup_id();
	e->inode = 0;
	e->dev   = 0;
	if (off_task_mm && off_mm_exefile && off_file_inode) {
		__u64 task = bpf_get_current_task();
		__u64 mm   = 0;
		if (!bpf_probe_read_kernel(&mm, sizeof(mm),
					   (void *)(task + off_task_mm)) && mm) {
			__u64 exefile = 0;
			if (!bpf_probe_read_kernel(&exefile, sizeof(exefile),
						   (void *)(mm + off_mm_exefile)) && exefile) {
				__u64 finode = 0;
				if (!bpf_probe_read_kernel(&finode, sizeof(finode),
							   (void *)(exefile + off_file_inode)) && finode) {
					if (off_inode_ino) {
						__u64 ino = 0;
						if (!bpf_probe_read_kernel(&ino, sizeof(ino),
									   (void *)(finode + off_inode_ino)))
							e->inode = ino;
					}
					if (off_inode_sb && off_sb_dev) {
						__u64 sb = 0;
						if (!bpf_probe_read_kernel(&sb, sizeof(sb),
									   (void *)(finode + off_inode_sb)) && sb) {
							__u32 dev = 0;
							if (!bpf_probe_read_kernel(&dev, sizeof(dev),
										   (void *)(sb + off_sb_dev)))
								e->dev = dev;
						}
					}
				}
			}
		}
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
