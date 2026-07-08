# CLI Reference

llmctl is primarily a TUI application, but it also ships a small command-line surface for scripting and recovery.

---

## Global

| Command | Description |
|---|---|
| `llmctl` | Launch the interactive TUI |
| `llmctl tui` | Explicitly launch the TUI |
| `llmctl --config <path>` | Use a specific config file |
| `llmctl version` | Print the build version |

---

## Instance Commands

| Command | Description |
|---|---|
| `llmctl run <model> <profile>` | Start a profile as a detached `llama-server` process |
| `llmctl stop <model> <profile>` | Stop a running instance |
| `llmctl ps` | List running instances |
| `llmctl logs <model> <profile>` | Show the log file for a running instance |
| `llmctl logs -f <model> <profile>` | Tail the log file |
| `llmctl status` | Show the current running-instance snapshot |
| `llmctl status --json` | Print the same snapshot as JSON |

---

## Import / Export

| Command | Description |
|---|---|
| `llmctl export` | Print all configured models and profiles as YAML |
| `llmctl import <file>` | Import models and profiles from a YAML export |
| `llmctl import <file> --merge` | Overwrite matching models instead of skipping them |

---

## Network

| Command | Description |
|---|---|
| `llmctl network rpc` | Switch to the RPC network profile |
| `llmctl network internet` | Switch back to the internet network profile |
| `llmctl network status` | Show routing and link speed |
| `llmctl network --internet-conn <name>` | Override the saved internet connection name |
| `llmctl network --rpc-conn <name>` | Override the saved RPC connection name |
| `llmctl network --iface <name>` | Override the interface used by `status` |

---

## Notes

- `llmctl` prints friendly text output by default.
- Use `--json` only where the command supports it.
- If you just want to use the app normally, start with the TUI.
