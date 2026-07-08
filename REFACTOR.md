# llmctl Refactor Plan — `dev-refactor` branch

> **How to use this doc**: Each phase is a self-contained chunk of work.
> After every phase, the build and all tests must pass before moving on.
> Check boxes off as tasks complete. Each phase ends with a **Milestone
> Verification** block — run those commands before marking the phase done.
>
> Branch: `dev-refactor` (based on `dev`)
> Context note: if this conversation compacts, re-read this file first.

---

## Guiding Principles

1. **Same package, smaller files** — all split files stay in `package tui`.
   No subpackages in Phase 1–5 (avoids export churn and circular deps).
2. **One file split per commit** — each file split is its own commit so
   `git bisect` stays useful.
3. **No logic changes** — pure file moves during splits. Logic/cleanup is
   Phase 6+.
4. **Green on every commit** — `go build ./... && go test ./...` must pass
   after every single commit.

---

## Current File Sizes (baseline, as of branch cut)

| File | Lines |
|------|-------|
| `internal/tui/view.go` | 1,825 |
| `internal/tui/form.go` | 1,221 |
| `internal/tui/update.go` | 1,115 |
| `internal/tui/model.go` | 922 |
| `internal/tui/settings.go` | 678 |
| `internal/tui/network.go` | 495 |
| `internal/tui/settings_view.go` | 371 |
| `internal/tui/form_view.go` | 317 |

---

## Phase 1 — Split `view.go` (1,825 → ~5 files)

**Goal:** Break the monolithic view file into focused rendering files,
each responsible for one screen section. All stay in `package tui`.

### Target file map

| New file | Functions to move |
|----------|-------------------|
| `view_layout.go` | `paneDimensions`, `computeSplitHeights`, `splitPaneHeight`, `formRowTextWidth`, `mainDetailsGeometry`, `mainDetailsLineCount`, `mainDetailsVisibleLines` |
| `view_models.go` | `renderModelsTree`, `renderRecentsList`, `renderSettingsList`, `renderTabBarLabels`, `renderHeaderLine` |
| `view_running.go` | `renderRunning`, `renderRunningRow`, `renderRunningRowWithWidth`, `renderRunningOutputPane`, `renderRunningOutputColumn` |
| `view_details.go` | `renderDetails`, `renderDetailsWindow`, `renderModelPreview`, `renderVRAMHeader`, `detailPair` type, all `renderDetail*` helpers |
| `view_rpc.go` | `renderRPCServerTab`, `renderRPCServerModeTab`, `renderRPCConnectionTab`, `renderRPCServerOutputPane`, `renderRPCConnectionOutputPane`, `renderClientStatusLines`, `clientModelSizeMeta`, `rpcServerLoadedVRAMMiB`, `tailFittingHeightRPC`, `isNoisyRPCLine`, `compressRPCLogLines` |
| `view.go` (remaining) | `View`, `viewMain`, `helpText`, `renderLeftPaneContent`, `firstLine`, `truncateText`, `tailFittingHeight`, `fmtLoadDur` |

### Tasks

- [x] **1a** Move layout helpers → `view_layout.go`, commit
- [x] **1b** Move models/recents/settings/header rendering → `view_models.go`, commit
- [x] **1c** Move running list + output pane → `view_running.go`, commit
- [x] **1d** Move details/preview/VRAM → `view_details.go`, commit
- [x] **1e** Move all RPC rendering → `view_rpc.go`, commit

### Milestone Verification — Phase 1

```sh
go build ./...
go test ./...
# view.go should now be under 350 lines
wc -l internal/tui/view.go
```

---

## Phase 2 — Split `model.go` (922 → ~4 files)

**Goal:** Separate type definitions, row-building, status-server logic,
and background polling from the core Model struct and `New()`.

### Target file map

