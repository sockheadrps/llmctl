# UI/UX Improvement Checklist

## Visual Polish

- [ ] **Expand color palette** — Currently uses only ANSI 24-color codes (205=magenta focus, 240/245 gray). Add semantic colors: blue for info/loading, amber for warnings.
- [ ] **Improve focus indicator** — Magenta-on-gray border change is subtle. Consider adding a subtle background highlight on the focused pane border, not just color change.
- [ ] **Tab bar visual hierarchy** — Models/Recents/Settings/Running all render identically with `[ label ]`. Make active tab visually distinct with background fill or underline, not just bold text.
- [ ] **Densify help bar** — `←/→ switch  ↑/k up  ↓/j down  enter select/run  s stop  e logs  del delete (press twice)  q quit` takes a full line. Group related keys or use symbols to reduce width.

## Layout & Information Density

- [ ] **Fix details panel layout** — `pair()` produces cramped two-column output like `Port: 8080 Ctx Size: 4096`. Fixed-width `%-24s` pushes values off-screen at narrow terminals. Consider single-column stacked layout or responsive width.
- [ ] **Group form fields** — The profile form has 14 fields in a single scrollable list (Key, Port, Ctx Size, Temp, Top P, Top K, Min P, Presence Penalty, Repetition Penalty, GPU Layers, Cache K/V, Extra Args, Notes, Flash toggle, Save). Group related fields with section headers (sampling params, cache params, etc.).
- [ ] **Reconsider VRAM bar placement** — `renderVRAMHeader()` places the VRAM bar inline with the "Running" title text, competing for attention. Move to its own line or into the details panel.

## Interaction Feedback

- [ ] **Add timer to pending-delete flash** — `pendingDeleteStyle` reverse-highlighting works but "(del again to confirm)" can be missed. Add a brief timer-based fade-out on first press so user knows they have a window to press again.
- [ ] **Add loading indicator for model picker** — Directory scanning has no visual feedback beyond "scanning /path1, /path2". Add a spinner or animated dots.

## Modal Overlays

- [ ] **Improve overlay centering** — `overlayCenter()` replaces background lines in a row band; can produce odd wrapping at non-standard terminal widths. Consider using `lipgloss.Place` with dimmed backdrop for true modal overlay effect.
- [ ] **Unify confirm/action modals** — `confirm_view.go` and `running_action_view.go` share the same pattern but have different layouts. Consolidate into a shared modal component with configurable buttons.

## Settings Screen

- [ ] **Reconsider Settings tab** — Only one category exists (`model_dirs`) yet it gets a full tab. Either collapse Settings into a context menu/sidebar until there are multiple categories, or add more settings (hotkeys, appearance, default profile params).
