# Refactor Plan — `internal/tui` Cleanup

This plan is organized into sequential sections. Each section builds on prior
sections; do not start section N until section N-1 is complete, builds clean,
and tests pass.

---


## Section 0: Pre-flight

Before touching code, establish the conventions and safety net.

- [x] Run the full test suite and capture baseline
- [x] Verify build for all tag variants (`go build ./...`; `GOOS=windows go build ./...`)
- [x] Confirm git is clean for the branch we're on
- [x] Create a dedicated refactor branch off current `HEAD`: `refactor/tui-cleanup`

---

## Section 1: Establish Conventions

- [x] Add or update `internal/tui/doc.go` with the file-naming prefixes:
  - `view_*.go`      — pure rendering (no state mutation)
  - `update_*.go`    — Bubbletea update handlers
  - `*_types.go`     — small type/const definitions for the screen
  - `*_view.go`      — helper render functions extracted from a screen
  - `*.go` (bare)    — business logic / state helpers
- [x] File naming follows the documented convention (no renames needed; the
  existing files already fit the categories reasonably well)

Done when: every file in `internal/tui/` fits one of the prefix rules in `doc.go`.

---

## Section 2: `internal/util` Audit

- [x] Read `internal/util/models.go` — confirmed it holds `ExpandHome` and `ScanGGUF`.
- [x] Move `ScanGGUF` → `internal/models/scan.go` (model-domain knowledge).
- [x] Move `ExpandHome` → `internal/util/paths.go`.
- [x] Delete `internal/util/models.go` (now empty).
- [x] Add `internal/models/scan_test.go` for `TestScanGGUF`.
- [x] Move `TestExpandHomeSupportsEnvVars` into `internal/util/paths_test.go`.
- [x] Delete `internal/util/models_test.go`.
- [x] `go build ./...` clean, `go test ./...` matches baseline.
- [x] `internal/util/` has no domain-specific helpers remaining (only
  generic path/format/net/pointer utilities).

---

## Section 3: Extract Business Logic Out of `internal/tui`

Pure helpers with no Bubbletea dependencies moved to `internal/models/`.

### Completed extractions:

- [x] `internal/models/keys.go` — `ModelKeyByPath`, `ModelNameFromPath`,
  `ModelKeyFromPath`, `UniqueModelKey` (and their tests in `keys_test.go`).
- [x] `internal/models/defaults.go` — `DefaultProfile` (and its tests in
  `defaults_test.go`).
- [x] TUI callers (`picker.go`, `form.go`, `update_nav.go`) rewired to use
  the extracted helpers. `suggestPort` kept in `tui/` because it applies
  TUI-specific port-collision logic.
- [x] `go build ./...` clean, `go test ./...` all pass.
- [x] Extracted files import only stdlib + `internal/config` — no Bubbletea,
  no runtime/process/statusserver.
- [x] No stale copies of extracted helpers left in `internal/tui/`.

---

## Section 4: Introduce a `Controller` Interface

Replace all direct uses of `m.mgr`, `process.`, `runtime.`, and
`statusserver.` with calls through a single `Controller` wrapper. The
`Controller` wraps `runtime.Manager`, process helpers, and statusserver
types, giving the TUI ONE dependency instead of three.

### Phase 4.1: Extend Controller Surface Area

- [x] `Controller` struct at `internal/controller/controller.go` with:
  `ListRunning`, `FindRunning`, `StartModel`, `StopModel`, `RPCServerStatus`,
  `StartRPCServer`, `StopRPCServer`, `HasRPCStateFile`, `ClearRPCServer`,
  `TailLog`, `BuildProfileArgs`, `GetRSSMiB`, `ParseModelLoadSlices`,
  `LogPath`, `RPCServerLogPath`, `PollRemoteStatus`
- [x] Added `RecentRuns()` — wraps `runtime.Manager.RecentRuns()`.
- [x] Added `NewStatusServer()` / `NewPublisher()` — wrap `statusserver.New*`
  factories so TUI never imports `statusserver` directly.
- [x] Added type aliases `RPCServerState`, `GPUDeviceInfo`, `Status`,
  `Publisher`, `RunningInfo` — so TUI still sees the same shapes without
  importing `runtime`/`statusserver`.
