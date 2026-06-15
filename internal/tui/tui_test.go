package tui

import (
	"strings"
	"testing"

	"github.com/AirmanBugs/textercism/internal/config"
	"github.com/AirmanBugs/textercism/internal/exercism"
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
	m := &model{cfg: cfg, track: "elixir", exercises: exs, result: Action{Kind: ActionNone}}
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
	case "q":
		msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
	}
	updated, _ := m.Update(msg)
	return updated.(*model)
}

func TestExerciseToActionToResult(t *testing.T) {
	m := newTestModel()

	// View renders the exercise list without panicking.
	if v := m.View(); !strings.Contains(v, "Lasagna") {
		t.Fatalf("exercise list view missing exercise; got:\n%s", v)
	}

	// Enter on the first exercise -> action screen.
	m = send(m, "enter")
	if m.screen != screenActions {
		t.Fatalf("expected action screen, got %v", m.screen)
	}
	if m.selected.Slug != "lasagna" {
		t.Fatalf("expected lasagna selected, got %q", m.selected.Slug)
	}

	// Enter on the first action (Continue/Open for a completed+not-downloaded
	// exercise -> ActionStart) sets the result and signals quit.
	m = send(m, "enter")
	if m.result.Kind == ActionNone {
		t.Fatalf("expected an action result, got none")
	}
	if m.result.Exercise != "lasagna" || m.result.Track != "elixir" {
		t.Fatalf("unexpected result target: %+v", m.result)
	}
}

func TestBackNavigation(t *testing.T) {
	m := newTestModel()
	m = send(m, "enter") // -> actions
	if m.screen != screenActions {
		t.Fatalf("setup: expected actions screen")
	}
	m = send(m, "q") // back -> exercises
	if m.screen != screenExercises {
		t.Fatalf("expected back to exercises, got %v", m.screen)
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
	// Not started, nothing local -> first action is Start.
	items := actionsFor(exercism.DNotStarted, exercism.NotOnDisk, false)
	if got := items[0].(actionItem); got.kind != ActionStart {
		t.Fatalf("not-started first action = %v, want Start", got.kind)
	}
	// Server-started but nothing local -> Continue that downloads (ActionStart).
	items = actionsFor(exercism.DStartedServer, exercism.NotOnDisk, false)
	if got := items[0].(actionItem); got.kind != ActionStart {
		t.Fatalf("server-started/no-disk first action = %v, want Start(download)", got.kind)
	}
	// Downloaded with edits -> first action is Open (continue without re-download).
	items = actionsFor(exercism.DInProgress, exercism.OnDiskEdited, false)
	if got := items[0].(actionItem); got.kind != ActionOpen {
		t.Fatalf("downloaded first action = %v, want Open", got.kind)
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
