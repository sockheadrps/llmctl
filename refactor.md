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

- [ ] Run the full test suite and capture baseline:
  `go test ./... > /tmp/refactor-baseline.txt && echo OK`
- [ ] Verify build for all tag variants:
  `go build ./... && GOOS=windows go build ./... && go build -tags debug ./...`
  (whatever extra tags the project currently uses)
- [ ] Confirm git is clean for the branch we're on (`dev`):
  `git status` — no untracked/uncommitted changes in `internal/` or `cmd/`
- [ ] Create a dedicated refactor branch off current `HEAD`:
  `git checkout -b refactor/tui-cleanup`

---

## Section 1: Establish Conventions

Add a small `doc.go` (or header comment block) to `internal/tui` that
documents the filename-prefix contract. This gives the refactor a vocabulary
and prevents future drift.

Files to create/modify:
- [ ] Add or update `internal/tui/doc.go` with:
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

`internal/util` currently holds 6 files mixing formatting, networking, path
helpers, and model-domain helpers. Move domain-specific ones where they belong
so `util` is strictly generic.

Expected moves (verify by reading first — contents drive the decision):
- [ ] `internal/util/models.go` → `internal/models/` (likely belongs there;
  model-domain helpers should live next to model types). Update all imports.
- [ ] `internal/util/models_test.go` → moved along with `models.go`
- [ ] `internal/util/format.go` → stays in `util` (generic formatter)
- [ ] `internal/util/net.go`, `util/port.go` → stay in `util` (generic network)
- [ ] `internal/util/paths.go` → inspect; may belong in `internal/config` if
  it's exclusively about config paths, otherwise stays in `util`
- [ ] `internal/util/pointers.go` → stays in `util` (generic helper)

Verification (structural + tests):
- [ ] `go build ./...` clean
- [ ] `go test ./internal/util/...` — still passes (util's own tests moved
      with the code where applicable)
- [ ] `go test ./internal/models/...` — existing `models_test.go` still passes
      (if any tests moved here they must continue to pass)
- [ ] `go test ./internal/process/...` — `TestBuildArgs`,
      `TestBuildStartArgs*`, `TestParseProfileArgs*` still pass — these
      exercise util's arg-formatting helpers that may have moved
- [ ] `internal/util/` has no import of `internal/models`, `internal/process`,
      or any sibling internal package (dependency must flow one way):
      `go list -f '{{.Imports}}' ./internal/util/` should show only stdlib
      and third-party

If `models.go` tests were in `util/models_test.go` and moved to
`internal/models/models_test.go`, the new test file must contain a test for
EVERY exported symbol that was in the moved file — no coverage drop.

---

## Section 3: Extract Business Logic Out of `internal/tui`

The TUI currently contains pure business logic that doesn't need Bubbletea.
Move it to the package that owns the concept.

- [ ] `internal/tui/export_args.go` → move logic into `internal/models` (or a
  new `internal/export` package if it's large). The TUI only calls this to
  compute args for a run — that's domain logic, not UI.
- [ ] Update all call sites in the TUI to import the new location
- [ ] `internal/tui/run.go`, `start.go`, `stop.go`, `stop_confirm.go`,
  `rpc_server_action.go`, `running_action.go` — audit for pure business logic
  vs actual Bubbletea message handling. Anything that doesn't take or return
  `tea.Msg`/`tea.Cmd` and isn't part of a `view_*`/`update_*` chain belongs
  in `internal/runtime` or `internal/process`. Leave in place for this section
  if they're tightly coupled to TUI state; flag them for Section 6.

Verification:
- [ ] `go build ./...` clean
- [ ] `go test ./...` matches baseline
- [ ] **Specific to this section:** The `TestExportArgsRoundTrip` characterization
      test from Section -1 MUST still pass without modification. If it needs
      updating, the refactor changed observable behavior — stop and investigate.
- [ ] `grep -r "bubbletea\|tea.Msg\|tea.Cmd" internal/tui/` still shows the
      Bubbletea dependency where expected; extracted files do NOT import Bubbletea
- [ ] Extracted business-logic files have their OWN unit tests (not relying on
      TUI tests to exercise them). For every exported function moved out of tui,
      verify there's a pure unit test for it in its new home that exercises
      inputs/outputs without going through the TUI

---

## Section 4: Introduce a `Controller` Interface

Right now `internal/tui` imports 8 internal packages directly. Adding an
interface lets the TUI depend on a contract instead of concrete packages —
critical before splitting the package.

- [ ] Define `internal/tui/controller.go` with an interface like:
  ```go
  type Controller interface {
      // Models/profiles
      Models() []models.Model
      Profiles(modelKey string) []models.Profile
      Running() []models.Running

      // Lifecycle
      Start(modelKey, profileKey string) error
      Stop(instanceID string) error
      Restart(instanceID string) error

      // Config
      Config() *config.Config
      SaveConfig(cfg *config.Config) error

      // Diagnostics
      Health() HealthSnapshot
      RecentRuns() []models.RecentRun
      // ... extend as needed
  }
  ```
- [ ] Implement the interface in a new struct (likely `*runtime.Manager` or a
  thin `tui.ControllerImpl` wrapper). Wire it in through `tui.NewModel(...)`.
- [ ] Refactor the TUI's hot paths to call `m.ctrl.Method()` instead of
  reaching into `process`, `runtime`, `statusserver`, etc. directly.
- [ ] Remove direct imports of `process`, `runtime`, `statusserver` from
  `internal/tui` where the interface covers them. Some imports may remain
  for message types — document which and why.

Verification:
- [ ] `go build ./...` clean
- [ ] `go test ./...` matches baseline
- [ ] **Section -1 contract check:** `TestOverviewPageDimensions` (from Section -1)
  MUST still pass unchanged. If the tick handler needs to update its assertion,
  the refactor changed behavior — stop and investigate the exact line that changed.
- [ ] **Controller acceptance test passes:** The `TestControllerImpl...` tests
  written in Section -1 (`Start`, `Stop`, `Models`, `Config` round-trip) MUST
  pass against the new `ControllerImpl`. These ARE the verification tests for
  this section — writing them *before* Section 4 was the point.
- [ ] **Regression sweep over existing TUI tests:** The following existing tests
  exercise tick-driven behavior and must still pass with identical assertions:
      `TestTickPublishesStatusOutsideMainScreen`
      `TestStatusServerRunsWithoutRPC`
      `TestRPCClientModePublishesToRemoteStatusServer`
  If any of these need their assertion logic adjusted (not just the method they
  call through), the Controller interface is too leaky.
- [ ] Import check: `go list -deps ./internal/tui/` should no longer include
  `internal/process`, `internal/statusserver`, or `internal/runtime` (or only
  for pure message-type definitions — document any exceptions in `controller.go`).

Done when: Every test in Section -1 passes against the new Controller without
changing test code, and the three named tick-related tests pass unchanged.

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
- [ ] `go build ./...` clean
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
- [ ] `go build ./...` clean for the whole module
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
