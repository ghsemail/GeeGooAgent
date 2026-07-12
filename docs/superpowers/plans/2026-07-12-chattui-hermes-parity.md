# Chat TUI Hermes Parity Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ship a Bubble Tea TUI for `geegoo chat` with Hermes-like collapsible thinking/tools, `/details`, overlays, mouse, and multi live-session (Phases A→B→C).

**Architecture:** Keep Agent Loop unchanged; UI consumes `EmitProgress` via `ProgressSink`. New `internal/cli/chattui` (Bubble Tea) for TTY; legacy `chatui`+`chatrepl` behind `--cli` / plain. Shared slash logic in `internal/cli/chatcmd`. Config under `display.*`.

**Tech Stack:** Go, charmbracelet/bubbletea, bubbles, lipgloss, glamour (existing), SQLite sessions (existing).

**Spec:** `docs/superpowers/specs/2026-07-12-chattui-hermes-parity-design.md`

---

## File map

| Path | Responsibility |
|------|----------------|
| `internal/config/display.go` | `DisplayConfig`, EffectiveMode, defaults |
| `internal/cli/progress/sink.go` | `ProgressSink` interface |
| `internal/cli/chatcmd/` | Shared slash parse (`/details`, existing cmds gradually) |
| `internal/cli/chattui/` | Bubble Tea app |
| `internal/cli/chatrepl/repl.go` | Use ProgressSink; optional TUI entry handoff |
| `cmd/geegoo/chat.go` | `--tui`/`--cli`, interface selection |
| `config.example.json` | `display` block |

---

## Phase A — Collapsible TUI core

### Task A1: Display config + EffectiveMode

**Files:**
- Create: `internal/config/display.go`
- Create: `internal/config/display_test.go`
- Modify: `internal/config/config.go` (`AppConfig.Display`)
- Modify: `config.example.json`

- [ ] **Step 1: Write failing tests** for `EffectiveMode` and defaults

```go
func TestEffectiveModeFallsBackToGlobal(t *testing.T) {
	d := DisplayConfig{DetailsMode: "collapsed"}
	if got := d.EffectiveMode("thinking"); got != ModeCollapsed {
		t.Fatalf("got %s", got)
	}
}
func TestEffectiveModeSectionOverride(t *testing.T) {
	d := DisplayConfig{DetailsMode: "collapsed", Sections: DisplaySections{Thinking: "expanded"}}
	if got := d.EffectiveMode("thinking"); got != ModeExpanded {
		t.Fatalf("got %s", got)
	}
}
```

- [ ] **Step 2: Implement** `Mode` constants, `DisplayConfig`, `Normalize`, `EffectiveMode`, wire into `AppConfig`
- [ ] **Step 3: `go test ./internal/config/ -run Display -count=1`** → PASS
- [ ] **Step 4: Commit** `feat(config): add display.details_mode for chat TUI`

### Task A2: ProgressSink interface

**Files:**
- Create: `internal/cli/progress/sink.go`
- Modify: `internal/cli/chatrepl/repl.go` (`attachProgress` accept sink; `ChatUI` already matches)

- [ ] **Step 1: Define**

```go
package progress
type Sink interface {
	EmitProgress(event string, data map[string]any)
}
```

- [ ] **Step 2: Change** `attachProgress` to type-assert UI as `progress.Sink` (ChatUI already has method)
- [ ] **Step 3: Compile** `go test ./internal/cli/chatrepl/ -count=1`
- [ ] **Step 4: Commit** `refactor(cli): extract ProgressSink interface`

### Task A3: Section model + collapse rules (pure, no TUI)

**Files:**
- Create: `internal/cli/chattui/section.go`
- Create: `internal/cli/chattui/section_test.go`

- [ ] **Step 1: Types** `SectionKind`, `Block` (ID, Kind, Title, Body, Live, UserExpanded *bool, Lines, Duration)
- [ ] **Step 2: `Block.IsExpanded(cfg DisplayConfig) bool`** — live⇒true; user override; else EffectiveMode ≠ collapsed/hidden; hidden⇒false
- [ ] **Step 3: Tests** for live/history/override/hidden
- [ ] **Step 4: Commit** `feat(chattui): collapsible section model`

