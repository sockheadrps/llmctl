# Profile Options

Every field available when creating or editing a profile. Fields are grouped by category.

All fields except **Port** are optional — the defaults work for most use cases.

---

## Core

| Field | Type | Default | Description |
|---|---|---|---|
| **Port** | integer | (suggested free port) | TCP port `llama-server` listens on. Must be unique per running instance. |
| **Alias** | string | — | Short display name shown in the Overview and Running tabs. Falls back to the profile key if not set. |
| **Host** | string | `127.0.0.1` | Network interface to bind. Use `0.0.0.0` to accept connections from other machines on the network. |
| **Notes** | string | — | Free-text notes visible only in the profile details pane. Not used at runtime. |

---

## Runtime

| Field | Type | Default | Description |
|---|---|---|---|
| **GPU Layers** | integer | `999` | Number of transformer layers loaded onto the GPU. `999` loads the entire model if VRAM allows. Set to `0` for CPU-only. Reduce if you run out of VRAM. |
| **CPU Only** | boolean | `false` | Force all inference to run on CPU, regardless of GPU Layers. No VRAM used. Slower but doesn't compete for GPU memory. |
| **Flash Attention** | boolean | `true` | Use Flash Attention for faster inference with lower memory usage on large contexts. Not supported by all model architectures — disable if the model fails to load. |
| **MLock** | boolean | `false` | Pin model memory pages so the OS can't swap them to disk. Prevents latency spikes on idle models. Requires enough free RAM to hold the full model. |
| **MMap** | boolean | (llama.cpp default) | Use memory-mapped file loading. Speeds up cold loads and reduces RAM duplication when multiple processes load the same file. |
| **Parallel** | integer | (llama.cpp default) | Number of simultaneous inference slots (concurrent request handling). Increases VRAM usage. |
| **Continuous Batching** | boolean | (llama.cpp default) | Process multiple requests in overlapping batches. Improves throughput under load. |
| **RPC Enabled** | boolean | (global setting) | Per-profile override for RPC offloading. Leave unset to follow the global RPC setting in llmctl Settings. |

---

## Context & Cache

| Field | Type | Default | Description |
|---|---|---|---|
| **Context Size** | integer | `8192` | Maximum token context window — how much text the model can hold at once. Larger values use more VRAM/RAM. |
| **Batch Size** | integer | (llama.cpp default) | Logical batch size for prompt processing. Larger values process prompts faster but use more memory. |
| **Micro-batch Size** | integer | (llama.cpp default) | Physical micro-batch size for compute. Usually set equal to or smaller than Batch Size. |
| **KV Offload** | boolean | (llama.cpp default) | Offload KV cache operations to the GPU. Reduces CPU overhead for large contexts. |
| **Cache Prompt** | boolean | (llama.cpp default) | Cache prompt processing to speed up repeated or shared prefixes. Useful for system prompts reused across requests. |
| **Cache RAM** | integer (MiB) | (llama.cpp default) | Maximum RAM (in MiB) to use for prompt caching. |
| **Cache Type K** | string | (llama.cpp default) | Data type for KV cache keys. Options: `f16`, `q8_0`, `q4_0`. Lower precision reduces VRAM at some quality cost. |
| **Cache Type V** | string | (llama.cpp default) | Data type for KV cache values. Same options as Cache Type K. |

---

## Sampling

These control how the model selects tokens during generation. The defaults work well for most conversational use. Adjust them if you need more deterministic, creative, or constrained output.

| Field | Type | Default | Description |
|---|---|---|---|
| **Temperature** | float | `0.6` | Controls randomness. Lower = more focused and deterministic. Higher = more varied and creative. `0.0` makes output fully deterministic. |
| **Top P** | float | `0.95` | Nucleus sampling. Only tokens whose cumulative probability reaches this threshold are considered. Lower values narrow the candidate pool. |
| **Top K** | integer | `20` | Limit token selection to the top K most probable tokens at each step. `0` disables. |
| **Min P** | float | `0.0` | Minimum token probability filter. Tokens below this fraction of the top token's probability are excluded. |
| **Presence Penalty** | float | — | Penalizes tokens that have already appeared anywhere in the output. Encourages introducing new topics. |
| **Frequency Penalty** | float | — | Penalizes tokens proportional to how often they've appeared. Discourages repetitive phrasing. |
| **Repetition Penalty** | float | — | Penalty applied to recently generated tokens (within Repeat Last N). Values above `1.0` reduce repetition. |
| **Repeat Last N** | integer | — | Number of most recent tokens to consider for the repetition penalty. |
| **Seed** | integer | — | Random seed for reproducible output. Same seed + same prompt produces the same output. Leave unset for random behavior. |

---

## Reasoning

For models that support extended reasoning (chain-of-thought). Leave these unset for standard models — they have no effect.

| Field | Type | Default | Description |
|---|---|---|---|
| **Reasoning** | string | — | Enable or control reasoning mode. Values: `on`, `off`, `auto`. |
| **Reasoning Budget** | integer | — | Token budget allocated for internal reasoning. Higher budgets allow more thorough reasoning at the cost of latency. |
| **Reasoning Format** | string | — | How reasoning content is returned in the response. Depends on the model and llama.cpp version. |

---

## Advanced

| Field | Type | Default | Description |
|---|---|---|---|
| **Extra Args** | string list | — | Raw CLI arguments passed directly to `llama-server`. Use this for experimental or llama.cpp-specific flags not exposed by the profile form. Example: `--no-mmap --threads 8`. |

---

## Read-Only Fields

These are tracked automatically and not user-editable.

| Field | Description |
|---|---|
| **Peak tok/s** | All-time highest tokens-per-second observed for this profile. Displayed in the Overview tab speed line. Persists across restarts. |
