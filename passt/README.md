# passt

A tiny, self-contained reimplementation of the core idea behind
[passt/pasta](https://passt.top): give an isolated network namespace
connectivity to the outside world **without any privileged host
configuration** — no bridges, no `veth` pairs, no NAT rules, no `iptables`.

Like `pasta`, it creates a user + network namespace, makes a **TAP** device
inside it, configures an address and a default route, and runs a command in the
namespace. The parent process keeps the other end of the TAP and runs a small
**userspace TCP/IP stack** that translates the Layer‑2 Ethernet frames coming
out of the namespace into ordinary Layer‑4 sockets on the host.

```
   namespace                      passt parent (host netns)
 ┌───────────┐   Ethernet frames  ┌────────────────────────┐   real sockets
 │  command  │◄──────────────────►│  userspace TCP/IP stack │◄───────────────► Internet
 │  (shell)  │     via TAP fd     │  ARP · ICMP · TCP · UDP │
 └───────────┘                    └────────────────────────┘
   10.0.0.2                          gateway 10.0.0.1
```

## What it does

| Layer | Behavior |
|-------|----------|
| **ARP** | Answers requests for the gateway address. |
| **ICMP** | Replies to echo requests to the gateway, so `ping <gateway>` works. |
| **TCP** | Proxies outbound connections to real host sockets (a minimal TCP state machine: handshake, in‑order data, windowing, retransmission, FIN close). |
| **UDP** | Forwards outbound datagrams via host sockets, with per‑flow idle expiry (so DNS works). |

Because the namespace reaches everything through the gateway, the host needs no
special privileges beyond what an unprivileged user namespace already grants
over its own network namespace. The guest believes it is talking directly to
the remote IPs; the stack transparently impersonates them.

## Requirements

| | |
|---|---|
| OS | Linux (uses TAP, `CLONE_NEWUSER`/`CLONE_NEWNET`, netlink) |
| Kernel | Unprivileged user namespaces enabled |
| Privileges | None — no root, no `CAP_NET_ADMIN` on the host |
| Dependencies | `golang.org/x/sys`; **no** `iproute2` needed (interface is configured directly via ioctl + netlink) |

## Build

```sh
go build
```

## Usage

```
passt [flags] [command [args...]]
```

With no command it runs `$SHELL` (or `/bin/sh`).

```sh
passt curl -sS https://example.com      # fetch through the userspace stack
passt                                    # interactive shell with proxied networking
passt python3 -m http.server 0           # run anything; its sockets are proxied
```

Flags:

| Flag | Default | Description |
|------|---------|-------------|
| `-address` | `10.0.0.2` | address assigned inside the namespace |
| `-gateway` | `10.0.0.1` | gateway address (impersonated by the stack) |
| `-prefix` | `24` | subnet prefix length |
| `-mtu` | `1500` | interface MTU |
| `-ifname` | `pasta0` | TAP interface name inside the namespace |

## How it works

1. **`runParent`** creates a Unix socket pair and re‑executes the binary with
   `CLONE_NEWUSER | CLONE_NEWNET`, mapping the current uid/gid to root inside
   the new namespaces.
2. The **child** (`runChild`) opens `/dev/net/tun`, creates the `pasta0` TAP
   device, configures its address/MTU/route via ioctl + netlink, hands the TAP
   file descriptor back to the parent over the socket pair (`SCM_RIGHTS`), and
   `exec`s the requested command inside the namespace.
3. The **parent** runs the userspace stack on the TAP fd: it reads Ethernet
   frames, answers ARP and ICMP locally, and bridges TCP/UDP to real host
   sockets. A TAP file descriptor stays bound to the network namespace it was
   created in, so the parent can drive the namespace's link from the host.

## Limitations

This is a teaching‑sized implementation, not a replacement for passt/pasta:

- IPv4 only; outbound (namespace‑initiated) flows only — no inbound port forwarding.
- The TCP stack favors clarity over completeness: fixed advertised window, no
  window scaling/SACK/timestamps, simple time‑based retransmission, and a
  best‑effort close rather than a full TIME‑WAIT state machine. It relies on the
  TAP link being lossless, so dropped guest segments are simply re‑requested.
- ICMP only answers the gateway; it does not forward pings to remote hosts.

## Tests

```sh
go test        # unit tests for the pure helpers (checksums, sequence math, netlink encoding)
```

The proxying paths were verified end‑to‑end by running `curl` (small and
multi‑megabyte transfers), a UDP echo round‑trip, DNS resolution, and a raw
ICMP echo to the gateway from inside the namespace.