### Task A4: `/details` parser in chatcmd

**Files:**
- Create: `internal/cli/chatcmd/details.go`
- Create: `internal/cli/chatcmd/details_test.go`

- [ ] **Step 1: Parse** `/details`, `/details cycle`, `/details thinking expanded`, `/details tools reset`, `/details last`
- [ ] **Step 2: Apply** to `*config.DisplayConfig` (mutate + return whether persist)
- [ ] **Step 3: Tests**
- [ ] **Step 4: Commit** `feat(chatcmd): parse /details like Hermes`

### Task A5: Minimal Bubble Tea transcript + collapse

**Files:**
- Create: `internal/cli/chattui/model.go`, `update.go`, `view.go`, `progress.go`, `app.go`
- Create: `internal/cli/chattui/model_test.go` (msg apply without Run)
- Modify: `go.mod` add `bubbletea`, `bubbles`

- [ ] **Step 1: Add deps** `go get github.com/charmbracelet/bubbletea@latest github.com/charmbracelet/bubbles@latest`
- [ ] **Step 2: Model** holds `[]Block` + reply text + display cfg; `ProgressMsg` applies events
- [ ] **Step 3: View** renders `▸/▾` headers; Enter toggles focused block
- [ ] **Step 4: `App.Run`** alt-screen; quit on ctrl+c twice / `/exit` stub
- [ ] **Step 5: Unit test** ProgressMsg builds thinking block
- [ ] **Step 6: Commit** `feat(chattui): bubbletea transcript with chevron collapse`

### Task A6: Wire `geegoo chat` entry

**Files:**
- Modify: `cmd/geegoo/chat.go`
- Modify: `internal/cli/chattui/bridge.go` (construct from `*app.App`, run one session loop — initially may shell to simplified agent turn)

**Note:** Full slash parity lands in Phase B. Phase A TUI must at least: accept user line, call existing `ReActLoop`/`Agent.Run` once per submit, stream ProgressMsg, show final reply, `/details` + `/exit`.

- [ ] **Step 1: Flags** `--tui`, `--cli`; resolve interface
- [ ] **Step 2: Bridge** create session like `chatrepl.NewWithSession`, set Agent progress → tea program Send
- [ ] **Step 3: Non-TTY / plain / `--cli`** → existing `chatrepl.Run`
- [ ] **Step 4: Manual smoke** (if TTY available) or unit-test bridge selection
- [ ] **Step 5: Commit** `feat(chat): default TTY chat to Bubble Tea TUI`

### Task A7: Persist display config from `/details`

**Files:**
- Modify: `internal/cli/chattui` + reuse config save pattern from `chatrepl` `/think`

- [ ] **Step 1: On `/details` that changes mode**, write `display` into config.json (same helper as think)
- [ ] **Step 2: Test** with temp config file
- [ ] **Step 3: Commit** `feat(chattui): persist /details to config.json`

---

## Phase B — Status, overlays, composer

### Task B1: Status bar + busy indicator
### Task B2: Composer (textarea) + slash menu (bubbles)
### Task B3: Overlay help + model picker
### Task B4: Approval modal via tea (replace stdin in TUI path)
### Task B5: Map `/verbose` → details modes

*(Each task: tests for pure logic + commit; follow same TDD pattern as A.)*

---

## Phase C — Mouse, multi-session, polish

### Task C1: Mouse tracking presets + `/mouse`
### Task C2: Virtual height + live-tail follow
### Task C3: Live session switcher (`/sessions`, Ctrl+X)
### Task C4: Docs + roadmap checkbox + README TUI section
### Task C5: Full regression: `--cli` path + `go test ./...`

---

## Spec coverage check

| Spec section | Tasks |
|--------------|-------|
| Collapsible thinking/tools | A3–A5 |
| details_mode + /details | A1, A4, A7 |
| Live vs historical | A3 |
| ProgressSink | A2, A5 |
| Entry + legacy | A6 |
| Status / overlays / composer | B1–B4 |
| Mouse / multi-session | C1–C3 |
| Docs | C4 |

## Execution note

User requested immediate execution. Implement **Phase A tasks A1→A7** in this branch first; then continue B→C without waiting.
