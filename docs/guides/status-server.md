# Status Server

The status server is llmctl's lightweight JSON endpoint for runtime visibility. It exposes the current running instances, health state, VRAM usage, tok/s, and RPC client snapshots.

---

## What It Exposes

The server serves a `GET /status` endpoint that returns JSON with the current snapshot.

You can also ask the CLI for the local instance list:

```bash
llmctl status
llmctl status --json
```

The plain-text form is handy for a terminal check. The JSON form is better for scripts.

---

## Why It Exists

The status server is mainly used for two things:

- RPC client discovery
- External monitoring or automation

In RPC mode, a client polls the server's status address, discovers the RPC endpoint, and shows the remote GPU telemetry in its own Overview tab.

---

## How To Read It In The TUI

The Overview tab is the human-friendly view of the status server snapshot:

- Local running instances appear under `Local`
- Remote client snapshots appear under `Remote`
- GPU 0 shows local telemetry
- GPU 1 shows remote telemetry when RPC is connected

That means you usually do not need to hit `/status` by hand unless you're integrating llmctl with another tool.

---

## Example

```bash
curl http://127.0.0.1:11435/status
```

Use the actual host and port shown in your llmctl settings or RPC server tab.
