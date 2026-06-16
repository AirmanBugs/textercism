package tui

import (
	"strings"
	"testing"

	"github.com/AirmanBugs/textercism/internal/config"
	"github.com/AirmanBugs/textercism/internal/exercism"
	"github.com/AirmanBugs/textercism/internal/sync"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

// newTestModel builds a model seeded with exercises on the exercises screen,
// bypassing any network calls.
func newTestModel() *model {
	cfg := &config.Config{Workspace: "/tmp/does-not-exist"}
	exs := []exercism.Exercise{
		{Slug: "lasagna", Title: "Lasagna", Difficulty: "easy", Status: exercism.Completed, IsUnlocked: true},
		{Slug: "two-fer", Title: "Two Fer", Difficulty: "easy", Status: exercism.NotStarted, IsUnlocked: true},
	}
	m := &model{cfg: cfg, backend: sync.NewLocal(cfg), track: "elixir", exercises: exs}
	m.list = newExerciseList(cfg, "elixir", exs, 80, 24)
	m.screen = screenExercises
	return m
}

func send(m *model, key string) *model {
	var msg tea.Msg
	switch key {
	case "enter":
		msg = tea.KeyMsg{Type: tea.KeyEnter}
	case "down":
		msg = tea.KeyMsg{Type: tea.KeyDown}
	case "esc":
		msg = tea.KeyMsg{Type: tea.KeyEsc}
	case "tab":
		msg = tea.KeyMsg{Type: tea.KeyTab}
	}
	updated, _ := m.Update(msg)
	return updated.(*model)
}

func TestExerciseToAction(t *testing.T) {
	m := newTestModel()

	// View renders the exercise list without panicking.
	if v := m.View(); !strings.Contains(v, "Lasagna") {
		t.Fatalf("exercise list view missing exercise; got:\n%s", v)
	}

	// Enter on the first exercise -> action screen with that exercise selected.
	m = send(m, "enter")
	if m.screen != screenActions {
		t.Fatalf("expected action screen, got %v", m.screen)
	}
	if m.selected.Slug != "lasagna" {
		t.Fatalf("expected lasagna selected, got %q", m.selected.Slug)
	}

	// The first item is the Instructions view; Enter on it focuses the pane and
	// stays on the action screen (no command, doesn't quit).
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(*model)
	if m.screen != screenActions {
		t.Fatalf("expected to stay on action screen, got %v", m.screen)
	}
	if !m.paneFocused {
		t.Fatalf("expected Enter on Instructions to focus the pane")
	}
}

func TestBackNavigation(t *testing.T) {
	m := newTestModel()
	m = send(m, "enter") // -> actions
	if m.screen != screenActions {
		t.Fatalf("setup: expected actions screen")
	}
	m = send(m, "esc") // back -> exercises
	if m.screen != screenExercises {
		t.Fatalf("expected back to exercises, got %v", m.screen)
	}
}

func sizeMsg(m *model, w, h int) *model {
	updated, _ := m.Update(tea.WindowSizeMsg{Width: w, Height: h})
	return updated.(*model)
}

func TestTabTogglesPaneFocus(t *testing.T) {
	m := newTestModel()
	m = sizeMsg(m, 120, 30)
	m = send(m, "enter") // -> action screen, list focused
	if m.paneFocused {
		t.Fatalf("expected list focused on entering action screen")
	}
	m = send(m, "tab")
	if !m.paneFocused {
		t.Fatalf("expected pane focused after tab")
	}
	m = send(m, "tab")
	if m.paneFocused {
		t.Fatalf("expected focus back on the list after second tab")
	}
}

func TestActionScreenLayout(t *testing.T) {
	// Wide terminal -> side by side; the action view contains both the action
	// list (e.g. "Submit") and the rendered instructions pane.
	m := newTestModel()
	m = sizeMsg(m, 120, 30)
	m = send(m, "enter") // -> actions for lasagna
	if m.stacked {
		t.Fatalf("expected side-by-side at width 120, got stacked")
	}
	v := m.View()
	if !strings.Contains(v, "Submit") {
		t.Fatalf("action view missing the action list; got:\n%s", v)
	}
	// The instructions pane renders the blurb fallback (exercise isn't on disk),
	// so the title should appear somewhere in the joined view.
	if !strings.Contains(v, "Lasagna") {
		t.Fatalf("action view missing instructions pane content; got:\n%s", v)
	}

	// Narrow terminal -> stacked.
	m2 := newTestModel()
	m2 = sizeMsg(m2, 70, 30)
	m2 = send(m2, "enter")
	if !m2.stacked {
		t.Fatalf("expected stacked at width 70, got side-by-side")
	}
}

func hasAction(items []list.Item, kind ActionKind) bool {
	for _, it := range items {
		if it.(actionItem).kind == kind {
			return true
		}
	}
	return false
}

func TestActionsForStatus(t *testing.T) {
	// View items (Instructions, Hints, Run tests) always lead the list.
	items := actionsFor(exercism.DNotStarted, exercism.NotOnDisk, false)
	for i, want := range []ActionKind{ActionInstructions, ActionHints, ActionTest} {
		if got := items[i].(actionItem); got.kind != want {
			t.Fatalf("item %d = %v, want %v", i, got.kind, want)
		}
	}

	// Not started / server-started with nothing local -> a Start action.
	if !hasAction(actionsFor(exercism.DNotStarted, exercism.NotOnDisk, false), ActionStart) {
		t.Fatal("not-started should offer Start")
	}
	if !hasAction(actionsFor(exercism.DStartedServer, exercism.NotOnDisk, false), ActionStart) {
		t.Fatal("server-started/no-disk should offer Start (download)")
	}
	// Downloaded -> Open (continue without re-download).
	if !hasAction(actionsFor(exercism.DInProgress, exercism.OnDiskEdited, false), ActionOpen) {
		t.Fatal("downloaded should offer Open")
	}
}

func TestPauseActionGatedOnSync(t *testing.T) {
	// Local-only (showSync=false): no Pause action offered.
	noSync := actionsFor(exercism.DInProgress, exercism.OnDiskEdited, false)
	if hasAction(noSync, ActionPause) {
		t.Fatalf("Pause should be hidden when no sync backend is configured")
	}
	// With a sync backend (showSync=true): Pause is offered.
	withSync := actionsFor(exercism.DInProgress, exercism.OnDiskEdited, true)
	if !hasAction(withSync, ActionPause) {
		t.Fatalf("Pause should be offered when a sync backend is configured")
	}
}

// Ensure the list delegate yields the expected item types.
var _ list.Item = exerciseItem{}
var _ list.Item = trackItem{}
var _ list.Item = actionItem{}
