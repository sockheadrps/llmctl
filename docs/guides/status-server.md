# Status Server

The status server is llmctl's lightweight runtime surface for browser and JSON access. It exposes the current running instances, health state, VRAM usage, tok/s, RPC client snapshots, and a small bounded history for charting.

---

## What It Exposes

The server serves:

- `GET /status` for the current snapshot
- `GET /history` for recent snapshots
- `GET /dashboard` for the browser view when dashboard serving is enabled

`/status` remains the source of truth for the live state. `/history` is what the dashboard uses to draw charts.

When history persistence is enabled, llmctl stores those samples in `~/.llmctl/status_history.json` and restores them on restart.

The dashboard also includes model and profile filters, a time-range control, and health-change coloring so you can zoom into one run or compare several at once.

The browser dashboard is off by default for new configs. You can enable or disable it from the Status Server settings while keeping `/status` and `/history` available for JSON and remote-client use.

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

The browser dashboard is the built-in monitoring view for this same server.

In RPC mode, a client polls the server's status address, discovers the RPC endpoint, and shows the remote GPU telemetry in its own Overview tab.

---

## How To Read It In The TUI

The Overview tab is the human-friendly view of the status server snapshot:

- Local running instances appear under `Local`
- Remote client snapshots appear under `Remote`
- GPU 0 shows local telemetry
- GPU 1 shows remote telemetry when RPC is connected

That means you usually do not need to hit `/status` by hand unless you're integrating llmctl with another tool.

If you want a browser view, open the same host and port at `/dashboard` while dashboard serving is enabled. The page reads `/status` and `/history` from the same server.

---

## Example

```bash
curl http://127.0.0.1:11435/status
```

Use the actual host and port shown in your llmctl settings or RPC server tab.
