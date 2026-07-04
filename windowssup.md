# Windows Compatibility Implementation Plan

## Overview

The application is already largely cross-platform. Most of the work required to support Windows involves:

* Cross-platform path handling
* Process launching
* Process shutdown
* Environment variable expansion
* Minor TUI considerations

The overall application architecture does **not** need significant changes.

---

# Configuration

## Linux

```yaml
llama_server_bin: /home/nonsrs/llama.cpp/build/bin/llama-server

models_dirs:
  - ~/models
```

## Windows

```yaml
llama_server_bin: C:\llama.cpp\build\bin\Release\llama-server.exe

models_dirs:
  - C:\models

models:
  phi4-mini:
    name: Phi 4 Mini
    path: C:\models\Phi-4-mini-instruct.Q8_0.gguf
    profiles:
      default:
        port: 8080
        ctx_size: 8192
        gpu_layers: 999
```

---

# Cross-Platform Path Handling

Never assume Linux-style paths.

Instead, use Go's `filepath` package everywhere.

```go
filepath.Clean(path)
filepath.Abs(path)
filepath.Join(base, name)
os.UserHomeDir()
```

Never hardcode:

```text
/
```

Instead use:

```go
filepath.Join(...)
```

or

```go
filepath.Separator
```

This allows Windows (`\`) and Linux (`/`) to work automatically.

---

# Home Directory Expansion

Linux commonly uses:

```text
~/models
```

Windows users may also expect:

```text
~\models
```

Go does **not** automatically expand `~`.

Implement manual expansion using:

```go
os.UserHomeDir()
```

Example:

```
~/models
↓

/home/nonsrs/models
```

or

```
~\models
↓

C:\Users\Ryan\models
```

---

# Environment Variable Expansion

Support environment variables inside the config.

Example:

```yaml
llama_server_bin: ${LLAMA_SERVER_BIN}

models_dirs:
  - ${USERPROFILE}\models
```

On Linux:

```text
${HOME}
```

On Windows:

```text
${USERPROFILE}
```

Expand these automatically before loading paths.

Benefits:

* No hardcoded usernames
* Easier portable configs
* Better multi-machine support

---

# Launching llama-server

Use `exec.Command()`.

Do **not** build one large command string.

Correct:

```go
cmd := exec.Command(config.LlamaServerBin, args...)

cmd.Stdout = os.Stdout
cmd.Stderr = os.Stderr
cmd.Stdin = os.Stdin

err := cmd.Start()
```

This works correctly on both Linux and Windows.

---

# Arguments

Continue using argument slices.

Example:

```go
args := []string{
    "-m", model.Path,
    "--port", strconv.Itoa(profile.Port),
    "-c", strconv.Itoa(profile.CtxSize),
}
```

Advantages:

* Proper quoting
* Handles spaces in file paths
* No platform-specific escaping

Never manually concatenate:

```
llama-server -m "C:\Program Files\..."
```

Let `exec.Command()` perform the quoting.

---

# Binary Names

Linux:

```yaml
llama_server_bin: /home/nonsrs/llama.cpp/build/bin/llama-server
```

Windows:

```yaml
llama_server_bin: C:\llama.cpp\build\bin\Release\llama-server.exe
```

The binary path should remain configurable rather than being hardcoded.

---

# Process Shutdown

Linux commonly relies on signals like:

* SIGTERM
* SIGINT

Windows process handling differs.

For portability, stop the server using:

```go
cmd.Process.Kill()
```

A future enhancement could implement graceful, OS-specific shutdown behavior.

---

# Host Binding

Avoid embedding machine-specific IP addresses.

Instead of:

```yaml
extra_args:
  - --host
  - 192.168.1.138
```

Prefer defaults such as:

```yaml
extra_args:
  - --host
  - 127.0.0.1
```

or

```yaml
extra_args:
  - --host
  - 0.0.0.0
```

This makes configurations portable across different computers.

---

# TUI Compatibility

If the application uses Bubble Tea, Lip Gloss, or similar libraries, the TUI should work on:

* Windows Terminal
* PowerShell
* Linux terminals
* macOS terminals

Older `cmd.exe` consoles may render less cleanly but should still function.

No major platform-specific TUI changes should be necessary.

---

# Recommended Configuration Structure

Repository:

```
config.example.yml
```

Ignored locally:

```
config.yml
```

On first launch:

1. Check whether `config.yml` exists.
2. If not:

   * Generate a default configuration.
   * Or copy `config.example.yml`.
3. Launch normally.

This keeps personal machine paths out of version control while providing a working template for users.

---

# Summary

The project is already well suited for cross-platform support. Most required work centers on:

* Using `filepath` for all filesystem operations
* Expanding `~` and environment variables
* Keeping `llama_server_bin` configurable
* Launching processes via `exec.Command()`
* Using argument slices instead of command strings
* Handling process termination in a portable way
* Avoiding machine-specific IP addresses in default configurations

With these changes, the application should run on Linux and Windows with minimal platform-specific code while preserving a single, portable configuration format.
