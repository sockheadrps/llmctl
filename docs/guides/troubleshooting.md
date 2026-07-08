# Troubleshooting

Short symptom-to-fix notes for the most common llmctl issues.

---

## `llama-server` not found

Likely cause:

- The binary is not on `PATH`
- Or the path in Settings is wrong

Fix:

- Point the Settings tab at the correct `llama-server` executable
- Or install the llama.cpp release you meant to use

---

## Model stays yellow or turns red

Likely cause:

- The model is still loading
- VRAM is too low
- The model file is missing or invalid
- The port is already in use

Fix:

- Open logs with `e`
- Lower GPU layers
- Pick a different port
- Re-check the model path and binary path

---

## No VRAM bar

Likely cause:

- `nvidia-smi` is missing
- The machine has no NVIDIA GPU
- The model is CPU-only

Fix:

- Install NVIDIA drivers if needed
- Or ignore it if you intended to run CPU-only

---

## RPC client is not connecting

Likely cause:

- The client is not pointing at the server's status address
- The firewall blocks the status port
- The client and server use mismatched llama.cpp releases

Fix:

- Re-copy the server address from the RPC Server tab
- Verify the server is online
- Make sure both machines use the same llama.cpp release

---

## GPU 1 is missing

Likely cause:

- RPC is not connected yet
- The client has not discovered the remote endpoint

Fix:

- Check the RPC Server tab on both machines
- Confirm the client is polling the correct status server address

---

## Network tab is unavailable

Likely cause:

- You are not on Linux
- Or `nmcli` is missing

Fix:

- Install NetworkManager tools on Linux
- Or use the RPC settings directly instead of the network tab

---

## Config parse error on launch

Likely cause:

- The YAML file was edited manually and no longer parses

Fix:

- Revert the bad edit
- Or use the TUI to repair the config

---

## What To Check First

1. Open logs with `e`.
2. Confirm the model and profile still exist.
3. Confirm the `llama-server` path and port.
4. Confirm the relevant host or RPC address.
