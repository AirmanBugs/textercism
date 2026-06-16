package tui

import (
	"fmt"
	"os/exec"
	"regexp"
	"runtime"
	"strings"

	"github.com/AirmanBugs/textercism/internal/exercism"
	"github.com/AirmanBugs/textercism/internal/render"
	"github.com/AirmanBugs/textercism/internal/sync"
	"github.com/AirmanBugs/textercism/internal/testresult"
	tea "github.com/charmbracelet/bubbletea"
)

// actionDoneMsg carries the result of a background action back to the model.
type actionDoneMsg struct{ status string }

// testDoneMsg carries parsed test results to show in the right pane.
type testDoneMsg struct {
	status string            // footer summary, e.g. "1 of 11 passed"
	result testresult.Result // parsed result (the clean view is rendered from this)
	raw    string            // raw, lightly cleaned output
	width  int
}

// instructionsReadyMsg signals a background download finished so the real
// instructions can be rendered into the pane.
type instructionsReadyMsg struct {
	exercise string
	status   string
}

// runAction dispatches the chosen action for the selected exercise. Tests run in
// the background and show their output in the right pane; the rest run in the
// background and post a status line. The TUI stays up throughout.
func (m *model) runAction(kind ActionKind) (tea.Model, tea.Cmd) {
	track, ex := m.track, m.selected.Slug

	switch kind {
	case ActionInstructions, ActionHints:
		// View-only items: the pane already shows them. Enter focuses the pane so
		// you can scroll/reveal without reaching for Tab.
		m.paneFocused = true
		return m, nil
	case ActionTest:
		if !exercism.Downloaded(m.cfg, track, ex) {
			m.status = "Not downloaded — Start it first."
			return m, nil
		}
		m.testRunning = true
		return m, tea.Batch(m.spinner.Tick, m.testCmd(track, ex))
	case ActionSubmit:
		return m, m.suspendSubmit(track, ex)

	case ActionStart, ActionRestart, ActionOpen:
		m.status = "Working…"
		return m, m.openCmd(track, ex, kind == ActionRestart)
	case ActionWeb:
		m.status = "Opening browser…"
		return m, m.webCmd(track, ex)
	case ActionPause:
		m.status = "Syncing…"
		return m, m.pauseCmd(track, ex)
	}
	return m, nil
}

// testCmd runs the exercise's tests in the background, parses the output, and
// returns the parsed result plus the raw output for the pane.
func (m *model) testCmd(track, ex string) tea.Cmd {
	cfg, width := m.cfg, m.viewport.Width
	return func() tea.Msg {
		out, _ := exercism.TestCmdTrace(cfg, track, ex).CombinedOutput()
		res := testresult.Parse(string(out))
		raw := render.Markdown("```\n"+cleanRaw(string(out))+"\n```\n", width)
		return testDoneMsg{status: res.Summary(), result: res, raw: raw, width: width}
	}
}

// cleanRaw strips the worst of the noise (compile warnings and the ExUnit
// preamble) from raw test output for the "show raw" view, while keeping the rest
// intact.
func cleanRaw(s string) string {
	var keep []string
	skipWarning := false
	for _, line := range strings.Split(s, "\n") {
		t := strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(t, "warning:"):
			skipWarning = true
			continue
		case skipWarning && (t == "" || strings.HasPrefix(t, "│") ||
			strings.HasPrefix(t, "└─") || strings.HasPrefix(t, "~~~") ||
			regexpDigitsBar.MatchString(t) || strings.HasPrefix(t, "typing violation") ||
			strings.HasPrefix(t, "While Elixir") || strings.HasPrefix(t, "given types") ||
			strings.HasPrefix(t, "where ") || strings.HasPrefix(t, "# type:") ||
			strings.HasPrefix(t, "# from:") || strings.HasPrefix(t, "left =") ||
			strings.HasPrefix(t, "right =") || strings.HasPrefix(t, "left ==")):
			continue
		default:
			skipWarning = false
			keep = append(keep, line)
		}
	}
	return strings.TrimSpace(strings.Join(keep, "\n"))
}

var regexpDigitsBar = regexp.MustCompile(`^\d+\s*│`)

