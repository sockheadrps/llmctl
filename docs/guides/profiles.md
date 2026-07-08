# Profiles

A profile is a saved launch configuration for a model. It defines exactly how `llama-server` will run: which port, how many GPU layers, how much context, and what sampling behavior to use.

One model can have as many profiles as you want. Each runs as a separate instance on its own port.

---

## Why Multiple Profiles?

A single GGUF can serve very different purposes depending on how it's configured. For example, the same Phi-4 model file might have:

- **FullGPU** — all layers on the GPU, large context, for general chat at full quality
- **LowMemory** — fewer GPU layers, smaller context, when VRAM is shared with other tasks
- **IDEAutoComplete** — CPU-only, tiny context, dedicated port for editor integration

Switching between use cases is just navigating to the right profile row and pressing `Enter`.

---

## Creating a Profile

1. Go to the **Models** tab and expand a model (`Enter` or `→`)
2. Navigate to **+ New Profile** and press `Enter`
3. Fill in the form and press `Enter` to save

The only required field is **Port**. Every other field has a default.

![New profile form with port, GPU layers, and context size fields visible](../assets/screenshots/new-profile-form.png)


---

## Copying a Profile

To duplicate an existing profile (useful for making a variant with one change):

1. Navigate to the profile row
2. Press `c`

A copy is created with the same settings. Rename and adjust as needed. You can also press `x` to open a model to copy the cli parameters to clipboard, and in model profile edits, import by pasting those cli flags directly into llmctl.

---

## Key Settings Explained

### Port

The TCP port `llama-server` listens on for OpenAI-compatible API requests. Every running instance must have a unique port — if two profiles share a port, the second one will fail to start.

Pick any free port above 1024. Common choices: `8080`, `8081`, `11434`.

### GPU Layers

How many transformer layers to load onto the GPU. More layers = faster inference but more VRAM.

| Value | Effect |
|---|---|
| `99` | Load everything onto the GPU (recommended if VRAM allows) |
| `0` | CPU-only — no GPU used at all |
| `20`–`40` | Partial offload — splits between GPU and RAM |

If you set this too high and run out of VRAM, `llama-server` will fail to start. Lower the value until it loads cleanly.

### Context Size

How many tokens the model can hold in its context window at once — roughly, how much conversation history it can "see." Larger context = more RAM/VRAM.

A good default is `4096`. Use `8192` or higher if you need to pass long documents or maintain long conversations. Use `1024` or `2048` if you're memory-constrained or using a profile for autocomplete where long context isn't needed.

### Flash Attention

When enabled, uses a more memory-efficient attention implementation. Faster on most hardware and uses less VRAM for large contexts — but not supported by all model architectures.

Enable it if your model supports it. If the model fails to start with it on, turn it off.

### CPU Only

Forces the model to run entirely in system RAM, bypassing the GPU. Useful when:
- You don't have a GPU
- Your GPU is fully occupied by other profiles
- You want a dedicated low-priority instance for background tasks

CPU inference is significantly slower than GPU but doesn't compete for VRAM. When a model is running CPU-only, its RAM usage appears in the System Telemetry box on the Overview tab.

### MLock

Pins the model's memory pages so the OS can't swap them to disk. Prevents latency spikes caused by page faults when the model hasn't been used in a while.

Recommended if you have enough RAM and want consistent response times. Leave it off if you're memory-constrained.

### Alias

A short display name shown in the Overview tab and Running tab instead of the full profile key. Useful when you have multiple profiles with similar names and want to tell them apart at a glance.

If no alias is set, the profile key is used.

---

## Editing a Profile

Navigate to the profile row and press `Enter` to open it. If the profile is currently running, stop it first — you can't edit a live instance.

Change any fields and press `Enter` to save.

---

## Deleting a Profile

Navigate to the profile row and press `Del`. Press `Del` again to confirm.

Deleting a profile doesn't affect the model or any other profiles. Running instances must be stopped before deletion.

---

## What Happens if Two Profiles Share a Port

The second profile to start will fail immediately — `llama-server` can't bind a port that's already in use. The health dot turns red and the log viewer (`e`) will show a "bind: address already in use" error.

Each profile needs its own port. If you're getting this error, check the **Running** tab to see what's already listening.

---

## Viewing Profile Settings

Select a profile row in the **Models** tab tree — the right pane shows the full configuration: port, GPU layers, context size, flash attention state, cache settings, and any extra args passed to `llama-server`.

![Expanded Models tree showing profile rows and the right-hand details pane](../assets/screenshots/concepts-model-profiles-tree.png)
