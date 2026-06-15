package tui

import (
	"fmt"
	"os/exec"
	"runtime"

	"github.com/AirmanBugs/textercism/internal/exercism"
	"github.com/AirmanBugs/textercism/internal/sync"
	tea "github.com/charmbracelet/bubbletea"
)

// actionDoneMsg carries the result of a background action back to the model.
type actionDoneMsg struct{ status string }

// execDoneMsg is returned after a suspended command (test/submit) finishes.
type execDoneMsg struct{ status string }

// runAction dispatches the chosen action for the selected exercise. test/submit
// suspend the TUI to run in the full terminal; the rest run in the background and
// post a status line. The TUI stays up either way.
func (m *model) runAction(kind ActionKind) (tea.Model, tea.Cmd) {
	track, ex := m.track, m.selected.Slug

	switch kind {
	case ActionTest:
		return m, m.suspendTest(track, ex)
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

// --- suspended (full-terminal) commands ---

func (m *model) suspendTest(track, ex string) tea.Cmd {
	if !exercism.Downloaded(m.cfg, track, ex) {
		m.status = "Not downloaded — Start it first."
		return nil
	}
	cmd := exercism.TestCmd(m.cfg, track, ex)
	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		if err != nil {
			return execDoneMsg{status: "Tests failed."}
		}
		return execDoneMsg{status: "Tests passed."}
	})
}

func (m *model) suspendSubmit(track, ex string) tea.Cmd {
	if !exercism.Downloaded(m.cfg, track, ex) {
		m.status = "Not downloaded — Start it first."
		return nil
	}
	// Run tests first in the full terminal; submit only if they pass.
	cfg := m.cfg
	test := exercism.TestCmd(cfg, track, ex)
	return tea.ExecProcess(test, func(err error) tea.Msg {
		if err != nil {
			return execDoneMsg{status: "Tests failed — not submitted."}
		}
		if _, serr := exercism.Submit(cfg, track, ex); serr != nil {
			return execDoneMsg{status: "Submit failed: " + serr.Error()}
		}
		return execDoneMsg{status: fmt.Sprintf("Submitted %s/%s to Exercism.", track, ex)}
	})
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
