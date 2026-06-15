// Package tui is the interactive Bubble Tea front-end: pick a track, browse
// exercises with status badges (arrow keys + filtering via bubbles/list), then
// choose an action. The selected action is returned to main, which runs it
// outside the alt-screen so test/editor output goes to the real terminal.
package tui

import (
	"fmt"

	"github.com/AirmanBugs/textercism/internal/config"
	"github.com/AirmanBugs/textercism/internal/exercism"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Action is what the user chose to do with an exercise.
type Action struct {
	Kind     ActionKind
	Track    string
	Exercise string
}

type ActionKind int

const (
	ActionNone ActionKind = iota
	ActionStart
	ActionRestart
	ActionOpen
	ActionTest
	ActionSubmit
	ActionPause
	ActionWeb
)

type screen int

const (
	screenTracks screen = iota
	screenExercises
	screenActions
)

type model struct {
	cfg    *config.Config
	client *exercism.Client

	showSync bool // whether to offer the "Pause & sync" action

	screen screen
	list   list.Model

	track     string
	exercises []exercism.Exercise
	selected  exercism.Exercise

	result Action
	err    error

	width, height int
}

// Run launches the interactive UI starting at the track picker. If startTrack is
// non-empty it jumps straight to that track's exercises. showSync controls
// whether the "Pause & sync" action is offered (off for the local-only backend).
// Returns the chosen Action (Kind ActionNone if the user quit without choosing).
func Run(cfg *config.Config, startTrack string, showSync bool) (Action, error) {
	m := &model{
		cfg:      cfg,
		client:   exercism.NewClient(cfg),
		showSync: showSync,
		result:   Action{Kind: ActionNone},
	}

	if startTrack != "" {
		m.track = startTrack
		if err := m.loadExercises(); err != nil {
			return m.result, err
		}
		m.screen = screenExercises
		m.list = newExerciseList(cfg, startTrack, m.exercises, 0, 0)
	} else {
		l, err := m.newTrackList()
		if err != nil {
			return m.result, err
		}
		m.list = l
		m.screen = screenTracks
	}

	prog := tea.NewProgram(m, tea.WithAltScreen())
	final, err := prog.Run()
	if err != nil {
		return Action{Kind: ActionNone}, err
	}
	fm := final.(*model)
	if fm.err != nil {
		return Action{Kind: ActionNone}, fm.err
	}
	return fm.result, nil
}

func (m *model) Init() tea.Cmd { return nil }

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.list.SetSize(msg.Width, msg.Height-1)
		return m, nil

	case tea.KeyMsg:
		// Let the list handle keys while filtering (so typing works).
		if m.list.FilterState() == list.Filtering {
			break
		}
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "q", "esc":
			return m.goBack()
		case "enter":
			return m.choose()
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m *model) View() string {
	return m.list.View()
}

// --- transitions ---

func (m *model) goBack() (tea.Model, tea.Cmd) {
	switch m.screen {
	case screenTracks:
		return m, tea.Quit
	case screenExercises:
		l, err := m.newTrackList()
		if err != nil {
			m.err = err
			return m, tea.Quit
		}
		m.list = l
		m.list.SetSize(m.width, m.height-1)
		m.screen = screenTracks
		return m, nil
	case screenActions:
		m.list = newExerciseList(m.cfg, m.track, m.exercises, m.width, m.height-1)
		m.screen = screenExercises
		return m, nil
	}
	return m, nil
}

