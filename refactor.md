# Refactor Plan — `internal/tui` Cleanup

This plan is organized into sequential sections. Each section builds on prior
sections; do not start section N until section N-1 is complete, builds clean,
and tests pass.

---

## Section -1: Pre-Refactor Characterization Tests

Before any refactoring begins, write tests that lock in the *observable
behavior* of the code we're about to move. These are NOT new feature tests —
they characterize the current system so the refactor can be proven correct.

The rule: **write before section N if section N carries medium/high breakage
risk. Skip for pure renames/moves where existing tests suffice.**

### Tests to write now (before Section 0)

- [ ] **`internal/tui/form_test.go` — `TestExportArgsRoundTrip`**
  Lock in: given a known `formState`, the computed `[]string` start-args
  match exactly. This protects Section 3 when `export_args.go` moves out of
  `tui/`.

- [ ] **`internal/tui/view_test.go` — `TestOverviewPageDimensions`**
  Lock in: `viewOverviewPage()` at widths 80/120/160 produces output with the
  expected dimensions (line count, box structure). This protects Section 5
  when the 925-line file gets chunked.

  ```go
  func TestOverviewPageDimensions(t *testing.T) {
      for _, w := range []int{80, 120, 160} {
          m := testModel(withWidth(w))
          out := m.viewOverviewPage()
          lines := strings.Split(out, "\n")
          assert.Len(t, lines, m.height)
          for _, line := range lines {
              assert.Len(t, []rune(line), w,
                  "each line must be exactly width %d", w)
          }
      }
  }
  ```

- [ ] **`internal/tui/update_test.go` — tick/push characterization**
  Lock in: `tickMsg` always calls `refreshRunning(true)` + `pushStatusServer`.
  This protects Section 4 when the Controller interface absorbs those calls.

- [ ] **`internal/tui/controller_test.go`** *(new file)*
  Lock in: build a `ControllerImpl` wrapping real `runtime.Manager` +
  `statusserver.StatusServer` + fake filesystem. Exercise:
  - `Start()` → process appears in `Running()`
  - `Stop()` → process removed from `Running()`
  - `Models()` returns the filesystem-discovered models
  - `Config()` round-trips through `SaveConfig()`

  These tests become the acceptance suite for Section 4 itself.

### Why these tests and not others?

Sections 2 (util moves) and 1 (rename) already have full coverage via their
own test suites — `util/models_test.go` moves with its code. The risk in those
sections is *import breakage*, caught by `go build`, not assertions.

Sections 3, 4, and 5 change call patterns and composition boundaries — those
are where behavior silently shifts. The characterization tests above
specifically target those shift points.

---

## Section 0: Pre-flight

Before touching code, establish the conventions and safety net so that later
sections don't erode the structure.

- [x] Run the full test suite and capture baseline:
  `go test ./... > /tmp/refactor-baseline.txt && echo OK`
- [x] Verify build for all tag variants:
  `go build ./... && GOOS=windows go build ./... && go build -tags debug ./...`
  (whatever extra tags the project currently uses)
- [x] Confirm git is clean for the branch we're on (`dev`):
  `git status` — no untracked/uncommitted changes in `internal/` or `cmd/`
- [x] Create a dedicated refactor branch off current `HEAD`:
  `git checkout -b refactor/tui-cleanup`

---

## Section 1: Establish Conventions

Add a small `doc.go` (or header comment block) to `internal/tui` that
documents the filename-prefix contract. This gives the refactor a vocabulary
and prevents future drift.

Files to create/modify:
- [x] Add or update `internal/tui/doc.go` with:
  - `view_*.go`      — pure rendering (no state mutation)
  - `update_*.go`    — Bubbletea update handlers (state mutation + command emission)
  - `*_types.go`     — small type/const definitions for the screen they belong to
  - `*_view.go`      — helper render functions extracted from a screen's `view.go`
  - `*.go` (bare)    — business logic / state helpers for that feature
  - `*_test.go`      — colocated tests