| New file | Functions/types to move |
|----------|-------------------------|
| `model_types.go` | `screen`, `paneFocus`, `leftMode`, `rowKind`, `row`, `healthMsg`, `tokSample`, `slotsMsg`, `vramMsg`, `remoteStatusMsg`, all screen/focus/mode constants |
| `model_rows.go` | `rebuildRows`, `rebuildRecentRows`, `buildSettingsRows`, `modelRowStyle`, `runningContains` |
| `model_status.go` | `shouldRunStatusServer`, `statusServerBindAddr`, `reconcileStatusServer`, `reconcileStatusPublisher`, `buildStatusSnapshot`, `pushStatusServer`, `clientID`, `clientName` |
| `model_checks.go` | `backgroundChecks`, `checkHealthCmd`, `checkSlotsCmd`, `checkVRAMCmd`, `checkRPCServerHealthCmd`, `pollRemoteStatusCmd` |
| `model.go` (remaining) | `Model` struct, `New()`, `Init()`, `refreshRunning`, `modelRunningStatus`, `setError`, `clearError`, `tailOrReason`, `logFileHasContent`, `applyTokSamples` |

### Tasks

- [x] **2a** Move all message/type definitions → `model_types.go`, commit
- [x] **2b** Move row-building functions → `model_rows.go`, commit
- [x] **2c** Move status-server logic → `model_status.go`, commit
- [x] **2d** Move background check commands → `model_checks.go`, commit

### Milestone Verification — Phase 2

```sh
go build ./...
go test ./...
# model.go should be under 350 lines
wc -l internal/tui/model.go
```

---

## Phase 3 — Split `update.go` (1,115 → ~4 files)

**Goal:** Keep the main `Update()` dispatcher and message handlers in
`update.go`; extract each input-handler domain into its own file.

### Target file map

| New file | Functions to move |
|----------|-------------------|
| `update_mouse.go` | `updateMouse` |
| `update_nav.go` | `moveFocusLeft`, `moveFocusRight`, `moveCursor`, `currentRow`, `selectRow`, `deleteSelected`, `duplicateSelectedProfile`, `updateModelSearch` |
| `update_main.go` | `updateMain` (the large keyboard routing function) |
| `update.go` (remaining) | `Update()` dispatcher + all `case <SomeMsg>:` blocks |

Note: network-specific input (`updateNetworkPicker`, `updateNetworkSwitch`,
`openNetworkPicker`, `openNetworkSwitch`) already lives in `network.go` —
confirm and leave it there.

### Tasks

- [x] **3a** Move `updateMouse` → `update_mouse.go`, commit
- [x] **3b** Move navigation helpers → `update_nav.go`, commit
- [x] **3c** Move `updateMain` → `update_main.go`, commit

### Milestone Verification — Phase 3

```sh
go build ./...
go test ./...
# update.go should be under 300 lines
wc -l internal/tui/update.go
```

---

## Phase 4 — Split `form.go` (1,221 → ~4 files)

**Goal:** Separate field definitions, CLI-arg parsing, and form
submission from the main form update/navigation logic.

### Target file map

| New file | Functions/types to move |
|----------|-------------------------|
| `form_types.go` | `formField`, `formState`, `formFieldIndex` constants (`fieldPort`…`fieldRPCEnabled`), `newTextInput` |
| `form_fields.go` | `fieldDefaultFlag`, `formFieldDescription`, `buildFormFields` |
| `form_parse.go` | `parseProfileArgs`, `applyImportedArgs`, `parseIntOrZero`, `parseIntPtr`, `parseFloatPtr`, `parseBoolPtr`, `parseReasoning`, `boolPtrOrEmpty`, `value` helper, `parsedProfile` |
| `form.go` (remaining) | `openForm`, `openEditForm`, `updateForm`, `submitForm`, `formDescriptionLineCount`, `formDescriptionVisibleLines`, `advanceDescriptionScroll` |

### Tasks

- [x] **4a** Move type/const definitions → `form_types.go`, commit
- [x] **4b** Move field metadata functions → `form_fields.go`, commit
- [x] **4c** Move parsing helpers → `form_parse.go`, commit

### Milestone Verification — Phase 4

```sh
go build ./...
go test ./... # form_test.go is the main coverage here
# form.go should be under 300 lines
wc -l internal/tui/form.go
```

---

## Phase 5 — Split `settings.go` (678 → ~2 files)

**Goal:** Separate settings type definitions and the RPC-specific
activation logic from the general settings routing.

### Target file map

