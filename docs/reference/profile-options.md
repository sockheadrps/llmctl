# Profile Options

Every field available when creating or editing a profile. Fields are grouped by category.

All fields except **Port** are optional - the defaults work for most use cases.

Fields that do not map to a single llama-server flag use `—` in the Flag column.

---

## Core

| Field | Flag | Type | Default | Description |
|---|---|---|---|---|
| **Port** | `--port` | integer | (suggested free port) | TCP port `llama-server` listens on. Must be unique per running instance. |
| **Alias** | `--alias` | string | — | Short display name shown in the Overview and Running tabs. Falls back to the profile key if not set. |
| **Host** | `--host` | string | `127.0.0.1` | Network interface to bind. Use `0.0.0.0` to accept connections from other machines on the network. |
| **Notes** | `—` | string | — | Free-text notes visible only in the profile details pane. Not used at runtime. |

---

## Runtime

| Field | Flag | Type | Default | Description |
|---|---|---|---|---|
| **GPU Layers** | `--n-gpu-layers` | integer | `999` | Number of transformer layers loaded onto the GPU. `999` loads the entire model if VRAM allows. Set to `0` for CPU-only. Reduce if you run out of VRAM. |
| **CPU Only** |  | boolean | `false` | Convenience toggle that automatically sets `--n-gpu-layers 0` so all inference runs on CPU. No VRAM used. Slower but does not compete for GPU memory. |
| **Flash Attention** | `--flash-attn` | boolean | `true` | Use Flash Attention for faster inference with lower memory usage on large contexts. Not supported by all model architectures - disable if the model fails to load. |
| **MLock** | `--mlock` | boolean | `false` | Pin model memory pages so the OS cannot swap them to disk. Prevents latency spikes on idle models. Requires enough free RAM to hold the full model. |
| **MMap** | `--mmap` / `--no-mmap` | boolean | (llama.cpp default) | Use memory-mapped file loading. Speeds up cold loads and reduces RAM duplication when multiple processes load the same file. |
| **Parallel** | `--parallel` | integer | (llama.cpp default) | Number of simultaneous inference slots (concurrent request handling). Increases VRAM usage. |
| **Continuous Batching** | `--cont-batching` / `--no-cont-batching` | boolean | (llama.cpp default) | Process multiple requests in overlapping batches. Improves throughput under load. |
| **RPC Enabled** |  | boolean | (global setting) | Convenience toggle for per-profile RPC behavior. Use it to force RPC off for this profile even when RPC is enabled in the client, or leave it unset to follow the global setting in llmctl Settings. |

---

## Context & Cache

| Field | Flag | Type | Default | Description |
|---|---|---|---|---|
| **Context Size** | `--ctx-size` | integer | `8192` | Maximum token context window - how much text the model can hold at once. Larger values use more VRAM/RAM. |
| **Batch Size** | `--batch-size` | integer | (llama.cpp default) | Logical batch size for prompt processing. Larger values process prompts faster but use more memory. |
| **Micro-batch Size** | `--ubatch-size` | integer | (llama.cpp default) | Physical micro-batch size for compute. Usually set equal to or smaller than Batch Size. |
| **KV Offload** | `--kv-offload` / `--no-kv-offload` | boolean | (llama.cpp default) | Offload KV cache operations to the GPU. Reduces CPU overhead for large contexts. |
| **Cache Prompt** | `--cache-prompt` / `--no-cache-prompt` | boolean | (llama.cpp default) | Cache prompt processing to speed up repeated or shared prefixes. Useful for system prompts reused across requests. |
| **Cache RAM** | `--cache-ram` | integer (MiB) | (llama.cpp default) | Maximum RAM (in MiB) to use for prompt caching. |
| **Cache Type K** | `--cache-type-k` | string | (llama.cpp default) | Data type for KV cache keys. Options: `f16`, `q8_0`, `q4_0`. Lower precision reduces VRAM at some quality cost. |
| **Cache Type V** | `--cache-type-v` | string | (llama.cpp default) | Data type for KV cache values. Same options as Cache Type K. |

---

## Sampling

These control how the model selects tokens during generation. The defaults work well for most conversational use. Adjust them if you need more deterministic, creative, or constrained output.

| Field | Flag | Type | Default | Description |
|---|---|---|---|---|
| **Temperature** | `--temp` | float | `0.6` | Controls randomness. Lower = more focused and deterministic. Higher = more varied and creative. `0.0` makes output fully deterministic. |
| **Top P** | `--top-p` | float | `0.95` | Nucleus sampling. Only tokens whose cumulative probability reaches this threshold are considered. Lower values narrow the candidate pool. |
| **Top K** | `--top-k` | integer | `20` | Limit token selection to the top K most probable tokens at each step. `0` disables. |
| **Min P** | `--min-p` | float | `0.0` | Minimum token probability filter. Tokens below this fraction of the top token's probability are excluded. |
| **Presence Penalty** | `--presence-penalty` | float | — | Penalizes tokens that have already appeared anywhere in the output. Encourages introducing new topics. |
| **Frequency Penalty** | `--frequency-penalty` | float | — | Penalizes tokens proportional to how often they've appeared. Discourages repetitive phrasing. |
| **Repetition Penalty** | `--repeat-penalty` | float | — | Penalty applied to recently generated tokens (within Repeat Last N). Values above `1.0` reduce repetition. |
| **Repeat Last N** | `--repeat-last-n` | integer | — | Number of most recent tokens to consider for the repetition penalty. |
| **Seed** | `--seed` | integer | — | Random seed for reproducible output. Same seed + same prompt produces the same output. Leave unset for random behavior. |

---

## Reasoning

For models that support extended reasoning (chain-of-thought). Leave these unset for standard models - they have no effect.

| Field | Flag | Type | Default | Description |
|---|---|---|---|---|
| **Reasoning** | `--reasoning` | string | — | Enable or control reasoning mode. Values: `on`, `off`, `auto`. |
| **Reasoning Budget** | `--reasoning-budget` | integer | — | Token budget allocated for internal reasoning. Higher budgets allow more thorough reasoning at the cost of latency. |
| **Reasoning Format** | `--reasoning-format` | string | — | How reasoning content is returned in the response. Depends on the model and llama.cpp version. |

---

## Advanced

| Field | Flag | Type | Default | Description |
|---|---|---|---|---|
| **Extra Args** | custom | string list | — | Raw CLI arguments passed directly to `llama-server`. Use this for experimental or llama.cpp-specific flags not exposed by the profile form. Example: `--no-mmap --threads 8`. |

---

## Read-Only Fields

These are tracked automatically and are not user-editable.

| Field | Flag | Description |
|---|---|---|
| **Peak tok/s** | `—` | All-time highest tokens-per-second observed for this profile. Displayed in the Overview tab speed line. Persists across restarts. |