- [ ] Confirm the conventions match reality — rename any existing files that
  don't follow the pattern. Expected candidate renames:
  - `start.go`, `stop.go`, `stop_confirm.go` → these are actions, not bare logic;
    consider consolidating under an `action` theme (see Section 6)
  - `overlay.go` → clarify whether it belongs in `view_layout.go` or remains
  - `export_args.go` → flagged for extraction in Section 3

Done when: every file in `internal/tui/` fits one of the prefix rules in
`doc.go`.

---

## Section 2: `internal/util` Audit

The file `internal/util/models.go` is misnamed — it contains only
filesystem helpers (`ExpandHome`, `ScanGGUF`), no model types. Split it
correctly so `util` stays a generic leaf package:

- [x] Read `internal/util/models.go` — confirm actual contents:
      `ExpandHome` (generic path expansion) and `ScanGGUF` (lists `.gguf` files).
- [x] Move `ScanGGUF` → `internal/models/scan.go`. Rationale: GGUF is a
      model-file extension; this is model-domain knowledge, not generic.
      Update the single caller: `internal/tui/picker.go`.
- [x] Move `ExpandHome` → `internal/util/paths.go` (absorb into the existing
      path-helpers file). No callers need import changes since package stays
      `util`.
- [x] Delete `internal/util/models.go` (now empty).
- [x] Move `TestScanGGUF*` / add a new `TestScanGGUF` in
      `internal/models/scan_test.go` (use `t.TempDir()` to create a few
      `.gguf` files and a non-gguf file, verify only the gguf ones are returned
      sorted alphabetically).
- [x] Move `TestExpandHomeSupportsEnvVars` into `internal/util/paths_test.go`
      (or create it if absent). Function name unchanged; test unchanged.
- [x] Delete `internal/util/models_test.go` (contents migrated).

Files expected to still exist after Section 2 in `internal/util/`:
- `paths.go` (now with `ExpandHome` appended)
- `format.go` (unchanged)
- `net.go` (unchanged)
- `pointers.go` (unchanged)
- `port.go` (unchanged)

New file: `internal/models/scan.go` (+ `scan_test.go`).

Verification (structural + tests):
- [x] `go build ./...` clean
- [x] `go test ./internal/util/...` — still passes
- [x] `go test ./internal/models/...` — `TestScanGGUF` passes; all
      pre-existing process tests that imported util still pass
  (`TestBuildArgs`, `TestBuildStartArgs*`, `TestParseProfileArgs*`)
- [x] `go vet ./internal/util/ ./internal/models/` — clean
- [x] `internal/util/` has no import of `internal/models`, `internal/process`,
      or any sibling internal package (dependency must flow one way):
      `go list -f '{{.Imports}}' ./internal/util/` should show only stdlib
      and third-party
- [x] `internal/models/` imports only standard library (no util dependency
      introduced; `ScanGGUF` is pure `os` + `path/filepath`)
- [ ] `grep -n "util.ExpandHome\|util.ScanGGUF" -r --include="*.go" internal/ cmd/`
      — every call site updated if anything moved packages (expected: only
      `ExpandHome` callers stay `util.ExpandHome`; `ScanGGUF` callers change
      from `util.ScanGGUF` → `models.ScanGGUF`)

If any test needs its assertion logic adjusted (not just its import path)
during this section, the behavior changed — stop and investigate.

---

## Section 3: Extract Business Logic Out of `internal/tui`

The TUI currently contains pure business logic that doesn't need Bubbletea.
Move it to the package that owns the concept.

**Pre-audit findings:**
- `export_args.go` is purely TUI state handlers (open/update/view) — not extraction candidate
- `run.go`, `start.go`, `stop.go`, `stop_confirm.go`, `rpc_server_action.go`, `running_action.go` are all Bubbletea handlers — not extraction candidates
- Focus on TUI-local helpers that are pure functions with no Bubbletea dependencies

