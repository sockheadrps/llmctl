# Screenshot Checklist

All screenshots go in `readmeassets/`. Replace any existing file with the same name.

---

## Hero

- [x] **`launch.gif`** — Animated screen recording showing the full app flow: launch → browse models → run a profile → watch it go from loading to up → open logs → stop. This is the first thing people see. Keep it under ~30 seconds.

---

## First Launch

- [x] **`launch.jpg`** — The TUI on first load, before any models are configured. Tab bar visible at top, empty state in both panes, footer hotkeys showing.

---

## Settings

- [x] **`settingsmodelsdir.jpg`** — Settings tab selected, Model Directories category active in right pane. Show at least one directory in the list with the cursor on it.

- [x] **`settingsrpc.jpg`** — Settings tab, RPC Server category active. Cursor on row 3 (Network Tab) so the description text is visible in the right pane. All four rows visible: Toggle RPC (Enabled), Endpoint, RPC Binary, Network Tab (Enabled).

---

## Models

- [x] **`addmodel.jpg`** — The GGUF file picker modal open, listing several `.gguf` files with the cursor on one.

- [x] **`modelsselect.jpg`** — Models tab, one model expanded showing its profiles underneath. Cursor on the model row (not inside profiles yet).

- [x] **`savedprofileselect.jpg`** — Models tab, cursor inside a model's profile list. At least two profiles visible plus `+ New Profile`. Active model underlined.

- [x] **`newprofile.jpg`** — Profile editor with fields filled in (port, ctx size, gpu layers). Works for both new and edited profiles.

- [x] **`export.jpg`** — Export modal open, showing the format options (shell script, copy, etc.). Cursor on one option.

---

## Running a Profile

- [x] **`moderun.jpg`** — The Run/Edit choice modal open over the main screen. Cursor on "Run".

---

## Running Models

- [x] **`modelrunning_loading.jpg`** — Main screen (Models tab) with a running profile showing the yellow `●` loading indicator and "loading" badge. Capture this right after hitting Run, while the model is still initialising.

- [x] **`modelrunning.jpg`** — Main screen with a profile showing the green `●` up indicator, port, and ideally a token rate visible.

---

## Running Tab

- [x] **`runningselect.jpg`** — Running tab selected, a profile highlighted in the left pane, right pane showing the live log preview tail.

- [x] **`modeloutputorstop.jpg`** — The action modal open on a running profile, showing the options (View Output, Stop, Copy Endpoint). Cursor on one option.

- [x] **`logs.jpg`** — Full log viewer open for a running or recently stopped profile. Several lines of llama-server output visible.

---

## Network Tab *(Linux only)*

- [x] **`networktab.jpg`** — Network tab selected. Status section showing active connection and link state. Switch and Configure rows visible in left pane. Right pane showing the details for whichever row is focused.

- [x] **`networktab_picker.jpg`** — Connection picker modal open (after selecting "Set internet conn…" or "Set RPC conn…"), listing two or three nmcli connection profiles, one marked active with a green dot.

---

## Notes

- Aim for a consistent terminal size across all shots (e.g. 220×50).
- Use a clean profile/color scheme so text is legible at reduced width in the README.
- The `launch.gif` should be recorded at the same terminal size as the stills.
