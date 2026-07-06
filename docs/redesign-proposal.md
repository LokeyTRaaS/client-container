# Proposal: refocus this repo as a node entropy agent

Status: proposed
Date: 2026-07-06

## Why this document exists

The current design of this repo (operator + CRD + sidecar injection + custom
device nodes) has problems that can't be patched away, and before building
anything more on top of it we should decide what this project actually is.
This document explains the reasoning in plain terms so the decision can be
made and challenged without needing deep Kubernetes or kernel background.

## How programs get randomness (the 2-minute version)

Almost no application invents its own randomness. They ask the operating
system kernel, either through the `getrandom()` system call or by reading
`/dev/urandom`. Every mainstream crypto library (OpenSSL, Go, Java, ...) does
this. The kernel maintains an internal "entropy pool" that it mixes various
unpredictable inputs into, and answers all of those requests from it.

So if you want LoKey's hardware randomness to reach applications, there are
exactly three ways in:

1. **The app calls the LoKey REST API itself.** Works today, main repo,
   nothing needed in Kubernetes. Only possible if you control the app's code.
2. **Feed the kernel's entropy pool on every node.** Then every application
   on that node benefits automatically, with zero changes, because they all
   ask the kernel anyway. Linux has a standard interface for this
   (the `RNDADDENTROPY` ioctl on `/dev/random`) and a well-known reference
   implementation: `rngd` from rng-tools.
3. **Expose a custom device file (e.g. `/dev/lokeyrng`) inside pods.** This
   is what the current sidecar design attempts.

## What's wrong with the current design (option 3)

- The operator injects sidecar containers by patching *running* pods. The
  Kubernetes API rejects this — the container list of a running pod is
  immutable. This feature has never worked. The correct mechanism (a
  mutating admission webhook) is a significantly bigger component.
- Even if injection worked, the device volume is only mounted into the
  injected containers, not the application containers, so apps can't see the
  device. Mounting it at `/dev` in app containers would hide their real
  `/dev/null` and `/dev/urandom` and break them.
- A device node created with `mknod` only works if a kernel driver is
  registered at that device number. Nothing in this design provides one.
- Most fundamentally: nobody needs option 3. Apps either use the kernel
  (option 2 covers them) or can call the API (option 1 covers them).

## Do nodes even need entropy feeding?

Honest caveat: since Linux kernel 5.6 (2020), `getrandom()` and
`/dev/urandom` are always properly seeded, and "entropy starvation" is
essentially a solved problem on modern systems. Feeding hardware entropy
into nodes is therefore a **defense-in-depth and compliance** feature —
provable hardware entropy source for regulated industries, gambling
certification, air-gapped deployments — not a fix for broken randomness.
The docs and marketing should say exactly that, because kernel-savvy readers
will dismiss the project if it overclaims.

## Proposal

Refocus this repo as **lokey-node-agent**:

- A single small binary: stream randomness from the LoKey box, inject it
  into the node's kernel pool via `RNDADDENTROPY` on `/dev/random`.
  The existing stream client and writer code is most of this already.
- Shipped as a **plain DaemonSet via a Helm chart**. No CRD, no operator:
  the agent's configuration is a URL and a couple of numbers — static
  values, nothing to reconcile. Operators earn their complexity when there
  is ongoing lifecycle logic; there is none here.
- Security context: one container with `CAP_SYS_ADMIN` (required for the
  ioctl) and a hostPath mount of `/dev/random` only — instead of today's
  `privileged` + `hostPID` + all of `/dev`.
- **The stream must be authenticated** (TLS with server verification, or an
  HMAC over chunks with a shared key). If we credit entropy into the kernel
  pool, an attacker who can tamper with the stream feeds us predictable
  bytes. Plain HTTP is not acceptable for this component.

Deleted: the operator, the CRD, sidecar injection, the mknod init container
and its seccomp profile. The client-container seccomp profile stays (adapted
for the ioctl).

Consumption paths after this change:

| Consumer | Path |
|---|---|
| Apps you control | LoKey REST API directly |
| Everything else | node agent → kernel pool → `getrandom()` |

## What "done" looks like (testable without reading the code)

- Deploy the Helm chart on a test cluster; agent pods run on every node.
- `cat /proc/sys/kernel/random/entropy_avail` on a node shows the pool
  being fed; agent logs show bytes credited.
- Kill the LoKey box: agent retries with backoff, node keeps working
  (kernel randomness never depends on the agent being up).
- Tamper with the stream (wrong cert / wrong HMAC key): agent refuses the
  data and says so in the logs.