func (m *model) choose() (tea.Model, tea.Cmd) {
	switch m.screen {
	case screenTracks:
		it, ok := m.list.SelectedItem().(trackItem)
		if !ok {
			return m, nil
		}
		m.track = it.slug
		if err := m.loadExercises(); err != nil {
			m.err = err
			return m, tea.Quit
		}
		m.list = newExerciseList(m.cfg, m.track, m.exercises, m.width, m.height-1)
		m.screen = screenExercises
		return m, nil

	case screenExercises:
		it, ok := m.list.SelectedItem().(exerciseItem)
		if !ok {
			return m, nil
		}
		m.selected = it.ex
		m.list = newActionList(m.cfg, m.track, it.ex, m.showSync, m.width, m.height-1)
		m.screen = screenActions
		return m, nil

	case screenActions:
		it, ok := m.list.SelectedItem().(actionItem)
		if !ok {
			return m, nil
		}
		m.result = Action{Kind: it.kind, Track: m.track, Exercise: m.selected.Slug}
		return m, tea.Quit
	}
	return m, nil
}

func (m *model) loadExercises() error {
	exs, err := m.client.Exercises(m.track)
	if err != nil {
		return fmt.Errorf("could not fetch exercises: %w", err)
	}
	m.exercises = exs
	return nil
}

func (m *model) newTrackList() (list.Model, error) {
	tracks, err := m.client.Tracks()
	if err != nil {
		return list.Model{}, fmt.Errorf("could not fetch tracks: %w", err)
	}
	items := make([]list.Item, 0, len(tracks))
	for _, t := range tracks {
		items = append(items, trackItem{
			slug:   t.Slug,
			title:  t.Title,
			joined: t.IsJoined,
			done:   t.NumCompletedExercises,
			total:  t.NumExercises,
		})
	}
	l := newList(items, "Select a track", m.width, m.height)
	return l, nil
}

// --- list construction ---

func newList(items []list.Item, title string, w, h int) list.Model {
	delegate := list.NewDefaultDelegate()
	l := list.New(items, delegate, w, h)
	l.Title = title
	l.Styles.Title = lipgloss.NewStyle().
		Foreground(lipgloss.Color("6")).Bold(true)
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)
	if w > 0 && h > 0 {
		l.SetSize(w, h)
	}
	return l
}

func newExerciseList(cfg *config.Config, track string, exs []exercism.Exercise, w, h int) list.Model {
	// Show locked exercises too, but they aren't selectable into actions.
	items := make([]list.Item, 0, len(exs))
	for _, e := range exs {
		local := exercism.LocalStateOf(cfg, track, e.Slug)
		items = append(items, exerciseItem{
			ex:      e,
			local:   local,
			display: exercism.Display(e.Status, local),
		})
	}
	l := newList(items, fmt.Sprintf("%s — exercises", track), w, h)
	return l
}

func newActionList(cfg *config.Config, track string, ex exercism.Exercise, showSync bool, w, h int) list.Model {
	local := exercism.LocalStateOf(cfg, track, ex.Slug)
	display := exercism.Display(ex.Status, local)
	items := actionsFor(display, local, showSync)
	title := fmt.Sprintf("%s %s  (%s)", display.Badge(), ex.Title, display.Label())
	l := newList(items, title, w, h)
	l.SetFilteringEnabled(false)
	return l
}

func actionsFor(display exercism.DisplayStatus, local exercism.LocalState, showSync bool) []list.Item {
	var items []list.Item
	switch {
	case local == exercism.NotOnDisk && display == exercism.DNotStarted:
		items = append(items, actionItem{"Start", "Download + open VS Code", ActionStart})
	case local == exercism.NotOnDisk:
		// Server-started but nothing local: continue downloads the stub.
		items = append(items, actionItem{"Continue", "Download stub + open VS Code", ActionStart})
	default:
		items = append(items, actionItem{"Continue", "Open in VS Code", ActionOpen})
	}

	items = append(items,
		actionItem{"Run tests", "Run the exercise's tests", ActionTest},
		actionItem{"Submit", "Test, then submit to Exercism", ActionSubmit},
	)
	if showSync {
		items = append(items, actionItem{"Pause & sync", "Save draft to your sync backend", ActionPause})
	}
	items = append(items,
		actionItem{"Restart", "Re-download stub (overwrites)", ActionRestart},
		actionItem{"Open in browser", "Open the exercise/solution page", ActionWeb},
	)
	return items
}