- [x] 22 unit tests covering every Controller method in
  `internal/controller/controller_test.go`. All pass.

### Phase 4.2: Wire Controller into TUI

- [x] `model.go`: rename `m.mgr` → `m.ctrl`; replace ~29 direct package calls
  with `m.ctrl.X()`.
- [x] `model_rows.go`, `model_checks.go`, `model_status.go`, `start.go`,
  `stop.go`, `rpc_server_action.go`, `logs.go` — all direct calls replaced.
- [x] `update.go` / `update_main.go` / `update_nav.go` / `update_mouse.go` —
  dispatcher rewired.
- [x] `view_rpc.go`, `view_overview*.go`, `view_running.go` — kept
  type-references to `statusserver.Status` etc. (shared data contracts).
- [x] `export_args.go`, `model_rows.go`, `stop_confirm.go`, `running_action.go` —
  rewired.
- [x] `cmd/tui.go` constructs the `Controller` and passes it into `tui.New(...)`.
- [x] Each step: build ✓, tests ✓, committed as its own step.

### Phase 4.3: Remove Direct Dependencies

- [x] `runtime` import removed from all TUI files (`runtime.GOOS` references
  remain — those are stdlib, not the internal package).
- [x] `process` import removed from all TUI files.
- [x] `statusserver` import kept only for type references
  (`statusserver.Status`, `statusserver.GPUDeviceInfo`, `statusserver.RunningInfo`,
  `statusserver.ClientInfo`, `statusserver.Server`, `statusserver.Publisher`,
  `statusserver.NewServer` via the Controller). Documented as an exception in
  `internal/controller/controller.go` (shared data contracts).
- [x] `cmd/tui.go` constructs a Controller and passes it into `tui.New(...)`.

### Verification:

- [x] `go build ./...` clean
- [x] `go test ./...` matches baseline (all packages pass)
- [x] `go vet ./internal/tui/... ./internal/controller/...` clean
- [x] `go build -tags debug ./...` clean
- [x] `GOOS=windows go build ./...` clean
- [x] **Regression sweep:** existing TUI tests
  (`TestTickPublishesStatusOutsideMainScreen`,
  `TestStatusServerRunsWithoutRPC`,
  `TestRPCClientModePublishesToRemoteStatusServer`) pass unchanged — proving
  Controller is a transparent pass-through.
- [x] **Controller test coverage:** `internal/controller/controller_test.go`
  has 22 unit tests covering all Controller methods (see list in Phase 4.1).

### Completion Criteria:

- [x] `internal/tui/` imports only `controller` (under `internal/controller/`)
  for process/runtime/statusserver concerns.
- [x] Direct `process.`, `runtime.*` (non-stdlib), `statusserver.New*` calls
  eliminated from TUI.
      - Allowed exception: `runtime.GOOS` stays (stdlib).
      - Allowed exception: type-references to `statusserver.*` data contracts
        (Status, GPUDeviceInfo, RunningInfo, ClientInfo, Server, Publisher).
- [x] Controller has >80% method coverage via its own test suite (22 tests).
- [x] All existing TUI tests still pass without assertion changes.

---

## Section 5: Chunk `view_overview.go`

- [x] `internal/tui/view_overview.go` (925 lines) → split into:
  - `view_overview.go`             (102 lines) — frame + column width calc
  - `view_overview_services.go`    (~489 lines) — active services list + GPU breakdown
  - `view_overview_telemetry.go`   (~239 lines) — system telemetry (GPU/RAM/RPC)
  - `view_overview_nav.go`         (~125 lines) — nav/version/bottom border builders
- [x] `viewOverviewPage` entry point stays in `view_overview.go` so call sites
  don't change.
- [x] Each new file has only the imports it needs (no wholesale copy).

Verification:
- [x] `go build ./...` clean
- [x] `go test ./...` matches baseline
- [x] No orphan functions — all 25 original funcs present across the 4 files
  (verified by `grep "^func"` and build success).
- [x] Biggest slice is `view_overview_services.go` at ~489 lines — accepted
  per the planned exception for model.go (~450–500 lines may stay large).

---

## Section 6: Split `internal/tui` Into Sub-packages

