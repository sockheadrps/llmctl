# dev branch notes

## Overview

`dev` now combines two related sets of changes:

- the dashboard and status-server work
- the profile form and model-edit UX improvements merged in from `main`

This note is meant to be a docs source-of-truth reference, not a task tracker.

## Profile Editing

### What changed

- The profile form has a clearer layer-splitting control.
- The layer split slider is easier to reach with keyboard navigation.
- Imported CLI args clear stale fields before applying parsed values.
- The helper text for the split control now explains local vs remote GPU layers more clearly.

### Good doc angles later

- How to edit a profile in the TUI
- How layer splitting works when RPC is enabled
- What happens when pasting CLI args into an existing form

## Status Server API

### What changed

- `GET /status` remains the live JSON snapshot.
- `GET /history` returns bounded history samples for charting.
- `GET /dashboard` serves the browser UI when dashboard serving is enabled.
- `GET /` redirects to `/dashboard` when enabled, otherwise to `/status`.
- Status snapshots now include per-GPU device slices alongside aggregate VRAM totals.
- Running-model entries now include per-GPU VRAM load slices when the runtime can see them.
- RPC server snapshots now publish their own per-GPU slices so split RPC loads can be shown from the server side too.
- RPC server startup now reuses an already-running `ggml-rpc-server` that matches the configured binary, host, and port instead of spawning a duplicate.
- Model VRAM slices on Windows now prefer the startup log lines (`CUDA* model buffer size = ...`, `RPC* model buffer size = ...`) so split RPC loads can be shown from the model log instead of guessing from raw PID GPU usage.

### Behavior notes

- The browser dashboard is read-only.
- History can be persisted across restarts.
- Dashboard serving can be disabled independently from JSON status access.

### Good doc angles later

- Status server overview
- What each endpoint is for
- When to use `/status` versus `/history`

## Browser Dashboard

### What changed

- The dashboard focuses on a single source selector:
  - local
  - remote
  - all
- The dashboard shows runs from the selected source and connected RPC telemetry.
- The main control row no longer exposes model, profile, or time-range filters.
- The active-model cards now always show their trend blocks inline.
- The dashboard includes:
  - current running model cards
  - tok/s trends
  - per-model VRAM and RAM trends
  - health transition badges when a run changes state
- Active-model cards now render per-GPU VRAM load cards when device slices are available.
- RPC-loaded models keep the `RPC` badge and can show a GPU-by-GPU load breakdown instead of only a single aggregate chart.
- The per-GPU cards now represent model-slice load distribution, not raw GPU capacity utilization.
- Split RPC models now merge matching local and remote peers so the UI can show the full model load across all GPUs.
- The dashboard groups split models across all peers first, then filters the visible group by source so the combined GPU slices stay intact.
- The dashboard now uses an explicit `allSourceRuns` helper so grouped cards keep working while still merging local and remote peers.
- The TUI Overview screen now labels remote active services as `RPC` and shows their per-GPU model load slices in the active-services list.
- The TUI Overview screen now merges the server-side RPC GPU slices with the connected client slices so the split model load is visible on both sides.
- The source trends panel now shows:
  - a per-GPU VRAM utilization card with stacked horizontal bars
  - a combined VRAM trend chart for the selected source
  - a compact summary line with local and remote active counts
- The source trends panel no longer shows a separate source tok/s card or a separate RAM trend chart.
- RAM is shown as a used-only summary because the status payload does not expose a total.
- Switching the source filter now resets the trend history window so old chart peaks do not leak into the new source view.
- Remote RPC-loaded models now show an `RPC` badge and a model-specific GPU load card that uses the matching source GPU total.
- The dashboard no longer has a separate Remote Clients section.
- The top status chip reflects connected client activity instead of implying only local runs exist.
- Remote runs are labeled with their source so the selected view is easier to read.

### Good doc angles later

- Dashboard overview and screenshots
- How to interpret the source selector
- How the expandable per-model trends work
- How to read the trend panels
- How remote clients appear in the browser view

## Config And Persistence

### What changed

- History persistence is toggleable in settings.
- Dashboard serving is toggleable in settings.
- New configs default the dashboard toggle to off.
- Existing configs without the field keep working through the helper fallback.
- Status samples are stored in a bounded in-memory history.
- Persistent history is written to `~/.llmctl/status_history.json`.

### Good doc angles later

- Status server settings
- Which toggles affect the browser dashboard
- What persists across restarts

## Relevant Files

- [`internal/tui/form.go`](/C:/Users/rpski/code25/llmctl/internal/tui/form.go)
- [`internal/tui/form_view.go`](/C:/Users/rpski/code25/llmctl/internal/tui/form_view.go)
- [`internal/tui/form_parse.go`](/C:/Users/rpski/code25/llmctl/internal/tui/form_parse.go)
- [`internal/statusserver/server.go`](/C:/Users/rpski/code25/llmctl/internal/statusserver/server.go)
- [`internal/statusserver/dashboard.html`](/C:/Users/rpski/code25/llmctl/internal/statusserver/dashboard.html)
- [`internal/config/config.go`](/C:/Users/rpski/code25/llmctl/internal/config/config.go)
- [`docs/guides/status-server.md`](/C:/Users/rpski/code25/llmctl/docs/guides/status-server.md)

## Notes For Future Docs

- Keep the dashboard documentation read-only and focused on monitoring.
- Treat the status server as the shared runtime surface for JSON, remote client snapshots, and the browser dashboard.
- Use the dashboard source selector language consistently:
  - local
  - all connected clients
  - one connected client