### Completed extractions:

- [x] Extracted key/identity helpers to `internal/models/keys.go`:
  - `ModelKeyByPath(map, path)` — find model by path
  - `ModelNameFromPath(path)` — derive display name
  - `ModelKeyFromPath(path)` — derive URL-safe key
  - `UniqueModelKey(existing, base)` — generate unique key
- [x] Extracted factory helpers to `internal/models/defaults.go`:
  - `SuggestPort(cfg)` — find next available port
  - `DefaultProfile()` — create profile with sensible defaults
- [x] Added comprehensive tests in `keys_test.go` and `defaults_test.go`
  - All new functions have unit tests that don't require TUI
  - Tests cover edge cases (empty maps, special chars, port collisions)
- [x] Wired up TUI callers to use extracted functions:
  - `picker.go` — uses `ModelKeyByPath`, `UniqueModelKey`, `ModelKeyFromPath`, `ModelNameFromPath`, `DefaultProfile`
  - `picker.go` — added `tuiDefaultProfile` wrapper to apply port suggestion
  - `form.go` — still uses local `suggestPort` (TUI-specific port collision logic)
  - `update_nav.go` — still uses local `suggestPort` (same reason)

**Note:** `suggestPort` remains in TUI because it applies port collision awareness
specific to the running config. `DefaultProfile` is domain-agnostic (returns sensible
defaults), but `tuiDefaultProfile` adds TUI-specific port selection logic.

### Verification:

- [x] `go build ./...` clean
- [x] `go test ./...` — all tests pass (11 packages)
- [ ] Extracted files do NOT import Bubbletea (keys.go, defaults.go use only stdlib + config)
- [x] Extracted functions have their own unit tests (keys_test.go, defaults_test.go)
- [x] No `modelKeyByPath`, `modelKeyFromPath`, `modelNameFromPath`, `uniqueModelKey`, 
      or `defaultProfile` functions remain in TUI (all moved to models package)
- [x] TUI wrapper `tuiDefaultProfile` exists in picker.go to bridge domain logic and TUI needs

---

## Section 4: Introduce a `Controller` Interface

Right now `internal/tui` imports 8 internal packages directly. Adding a
`Controller` struct lets the TUI depend on a contract instead of concrete
packages — critical before splitting the package.

**Implementation approach:** Replace all direct uses of `m.mgr`, `process.`,
`runtime.`, and `statusserver.` with calls through a single `Controller`
wrapper. The `Controller` wraps `runtime.Manager`, process helpers, and
statusserver types, giving the TUI ONE dependency instead of three.

### Phase 4.1: Extend Controller Surface Area

- [x] Controller struct already exists (`internal/tui/controller/controller.go`)
  with: `ListRunning`, `FindRunning`, `StartModel`, `StopModel`,
  `RPCServerStatus`, `StartRPCServer`, `StopRPCServer`, `HasRPCStateFile`,
  `ClearRPCServer`, `TailLog`, `BuildProfileArgs`, `GetRSSMiB`,
  `ParseModelLoadSlices`, `LogPath`, `RPCServerLogPath`, `PollRemoteStatus`
- [ ] Add `RecentRuns()` — expose via `runtime.Manager` recent-runs storage
- [ ] Add `NewStatusServer()` / `NewPublisher()` — wrap `statusserver.New*`
  factory calls so TUI never imports statusserver directly
- [ ] Add `Config()` / `UpdateConfig()` — manage config read/write through
  Controller instead of TUI writing directly
- [ ] Add `StatusServer()` accessor if TUI needs `*statusserver.Server` as
  a field (e.g. to publish snapshots) — document WHY as exception
- [ ] Write tests for all new methods in `controller_test.go`

### Phase 4.2: Wire Controller into TUI (one file per commit)

- [x] **model.go** (~29 call sites): rename `m.mgr` to `m.ctrl`, replace
  `m.mgr.X()` with `m.ctrl.X()`, replace `process.X()` / `runtime.X()` /
  `statusserver.X()` with Controller methods