| New file | Functions/types to move |
|----------|-------------------------|
| `settings_types.go` | `settingsCategoryDef`, `rpcContentState`, `statusServerContentState`, `settingsState`, `dirsContentState`, `binContentState`, all category `const` blocks |
| `settings_rpc.go` | `activateRPCRow`, `openRemoteStatusAddrForm`, `openRPCEndpointForm`, `openRPCServerBinForm`, `openRPCServerPortForm`, `copyFirewallRule` |
| `settings.go` (remaining) | `enterSettingsCategory`, `activateSettingsContentRow`, `settingsContentMoveCursor`, `activateStatusServerRow`, `openStatusServerHostForm`, `openStatusServerPortForm`, `deleteDirRow`, `openLlamaServerBinForm`, `saveAndApplyBin` |

### Tasks

- [x] **5a** Move settings type/const defs → `settings_types.go`, commit
- [x] **5b** Move RPC-specific activation → `settings_rpc.go`, commit

### Milestone Verification — Phase 5

```sh
go build ./...
go test ./... # status_server_test.go and update_test.go cover settings paths
wc -l internal/tui/settings.go
```

---

## Phase 6 — Client/Server Mode Separation (logic, not just files)

**Goal:** After all file splits are clean, consolidate all
`RPCMode == "client"` and `RPCMode == "server"` branches so each concern
is in one place. This phase involves logic changes, not just moves.

### Tasks

- [ ] **6a** Audit all `m.cfg.RPCMode` branches across the codebase
      (`grep -n 'RPCMode' internal/tui/*.go`)
- [ ] **6b** Consolidate server-mode-only rendering into `view_rpc.go`
      (no server branches scattered in `view.go` or `view_running.go`)
- [ ] **6c** Consolidate client-mode-only rendering into a new
      `view_rpc_client.go`
- [ ] **6d** Review `settings_rpc.go` — split client vs server settings
      handlers into clearly named funcs
- [ ] **6e** In `model_status.go`, document (comment) which functions are
      client-only vs server-only vs shared
- [ ] **6f** Remove any dead `RPCMode` checks that can't be reached given
      the current settings flow (verify with tests)

### Milestone Verification — Phase 6

```sh
go build ./...
go test ./...
# Manual: launch in server mode, verify RPC server tab works
# Manual: launch in client mode, verify RPC Connection tab works
# No RPCMode checks should appear in view.go, model.go, or update.go main bodies
grep -n 'RPCMode' internal/tui/view.go internal/tui/model.go internal/tui/update.go
```

---

## Phase 7 — Final Cleanup & PR

- [ ] **7a** Run `go vet ./...` and fix any issues
- [ ] **7b** Run `go test -race ./...` and fix any races
- [ ] **7c** Delete `REFACTOR.md` from the branch (or move to docs/)
- [ ] **7d** Open PR: `dev-refactor` → `dev`
- [ ] **7e** Squash-friendly: each phase's commits grouped together

### Final Verification

```sh
go build ./...
go test -race ./...
go vet ./...
# All files in internal/tui/ should be under 400 lines
Get-ChildItem internal/tui/*.go | ForEach-Object { "$($_.Name): $($(Get-Content $_).Count)" }
```

---

## Quick-Reference: File → Phase Mapping

| File | Phase |
|------|-------|
| `view.go` → 5 files | Phase 1 |
| `model.go` → 5 files | Phase 2 |
| `update.go` → 4 files | Phase 3 |
| `form.go` → 4 files | Phase 4 |
| `settings.go` → 3 files | Phase 5 |
| Logic consolidation | Phase 6 |
| PR + cleanup | Phase 7 |

---

## Progress Tracker

| Phase | Status |
|-------|--------|
| Phase 1 — Split view.go | ✅ Done (view.go: 1825 → 279 lines) |
| Phase 2 — Split model.go | ✅ Done (model.go: 922 → 370 lines) |
| Phase 3 — Split update.go | ✅ Done (update.go: 1115 → 334 lines) |
| Phase 4 — Split form.go | ✅ Done (form.go: 1221 → 688 lines) |
| Phase 5 — Split settings.go | ✅ Done (settings.go: 678 → 358 lines) |
| Phase 6 — Client/Server separation | ⬜ Not started |
| Phase 7 — Final cleanup + PR | ⬜ Not started |
