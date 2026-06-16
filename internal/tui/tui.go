// Package tui is the interactive Bubble Tea front-end: pick a track, browse
// exercises with status badges (arrow keys + filtering via bubbles/list), then
// choose an action. The selected action is returned to main, which runs it
// outside the alt-screen so test/editor output goes to the real terminal.
package tui

import (
	"fmt"

	"github.com/AirmanBugs/textercism/internal/config"
	"github.com/AirmanBugs/textercism/internal/exercism"
	"github.com/AirmanBugs/textercism/internal/render"
	"github.com/AirmanBugs/textercism/internal/sync"
	"github.com/AirmanBugs/textercism/internal/testresult"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ActionKind identifies an action the user can run on an exercise.
type ActionKind int

const (
	ActionStart ActionKind = iota
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
	cfg     *config.Config
	client  *exercism.Client
	backend sync.Backend

	showSync bool // whether to offer the "Pause & sync" action

	screen   screen
	list     list.Model
	viewport viewport.Model // right-hand pane on the action screen
	spinner  spinner.Model  // shown in the status line while tests run
	stacked  bool           // true when the terminal is too narrow for side-by-side
	status   string         // transient status line on the action screen

	pane           paneMode          // what the right pane shows: instructions or test output
	paneFocused    bool              // true when the right pane (not the action list) has focus
	instructions   string            // rendered instructions (cached for the selected exercise)
	testResult     testresult.Result // last parsed test result (clean view rendered from this)
	testRaw        string            // rendered raw test output (for the "r" toggle)
	showRawTest    bool              // whether the test pane shows raw output
	showAssertions bool              // whether the clean view expands assertion detail
	testRunning    bool              // a test run is in progress (drives the spinner)

	track     string
	exercises []exercism.Exercise
	selected  exercism.Exercise

	err error

	width, height int
}

// paneMode is what the action screen's right pane is showing.
type paneMode int

const (
	paneInstructions paneMode = iota
	paneTestOutput
)

// minSideBySideWidth is the terminal width below which the action screen stacks
// the instructions under the action list instead of placing them side by side.
const minSideBySideWidth = 90

// actionPaneWidth is how wide the action list is in the side-by-side layout.
const actionPaneWidth = 34

// Run launches the interactive UI starting at the track picker. If startTrack is
// non-empty it jumps straight to that track's exercises. Actions (start, test,
// submit, …) run inside the TUI: test/submit suspend to the full terminal, the
// rest run in the background and report a status line.
func Run(cfg *config.Config, backend sync.Backend, startTrack string) error {
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = statusStyle

	m := &model{
		cfg:      cfg,
		client:   exercism.NewClient(cfg),
		backend:  backend,
		showSync: backend.SyncsAcrossDevices(),
		spinner:  sp,
	}

	if startTrack != "" {
		m.track = startTrack
		if err := m.loadExercises(); err != nil {
			return err
		}
		m.screen = screenExercises
		m.list = newExerciseList(cfg, startTrack, m.exercises, 0, 0)
	} else {
		l, err := m.newTrackList()
		if err != nil {
			return err
		}
		m.list = l
		m.screen = screenTracks
	}

	prog := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())
	final, err := prog.Run()
	if err != nil {
		return err
	}
	return final.(*model).err
}

