# llmctl browser dashboard checklist

## Goal

Build a browser-style dashboard for server-enabled `llmctl` processes by extending the existing HTTP status surface with history and a simple web UI.

## Current recommendation

- [x] Add a bounded history layer and a `/history` endpoint.
- [x] Add a lightweight `/dashboard` page that reads `/status` and `/history`.
- [x] Keep `/status` as the live snapshot source of truth.

## Phase 1: Data capture

- [x] Decide what gets sampled.
- [x] Define the history JSON shape.
- [x] Add a bounded in-memory store.
- [x] Write tests around sample retention.
- [x] Write tests around serialization and schema stability.

## Phase 2: API surface

- [x] Expose `/history`.
- [x] Keep `/status` unchanged.
- [x] Decide whether `/history` should be per-process, per-host, or both. (Per status-server instance.)
- [x] Add basic error handling. (The dashboard surfaces fetch failures cleanly.)
- [x] Decide whether CORS or origin restrictions are needed. (Same-origin dashboard; no extra CORS layer needed.)

## Phase 3: Browser dashboard

- [x] Add `/dashboard`.
- [x] Render the current state.
- [x] Draw charts from history.
- [x] Show health transitions clearly.
- [x] Make the layout work on desktop.
- [x] Make the layout work on smaller windows.
- [x] Show connected RPC client snapshots.
- [x] Add an all-client summary row.
- [x] Make the main model chart source-selectable across local and connected clients.
- [x] Add a setting to disable the browser dashboard while keeping `/status` and `/history`.
- [x] Default the dashboard to off for new configs.

## Phase 4: Polish

- [x] Add filters by model.
- [x] Add filters by profile.
- [x] Add time-range controls.
- [x] Add color coding for health changes.
- [x] Add a persistence toggle for history.

## Open questions

- [x] Should history persist across restarts? Yes, with a toggle.
- [x] Should the browser view stay read-only? Yes, keep it read-only.

## Risks

- [ ] Extra HTTP exposure may reveal workload details that some users do not want shared.
- [ ] A charting UI can easily grow into a mini web app if we are not strict about scope.
- [ ] High-frequency sampling could add noise or overhead if we try to track too much.

## First milestone

- [x] Show current running models.
- [x] Show health.
- [x] Show tok/s trend.
- [x] Show VRAM trend.
- [x] Confirm the page feels useful before expanding scope.
