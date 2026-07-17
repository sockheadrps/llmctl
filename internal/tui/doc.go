// Package tui implements llmctl's interactive terminal UI, built on
// Bubbletea. It owns the Model/Update/View loop, the keyboard and mouse
// input dispatchers, and the rendering of each screen.
//
// # File-naming conventions
//
// Every file in this package falls into exactly one category. The prefix
// is meaningful and new files must respect it:
//
//   - view_*.go        Pure rendering. Functions take state and return a
//                      string. No state mutation, no command emission,
//                      no Bubbletea.Update-style logic.
//   - update_*.go      Bubbletea Update handlers. Accept a tea.Msg, return
//                      (tea.Model, tea.Cmd). State mutation happens here.
//   - *_types.go       Small type and constant definitions for the screen
//                      or feature they belong to. No logic.
//   - *_view.go        Helper render functions extracted from a parent
//                      screen's view.go (e.g. form_view.go is pulled out
//                      of a form.go dispatcher). Pure rendering.
//   - *.go (bare)      Business logic / state helpers / commands for the
//                      feature they belong to. May take or return Bubbletea
//                      types; no rendering.
//   - *_other.go       Build-tag gated alternatives (e.g. clipboard_other.go
//                      for !linux && !darwin && !windows).
//
// Adding a file outside these conventions requires updating this doc block
// and a commit message explaining why.
package tui