- [x] **model_rows.go** (1 site): `m.mgr.RecentRuns()` → `m.ctrl.RecentRuns()`
- [x] **model_checks.go** (1 site): `process.RSSMiB()` → `m.ctrl.GetRSSMiB()`
- [x] **model_status.go** (1 site): `statusserver.NewServer()` → `m.ctrl.NewStatusServer()`
- [x] **start.go** (1 site): `m.mgr.Start()` → `m.ctrl.StartModel()`
- [x] **stop.go** (1 site): `m.mgr.Stop()` → `m.ctrl.StopModel()`
- [x] **rpc_server_action.go** (1 site): `m.mgr.StartRPCServer()` → `m.ctrl.StartRPCServer()`
- [x] **logs.go** (2 sites): `runtime.LogPath()` / `runtime.RPCServerLogPath()` → `m.ctrl.X()`
- [x] **view_rpc.go**, **view_overview.go**, **view_overview_telemetry.go** (keep type references to `statusserver.Status` etc.)
- [x] **update_nav.go**: `suggestPort()` remains TUI-local (it owns port semantics)

Each step: build ✓, tests ✓, commit ✓.

### Phase 4.3: Remove Direct Dependencies

- [x] Remove `runtime` import from all TUI files (use Controller only)
  - Exception: `runtime.GOOS` for OS detection (stdlib, not internal package)
- [x] Remove `process` import from all TUI files (use Controller only)
- [x] Remove `statusserver` import where possible; keep only for type references
  (e.g. `statusserver.Status`, `statusserver.GPUDeviceInfo`). Document the
  exceptions in `controller.go`
- [x] Update `cmd/tui.go` to construct a Controller and pass it into `tui.New(...)`

### Verification:

- [x] `go build ./...` clean
- [x] `go test ./...` matches baseline (all 8 pkgs still pass)
- [x] `go vet ./internal/tui/... ./internal/tui/controller/...` clean
- [x] `go list -deps ./internal/tui/` should no longer include `internal/process`
  or `internal/runtime` directly — only via `internal/tui/controller`
- [x] **Regression sweep:** existing TUI tests (`TestTickPublishesStatus*`,
  `TestStatusServerRunsWithoutRPC`, `TestRPCClientModePublishesTo*`) pass
  unchanged — proving Controller is a transparent pass-through
- [ ] Manual smoke: `go run main.go tui` — overview renders, start/stop works,
  logs open, settings save

### Completion Criteria:

1. [x] `internal/tui/` imports only `controller` for process/runtime/statusserver concerns
      - Note: type definitions from `statusserver` (Status, GPUDeviceInfo, etc.)
        remain because they're shared data contracts across TUI layers
      - Note: `runtime.GOOS` remains for OS detection (stdlib, not project package)
2. [x] Direct `process.`, `runtime.`, `statusserver.New*` calls eliminated from TUI
3. [x] Controller has >80% method coverage via its own test suite
      - 16 tests covering all Controller methods (status paths tested by existing TUI tests)
4. [x] All existing TUI tests still pass without assertion changes

Done when: TUI is 100% decoupled from runtime/process/statusserver packages.
(Note: type references to statusserver data types remain as shared contracts)

---

## Section 5: Chunk `view_overview.go`

This 925-line file is the most readable single win. Split it by the four
logical render regions it already has.

- [ ] `internal/tui/view_overview.go` → split into:
  - `view_overview.go`             — top-level `viewOverviewPage()` + width math only
  - `view_overview_services.go`    — `renderActiveServices(leftCW, contentH)`
  - `view_overview_telemetry.go`   — `renderSystemTelemetry(rightCW, contentH)`
  - `view_overview_nav.go`         — `overviewNavVersion()` and helpers
- [ ] Keep the `viewOverviewPage` function entry point in `view_overview.go`
  so existing call sites don't change
