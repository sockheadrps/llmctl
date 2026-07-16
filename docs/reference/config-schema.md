# Config Schema

This page is intentionally brief. llmctl is designed to be configured through the TUI, not hand-edited YAML.

If you do need to inspect the file, the important top-level areas are:

- `llama_server_bin` - path to the `llama-server` binary
- `models_dirs` - directories scanned for GGUF files
- `models` - imported models and their profiles
- RPC settings - server/client mode, remote status address, and bind settings
- Status server settings - enable flag, bind settings, dashboard toggle, and history persistence toggle
- Network settings - saved internet and RPC interface names on Linux

The key split is:

- `rpc_enabled` and `rpc_mode` control distributed GPU transport
- `status_server_enabled` controls the local JSON/dashboard server
- `status_history_persist` controls history persistence
- `status_dashboard_enabled` controls whether `/dashboard` is served

---

## Typical Shape

```yaml
llama_server_bin: C:\llama\llama-server.exe
models_dirs:
  - D:\models
models:
  my-model:
    name: my-model
    path: D:\models\my-model.gguf
    profiles:
      default:
        port: 8080
        ctx_size: 8192
        gpu_layers: 99
status_history_persist: true
status_dashboard_enabled: false
```

Notes:

- `status_server_enabled` is independent from `rpc_enabled`
- `status_history_persist` defaults to `true`
- `status_dashboard_enabled` defaults to `false`
- The dashboard is read-only monitoring only

---

## When To Edit By Hand

- Restoring a config after a bad TUI change
- Scripting a machine setup
- Moving the config to another host

For ordinary use, the TUI is the safer way to make changes.