// suspendSubmit runs tests then submits, in the full terminal (submit is a
// deliberate, infrequent action where seeing full output matters).
func (m *model) suspendSubmit(track, ex string) tea.Cmd {
	if !exercism.Downloaded(m.cfg, track, ex) {
		m.status = "Not downloaded — Start it first."
		return nil
	}
	cfg := m.cfg
	test := exercism.TestCmd(cfg, track, ex)
	return tea.ExecProcess(test, func(err error) tea.Msg {
		if err != nil {
			return actionDoneMsg{status: "Tests failed — not submitted."}
		}
		if _, serr := exercism.Submit(cfg, track, ex); serr != nil {
			return actionDoneMsg{status: "Submit failed: " + serr.Error()}
		}
		return actionDoneMsg{status: fmt.Sprintf("Submitted %s/%s to Exercism.", track, ex)}
	})
}

// downloadForInstructions downloads the exercise (so its README exists locally)
// purely to populate the instructions pane. Returns a message the model uses to
// re-render the pane.
func (m *model) downloadForInstructions(track, ex string) tea.Cmd {
	cfg, backend := m.cfg, m.backend
	return func() tea.Msg {
		if !exercism.Downloaded(cfg, track, ex) {
			ref := sync.DraftRef{Track: track, Exercise: ex}
			_ = backend.Pull(ref, exercism.ExerciseDir(cfg, track, ex))
		}
		if !exercism.Downloaded(cfg, track, ex) {
			if _, err := exercism.Download(cfg, track, ex); err != nil {
				return instructionsReadyMsg{exercise: ex, status: "Could not load instructions: " + err.Error()}
			}
		}
		return instructionsReadyMsg{exercise: ex}
	}
}

// --- background commands ---

// openCmd downloads (if needed) and opens the exercise in VS Code without leaving
// the TUI. force re-downloads the stub.
func (m *model) openCmd(track, ex string, force bool) tea.Cmd {
	cfg, backend := m.cfg, m.backend
	return func() tea.Msg {
		if !exercism.Downloaded(cfg, track, ex) {
			// Try a draft from the sync backend before downloading a fresh stub.
			ref := sync.DraftRef{Track: track, Exercise: ex}
			_ = backend.Pull(ref, exercism.ExerciseDir(cfg, track, ex))
		}
		if !exercism.Downloaded(cfg, track, ex) || force {
			if _, err := exercism.Download(cfg, track, ex); err != nil {
				return actionDoneMsg{status: "Download failed: " + err.Error()}
			}
		}
		if err := exercism.OpenVSCode(cfg, track, ex); err != nil {
			if err == exercism.ErrCodeNotFound {
				return actionDoneMsg{status: "code not on PATH: " + exercism.ExerciseDir(cfg, track, ex)}
			}
			return actionDoneMsg{status: "Open failed: " + err.Error()}
		}
		return actionDoneMsg{status: fmt.Sprintf("Opened %s/%s in VS Code.", track, ex)}
	}
}

func (m *model) webCmd(track, ex string) tea.Cmd {
	return func() tea.Msg {
		openURL(fmt.Sprintf("https://exercism.org/tracks/%s/exercises/%s", track, ex))
		return actionDoneMsg{status: "Opened in browser."}
	}
}

// openHintDocs opens the unique doc URLs referenced by the currently-revealed
// hints (so unrevealed hints' links aren't spoiled).
func (m *model) openHintDocs() tea.Cmd {
	var urls []string
	seen := map[string]bool{}
	for i := 0; i < m.hintsShown && i < len(m.hints.Items); i++ {
		for _, u := range m.hints.Items[i].Links {
			if !seen[u] {
				seen[u] = true
				urls = append(urls, u)
			}
		}
	}
	if len(urls) == 0 {
		return func() tea.Msg { return actionDoneMsg{status: "No doc links in the revealed hints yet."} }
	}
	return func() tea.Msg {
		for _, u := range urls {
			openURL(u)
		}
		return actionDoneMsg{status: fmt.Sprintf("Opened %d doc link(s).", len(urls))}
	}
}

func (m *model) pauseCmd(track, ex string) tea.Cmd {
	cfg, backend := m.cfg, m.backend
	return func() tea.Msg {
		if !backend.SyncsAcrossDevices() {
			return actionDoneMsg{status: "No sync backend configured."}
		}
		ref := sync.DraftRef{Track: track, Exercise: ex}
		if err := backend.Push(ref, exercism.ExerciseDir(cfg, track, ex)); err != nil {
			return actionDoneMsg{status: "Sync failed: " + err.Error()}
		}
		return actionDoneMsg{status: "Draft synced."}
	}
}

func openURL(url string) {
	cmd := "xdg-open"
	if runtime.GOOS == "darwin" {
		cmd = "open"
	}
	if _, err := exec.LookPath(cmd); err == nil {
		_ = exec.Command(cmd, url).Run()
	}
}