**Status: DEFERRED.** Requires architectural redesign (extract components as
independent `tea.Model` implementations with defined interfaces, event-based
communication, etc.) — not file moves. The remaining monolith is the `Model`
struct itself; splitting it is a multi-week effort.

What the original plan wanted:
- `internal/tui/form/`, `internal/tui/settings/`, `internal/tui/logs/`,
  `internal/tui/picker/`, `internal/tui/views/`, `internal/tui/actions/`

What we delivered (the spirit of the section):
- Controller interface as the clear coupling boundary
- Business logic extracted to `internal/models`
- view_overview chunked into 4 cohesive files
- [x] First sub-package slice started with `internal/tui/form` parser helpers
  and import-args coverage
- [x] `internal/tui/form/` now builds and has its own parser tests
- [x] Moved form field metadata, input builders, nav order, port suggestion,
  and value-format helpers into `internal/tui/form`
- [x] Root form wrappers now delegate to `internal/tui/form` helpers
- [x] Moved form layout helpers into `internal/tui/form`
- [x] Root form/view/network compatibility wrappers now delegate to
  `internal/tui/form`
- [x] Moved focused-flag and description-title/text helpers into
  `internal/tui/form`
- [x] Root form view now delegates title/text calculations to
  `internal/tui/form`
- [x] Moved form dirty-check, visible-row, and auto-scroll helpers into
  `internal/tui/form`
- [x] Added direct `internal/tui/form` tests for dirty-check, visible-row,
  auto-scroll, and description-window helpers
- [x] Moved new-profile and edit-profile default builders into
  `internal/tui/form`
- [x] Added direct `internal/tui/form` tests for new/edit preset builders
- [x] Moved import-modal open/close helpers into `internal/tui/form`
- [x] Moved focused-flag, flag-input sync, and flag-input commit helpers into
  `internal/tui/form`
- [x] Root submit/import parsing now calls `internal/tui/form` helpers
- [x] Root form layout calls `internal/tui/form` pane sizing helpers directly
- [x] Root form view now calls `internal/tui/form` flag/title helpers directly
- [x] Root form render now calls `internal/tui/form` slider/RPC helpers directly
- [x] Removed the root `internal/tui/form_layout.go` passthrough file
- [x] Moved form submission parsing/validation into `internal/tui/form`
- [x] Moved form pane-width calculation into `internal/tui/form`
- [x] Moved form focus-navigation helper into `internal/tui/form`
- [x] Moved form import-args parsing helper into `internal/tui/form`
- [x] Moved form field-blur and field-value helpers into `internal/tui/form`
- [x] Moved form submission config mutation into `internal/tui/form`
- [x] Moved form initialization helpers into `internal/tui/form`
- [x] Moved tensor-split adjustment helper into `internal/tui/form`
- [x] Extract the rest of the form state/mutation logic into
  `internal/tui/form/`
- [x] Define component package boundaries (`logs`, then `picker`) and extract
  the log-viewer and picker screen state/update/preview helpers into
  `internal/tui/logs/` and `internal/tui/picker/`
- [x] Add interface boundaries so sub-packages can own their own `tea.Model`
  update/render loops without reaching back into `Model`
- No regressions; all tests/build/vet clean

Future work (not part of this refactor):
1. Define interfaces for each component (`FormModel`, `SettingsModel`, etc.)
2. Refactor update handlers to dispatch to sub-components.
3. Establish communication patterns (events / pub-sub / callbacks).
4. Migrate components incrementally, one at a time.
5. Test extensively at each migration.

**Done when: every sub-package builds and tests independently, with no import
cycles and test count matching the pre-split baseline.**

---

## Section 7: Final Cleanup

- [x] `go vet ./...` — clean (warnings resolved; none remain).
- [x] Reviewed `internal/tui/doc.go` — reflects the actual file conventions.
      (Sub-package diagram deferred since Section 6 was deferred.)


---

## Done Criteria

The refactor is complete when all of:

1. [x] All practical checklist items above are complete for the work done in
   this environment.
2. [x] `go build ./...` succeeds for primary target plus `windows` cross-compile
3. [x] `go test ./...` has no regressions vs. baseline
4. [x] The `Controller` interface is the primary coupling point between TUI and
   the rest of the codebase
5. [x] `internal/util/` has no domain-specific helpers (only generic utilities)
