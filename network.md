# llmctl network

`llmctl network` adds a small CLI for switching and inspecting the local machine's network setup. The TUI (`llmctl tui`) also has a Network tab that shows live link status and lets you switch profiles interactively.

## Commands

- `llmctl network rpc`
  - Brings down the configured internet connection.
  - Brings up the configured RPC connection.

- `llmctl network internet`
  - Brings down the configured RPC connection.
  - Brings up the configured internet connection.

- `llmctl network status`
  - Prints `ip route`.
  - Prints the current link speed for the configured interface using `ethtool`.

## TUI Network Tab

The `Network` tab in `llmctl tui` provides:

- Live status panel refreshed every 2 seconds: active connection name, link state (UP/DOWN), speed, carrier, and interface name.
- Two action rows — `→ RPC` and `→ Internet` — navigated with arrow keys.
- Press `enter` on an action row to open a confirm modal; confirm to switch, `esc` to cancel.
- The right-hand Details pane shows full status fields and a description of the selected action.
- A "switching network…" indicator appears in the status bar while the switch is in flight.

## Defaults

- Internet connection: `Wired connection 1`
- RPC connection: `enp8s0`
- Status interface: `enp8s0`

## Flags

These flags are available on `llmctl network` and on `llmctl tui`:

- `--internet-conn` — nmcli connection name for internet access
- `--rpc-conn` — nmcli connection name for the RPC link
- `--iface` — network interface for status checks

## Behavior

- `nmcli` is used for profile switching and active-connection detection.
- `ip link show` is used for link state (UP/DOWN).
- `ethtool` is used for speed and carrier inspection.
- Missing system tools are reported as errors.
- `nmcli connection down` errors are silently ignored (the profile may already be inactive); `connection up` errors are surfaced.

## Ideas / Backlog

### Robustness

- [ ] Surface `nmcli connection down` errors instead of silently discarding them (CLI path — TUI already surfaces `up` errors)
- [ ] Verify the switch actually succeeded — nmcli can exit 0 while the link stays down
- [ ] Guard against switching to the already-active profile (redundant flap)

### Auto-detection

- [ ] `llmctl network detect` — print a human summary of the active profile (name, interface, speed, link state) — useful as a quick CLI health check
- [ ] Auto-guess defaults on first run — scan `nmcli connection show` for likely candidates, prompt to confirm

### Health check

- [ ] `llmctl network check` — poll `ip link show` until the interface reaches `state UP`, optionally ping a known RPC host, exit non-zero on timeout
- [ ] `--wait` flag on `rpc` / `internet` — block until the interface is up (or timeout), then print final status; makes scripting reliable
- [ ] `--json` flag on `status` — structured output for scripting without grepping ethtool

### Won't do

- ~~`llmctl network configure` — per-machine config file~~ — per-invocation flags are preferred
