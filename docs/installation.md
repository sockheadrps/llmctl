# Installation

## Prerequisites

llmctl manages `llama-server` processes — it doesn't bundle its own inference engine. You need the llama.cpp binaries before llmctl can do anything useful.

For the full dependency rundown, see the [README Requirements section](https://github.com/sockheadrps/llmctl#requirements). The list below is the same set of system dependencies, organized for installation.

### System dependencies

### Required

**llama-server** (from [llama.cpp](https://github.com/ggerganov/llama.cpp))

The inference engine llmctl wraps. Every model you run goes through this binary.

- Download a pre-built release from the [llama.cpp releases page](https://github.com/ggerganov/llama.cpp/releases) — pick the build that matches your hardware (CUDA, Metal, CPU-only, etc.)
- Or build from source if you need a custom configuration
- The binary must either be on your `PATH` or its location set in llmctl's Settings tab after first launch

### Required for distributed GPU (RPC mode)

**ggml-rpc-server** (also from llama.cpp, same release)

Only needed on machines that will act as an RPC server — contributing their GPU to another machine running the model. If you're only running models locally, you don't need this.

### Optional but recommended

**nvidia-smi** (comes with NVIDIA GPU drivers)

Used to read VRAM usage per process. Without it, the VRAM bar in the Running tab and GPU telemetry in the Overview tab won't appear. Everything else works fine.

### Required for clipboard features on Linux

llmctl can copy model addresses and endpoints to your clipboard. On Linux this needs one of:

| Display server | Package |
|---|---|
| X11 | `xclip` or `xsel` |
| Wayland | `wl-clipboard` |

Not needed on Windows (uses built-in `clip.exe`) or when connecting over SSH (uses OSC52 terminal escape sequences, which most modern terminals support).

### Required for networking features on Linux

The Network tab and `llmctl network` subcommands depend on standard Linux networking tools:

- `nmcli` from NetworkManager - used to bring connection profiles up and down
- `ethtool` - used by `llmctl network status` to report link speed
- `polkit` authorization for NetworkManager - needed if your user cannot already manage connections

---

## Installing llmctl

### Download a release binary

Go to the [llmctl releases page](https://github.com/sockheadrps/llmctl/releases) and download the binary for your platform.

**Linux / macOS**
```bash
chmod +x llmctl
sudo mv llmctl /usr/local/bin/
```

**Windows**

Move `llmctl.exe` somewhere on your `PATH`, or run it directly from a directory.

### Build from source

Requires Go 1.21 or later.

```bash
git clone https://github.com/sockheadrps/llmctl.git
cd llmctl
go build -o llmctl .
```

---

## First launch

Run `llmctl` in your terminal. On first launch you'll see the Overview tab with empty Active Services — nothing is running yet.

![llmctl on first launch — Overview tab, empty state](/assets/screenshots/first-launch.png)

The first thing to do is point llmctl at your `llama-server` binary:

1. Press `d` to reach the **Settings** tab
2. Under **llama-server binary**, enter the full path to your `llama-server` executable
3. Under **Model directories**, add one or more folders where your GGUF files live

From there, head to the [Quickstart](./quickstart) to import your first model and get it running.