func (m *model) Init() tea.Cmd { return nil }

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.relayout()
		return m, nil

	case actionDoneMsg:
		// A background action finished: show its status and refresh the action
		// list + pane (local state may have changed, e.g. after download).
		m.status = msg.status
		if m.screen == screenActions {
			m.list = newActionList(m.cfg, m.track, m.selected, m.showSync, 0, 0)
			m.relayout()
		}
		return m, nil

	case testDoneMsg:
		// Tests finished: show the clean results in the right pane. Focus stays on
		// the action list (don't yank it away).
		m.testRunning = false
		m.status = msg.status
		m.testResult = msg.result
		m.testRaw = msg.raw
		m.showRawTest = false
		m.pane = paneTestOutput
		if m.screen == screenActions {
			m.viewport.SetContent(m.paneContent())
			m.viewport.GotoTop()
		}
		return m, nil

	case instructionsReadyMsg:
		// A background download for instructions finished. Only apply if the user
		// is still on this exercise's instructions pane.
		if m.screen == screenActions && m.selected.Slug == msg.exercise {
			if msg.status != "" {
				m.status = msg.status
			}
			m.list = newActionList(m.cfg, m.track, m.selected, m.showSync, 0, 0)
			if m.pane == paneInstructions {
				m.viewport.SetContent(m.renderInstructions(m.viewport.Width))
				m.viewport.GotoTop()
			}
			m.relayout()
		}
		return m, nil

	case spinner.TickMsg:
		// Keep the spinner animating only while a test run is in progress.
		if m.testRunning {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
		return m, nil

	case tea.MouseMsg:
		// Trackpad/wheel scrolls the right pane on the action screen.
		if m.screen == screenActions {
			var cmd tea.Cmd
			m.viewport, cmd = m.viewport.Update(msg)
			return m, cmd
		}
		return m, nil

	case tea.KeyMsg:
		// Most keypresses clear a stale status line (but not pane scrolling).
		switch msg.String() {
		case "pgup", "pgdown", "ctrl+u", "ctrl+d":
		default:
			m.status = ""
		}
		// Let the list handle keys while filtering (so typing works).
		if m.list.FilterState() == list.Filtering {
			break
		}
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "esc":
			return m.goBack()
		}

		if m.screen == screenActions {
			switch msg.String() {
			case "tab":
				// Toggle focus between the action list and the right pane.
				m.paneFocused = !m.paneFocused
				return m, nil
			case "i":
				if m.pane == paneTestOutput {
					m.pane = paneInstructions
					m.paneFocused = false
					m.viewport.SetContent(m.paneContent())
					m.viewport.GotoTop()
					return m, nil
				}
			case "r":
				// Toggle raw vs. clean test output when showing test results.
				if m.pane == paneTestOutput {
					m.showRawTest = !m.showRawTest
					m.viewport.SetContent(m.paneContent())
					m.viewport.GotoTop()
					return m, nil
				}
			case "a":
				// Toggle assertion detail in the clean test view.
				if m.pane == paneTestOutput && !m.showRawTest {
					m.showAssertions = !m.showAssertions
					m.viewport.SetContent(m.paneContent())
					m.viewport.GotoTop()
					return m, nil
				}
			case "pgup", "pgdown", "ctrl+u", "ctrl+d":
				// Page keys always scroll the pane regardless of focus.
				var cmd tea.Cmd
				m.viewport, cmd = m.viewport.Update(msg)
				return m, cmd
			}

			// When the pane is focused, arrows/j/k scroll it; otherwise they (and
			// Enter) drive the action list.
			if m.paneFocused {
				var cmd tea.Cmd
				m.viewport, cmd = m.viewport.Update(msg)
				return m, cmd
			}
		}

		if msg.String() == "enter" {
			return m.choose()
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

// relayout sizes the list and instructions viewport for the current screen and
// terminal dimensions. On the action screen it splits the width between the
// action list and the instructions pane (or stacks them when too narrow).
func (m *model) relayout() {
	bodyH := m.height - 1 // leave a row for the footer
	if bodyH < 1 {
		bodyH = 1
	}

	if m.screen != screenActions {
		m.list.SetSize(m.width, bodyH)
		return
	}

	m.stacked = m.width < minSideBySideWidth
	if m.stacked {
		listH := bodyH / 2
		m.list.SetSize(m.width, listH)
		m.viewport.Width = m.width
		m.viewport.Height = bodyH - listH
	} else {
		m.list.SetSize(actionPaneWidth, bodyH)
		m.viewport.Width = m.width - actionPaneWidth - 2 // 2 = gap
		m.viewport.Height = bodyH
	}

	// Fill the pane with whatever it's currently showing, re-wrapped to the
	// (possibly changed) width, preserving scroll position as best we can.
	off := m.viewport.YOffset
	m.viewport.SetContent(m.paneContent())
	m.viewport.SetYOffset(off)
}

// paneContent returns the rendered text for the right pane based on its mode.
func (m *model) paneContent() string {
	if m.pane == paneTestOutput {
		if m.showRawTest {
			return m.testRaw
		}
		return renderTestView(m.testResult, m.viewport.Width, m.showAssertions)
	}
	return m.renderInstructions(m.viewport.Width)
}

var (
	footerStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	statusStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("6"))
	focusBar    = lipgloss.NewStyle().Foreground(lipgloss.Color("6")) // cyan
	dimBar      = lipgloss.NewStyle().Foreground(lipgloss.Color("8")) // grey
)

