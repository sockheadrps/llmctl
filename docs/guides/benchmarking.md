# Benchmarking

llmctl does not have a separate benchmark runner. Instead, it gives you enough visibility to compare profiles in a repeatable way: load time, current tok/s, rolling average tok/s, peak tok/s, and VRAM or RAM usage are all visible in the TUI.

---

## What To Compare

Keep the prompt, context, and hardware the same, then vary one thing at a time:

- GPU layers
- Context size
- Flash attention on or off
- CPU-only vs GPU-backed
- Template choice
- Quantization / model file

That makes the comparison useful instead of just noisy.

---

## A Simple Benchmark Loop

1. Create two or more profiles for the same model.
2. Keep the prompt and sampling settings identical.
3. Start one profile at a time.
4. Watch the Overview tab while you run the same request.
5. Record the load time and the tok/s numbers you see in the dashboard.

The right place to compare results is the Overview tab:

- `Current` shows the live generation speed.
- `Avg` shows the rolling average for the current session.
- `Peak` shows the best speed llmctl has seen for that profile.

---

## Reading The Numbers

### Load time

Shown in the Running tab and in the running-instance row. Use it to compare startup cost across profiles.

### tok/s

Shown in the Overview and Running tabs. Use it for generation speed comparisons.

### VRAM and RAM

- GPU-backed profiles consume VRAM.
- CPU-only profiles consume RAM instead of VRAM.
- The Overview and Running tabs show this live so you can see when one profile is just too big for the machine.

---

## Tips For Repeatability

- Use the same prompt every time.
- Wait for the instance to reach `up` before measuring generation speed.
- Compare profiles one at a time instead of multiple active instances on the same GPU.
- If you want to compare cold-start behavior, restart the profile between runs.
- If you want steady-state throughput, let the model warm up and then compare the `Avg` and `Peak` values.

---

## When Benchmarking Helps Most

- Choosing between two quantizations of the same model
- Finding the largest practical context size
- Comparing CPU-only and GPU-backed profiles
- Checking whether a template is a good starting point
- Deciding whether to offload more layers to the GPU