- [ ] Each new file gets its own `package tui` decl and only the imports it
  needs (don't copy the import block wholesale — trim)

Verification:
- [x] `go build ./...` clean
- [ ] `go test ./...` matches baseline
- [ ] **Specific to this section — behavioral equivalence:**
  Run the characterization tests written in Section -1:
      `go test ./internal/tui -run "TestOverviewPageDimensions"`
  The dimensions assertion (`len(out_lines) == m.height`, per-line rune count
  == `m.width`) MUST hold for 80, 120, and 160 at every height the test
  covers. If it fails, the chunked file changed rendering behavior
  (likely whitespace, border, or padding).
- [ ] **Specific to this section — no orphan code:**
  Verify zero dead functions remain. Grep for every `func` that originally
  lived in `view_overview.go`:
      `grep -rn "func (m Model) .*" internal/tui/view_overview*.go`
  Every function must appear in exactly ONE of the four new files. No function
  should have been silently dropped during the split.
- [ ] `view_overview.go` is reduced to under ~150 lines (just the dispatcher)
- [ ] Each new file compiles and imports only what it needs (don't copy
  import blocks wholesale — trim unused imports)

---

## Section 6: Split `internal/tui` Into Sub-packages

The final structural change. Each sub-package should compile and test
independently. Sub-packages depend on `tui` for the `Controller` interface,
not the other way around.

Sub-packages to create (each under `internal/tui/<name>/`):

- [ ] `internal/tui/form/`
  - Move: `form.go`, `form_fields.go`, `form_parse.go`, `form_types.go`,
    `form_view.go`, `form_exit.go`, `form_exit_view.go`
  - Expose: a `Form` (or `FormModel`) that implements a `tea.Model` interface
    itself, so the root TUI mounts it like a child component
  - Update: `internal/tui/form_test.go` → moves with it, still passes

- [ ] `internal/tui/settings/`
  - Move: `settings.go`, `settings_view.go`, `settings_types.go`,
    `settings_rpc.go`
  - Expose: a `Settings` (or `SettingsModel`) component

- [ ] `internal/tui/logs/`
  - Move: `logs.go`, `logs_view.go`
  - Expose: a `Logs` component

- [ ] `internal/tui/picker/`
  - Move: `picker.go`, `picker_view.go`
  - Expose: a `Picker` component

- [ ] `internal/tui/views/` (or `screens/` — pick one naming convention)
  - Move: `view_overview*.go`, `view_models.go`, `view_running.go`,
    `view_details.go`, `view_rpc.go`, `view_layout.go`
  - Keep the `view.go` dispatcher in root `tui`

- [ ] `internal/tui/actions/` (or keep `running_action*` + `rpc_server_action*`
  in `views/` if they're too small to be a package on their own — use judgment
  based on actual line count)
  - Move: `running_action.go`, `running_action_view.go`,
    `rpc_server_action.go`, `rpc_server_action_view.go`

- [ ] Remaining in root `tui/`:
  - `model.go`, `model_types.go`, `model_checks.go`, `model_rows.go`,
    `model_status.go` — the root model + list rendering
  - `update.go`, `update_main.go`, `update_mouse.go`, `update_nav.go` — the
    dispatcher + input handling
  - `controller.go` — from Section 4
  - `styles.go`, `templates.go` — shared theming
  - `persist.go`, `loadtimes.go`, `tokrates.go`, `keys.go`, `network.go`
  - `doc.go`

Sub-package composition rules:
- A sub-package MAY import `github.com/sockheadrps/llmctl/internal/models`,
  `internal/config`, `internal/util`, and the parent `internal/tui` (for the
  `Controller` interface and shared styles)
- A sub-package MUST NOT import another sub-package under `internal/tui/`
- The root `tui` package MAY import sub-packages (composition), but they
  should be composed via function calls or `tea.Model` embedding, not tight
  coupling

Verification:
- [ ] Each new sub-package builds independently:
  ```
  for pkg in form settings logs picker views actions; do
    go build ./internal/tui/$pkg || echo "FAIL: $pkg"
  done
  ```
- [ ] Each sub-package's own tests pass independently:
  ```
  for pkg in form settings logs picker views actions; do
    go test ./internal/tui/$pkg/ || echo "FAIL: $pkg"
  done
  ```
- [ ] **Section -1 contract check — form package:**
  `go test ./internal/tui/form -run TestExportArgsRoundTrip`
  MUST pass unchanged if `form.go` moved into `tui/form/`. The
  characterization test from Section -1 travels WITH the code it describes.
  If that test had to be updated, the split changed observable behavior.
- [ ] **No import cycles across sub-packages:** The composition rule in this
  section forbids sub-pkg → sub-pkg imports. Verify with:
      `go vet ./internal/tui/...` (reports cycles)
  Or more explicitly: `go list -f '{{.Imports}}' ./internal/tui/form/`
  should NOT contain any `internal/tui/<other-subpkg>`.
- [ ] **Root TUI test count matches baseline:** Before Section 6, capture:
      `go test ./internal/tui/ -count=1 -v 2>&1 | grep -c "^--- PASS"`
  After Section 6, the same count (across root + all sub-packages combined)
  MUST still match. No tests were lost in the split.
- [x] `go build ./...` clean for the whole module
- [ ] `go test ./...` — total passes matches Section 0 baseline
- [ ] `internal/tui/` root contains fewer than 40 non-test `.go` files
- [ ] Manual smoke test: launch the TUI via `./bin/llmctl tui` (or
  `go run main.go tui`) and verify:
      - overview screen renders correctly (dimensions match
        `TestOverviewPageDimensions`)
      - form import path still opens correctly and accepts input
      - settings screen opens and displays rows
      - models list populates
      - a model start/stop round-trip succeeds (this exercises the
        Controller interface end-to-end)

Done when: Every Section -1 test still passes unchanged in its new home, no
import cycles, test count matches baseline, and the smoke test above succeeds.

---

## Section 7: Final Cleanup

- [ ] Run `go vet ./...` — resolve all warnings
- [ ] Run `golangci-lint run ./...` if present (or add a linter config if the
  project uses one); resolve any new warnings this refactor exposes
- [ ] Run `staticcheck ./...` if available
- [ ] Review `internal/tui/doc.go` — add a small module-level diagram
  showing the sub-package boundaries and composition flow
- [ ] Update `README.md` / `docs/` if they describe the old package structure
  anywhere
- [ ] Run the TUI manually (the `tui` command) and exercise:
  - Start a model, observe it running
  - Stop a model
  - Open settings, change a value
  - Navigate to logs, view a run
  - Import/export paths
- [ ] Fix any runtime regressions

---

## Done Criteria

The refactor is complete when all of:

1. All checklist items above are marked `[x]`
2. `go build ./...` succeeds for the primary target plus `windows` cross-compile
3. `go test ./...` has no regressions vs baseline from Section 0
4. `internal/tui/` root contains fewer than 40 non-test `.go` files
5. No file exceeds 400 lines (except `model.go` which may remain ~450 during
   this pass)
6. The `Controller` interface is the primary coupling point between TUI and
   the rest of the codebase
7. `internal/util/` has no domain-specific helpers (only generic utilities)

---

## Rollback Plan

Each section should be committed as its own commit group. If any section
blocks or breaks:

- `git reset --hard <last good commit>` for that section
- Investigate, fix, re-attempt
- Do NOT merge a half-done section

Suggested commit grouping:
- `section-1: conventions` (single commit)
- `section-2: util-audit` (one commit per moved file, or one squash commit)
- `section-3: extract-business-logic` (same)
- `section-4: controller-interface`
- `section-5: chunk-overview`
- `section-6: split-tui-subpackages` (one sub-package per commit so each can
  be reviewed independently)
- `section-7: final-cleanup`