func (m *model) View() string {
	if m.screen != screenActions {
		return m.list.View()
	}

	// Footer reflects what the keys do given the current focus and pane.
	var hint string
	if m.paneFocused {
		hint = "↑/↓ scroll · tab actions · esc back"
	} else {
		hint = "↑/↓ select · enter run · tab scroll pane · esc back"
	}
	if m.pane == paneTestOutput {
		raw := "r raw"
		if m.showRawTest {
			raw = "r clean"
		}
		paneHints := "i instructions · " + raw
		if !m.showRawTest {
			assert := "a assertions"
			if m.showAssertions {
				assert = "a hide"
			}
			paneHints += " · " + assert
		}
		hint = paneHints + " · " + hint
	}
	line := footerStyle.Render(hint)
	switch {
	case m.testRunning:
		line = m.spinner.View() + statusStyle.Render("Running tests…") + "  " + line
	case m.status != "":
		line = statusStyle.Render(m.status) + "  " + line
	}

	if m.stacked {
		return m.list.View() + "\n" + m.viewport.View() + "\n" + line
	}

	// A thin vertical bar between the panes, colored to show which has focus.
	bar := m.focusGutter()
	body := lipgloss.JoinHorizontal(lipgloss.Top, m.list.View(), bar, m.viewport.View())
	return body + "\n" + line
}

// focusGutter is the column between the action list and the pane; it's cyan on
// the side that currently has focus, so it's obvious which arrows will move.
func (m *model) focusGutter() string {
	h := m.viewport.Height
	if h < 1 {
		h = 1
	}
	left, right := focusBar, dimBar
	if m.paneFocused {
		left, right = dimBar, focusBar
	}
	rows := make([]string, h)
	for i := range rows {
		rows[i] = left.Render("│") + right.Render("│")
	}
	return lipgloss.JoinVertical(lipgloss.Left, rows...)
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
		return m, m.enterActions(it.ex)

	case screenActions:
		it, ok := m.list.SelectedItem().(actionItem)
		if !ok {
			return m, nil
		}
		return m.runAction(it.kind)
	}
	return m, nil
}

// enterActions switches to the action screen for ex. If the exercise isn't
// downloaded yet, it kicks off a download (returning a command) and shows a
// loading message until the real instructions are available.
func (m *model) enterActions(ex exercism.Exercise) tea.Cmd {
	m.selected = ex
	m.status = ""
	m.pane = paneInstructions
	m.paneFocused = false
	m.testResult = testresult.Result{}
	m.testRaw = ""
	m.showRawTest = false
	m.list = newActionList(m.cfg, m.track, ex, m.showSync, m.width, m.height-1)
	m.viewport = viewport.New(m.width, m.height)
	m.screen = screenActions

	if exercism.Downloaded(m.cfg, m.track, ex.Slug) {
		m.instructions = "" // rendered lazily in relayout at the right width
		m.relayout()
		return nil
	}

	// Not on disk: show a loading note and download so the full README appears.
	m.instructions = ""
	m.relayout()
	m.viewport.SetContent(render.Markdown("# "+ex.Title+"\n\n_Loading instructions…_", m.viewport.Width))
	return m.downloadForInstructions(m.track, ex.Slug)
}

// renderInstructions returns the rendered README for the selected exercise, or a
// blurb + hint when it isn't downloaded yet.
func (m *model) renderInstructions(width int) string {
	if text, ok := render.Instructions(m.cfg, m.track, m.selected.Slug, width); ok {
		return text
	}
	blurb := m.selected.Blurb
	if blurb == "" {
		blurb = "_No description available._"
	}
	return render.Markdown(
		fmt.Sprintf("# %s\n\n%s\n\n_Not downloaded yet — choose **Start** to get the full instructions._",
			m.selected.Title, blurb), width)
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
	// We drive navigation/quit ourselves (Esc to go back), so disable the list's
	// own q/esc quit binding and its mention in help.
	l.KeyMap.Quit = key.NewBinding()
	l.DisableQuitKeybindings()
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
