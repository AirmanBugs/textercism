// Package actions composes the lower-level exercism operations into the
// user-facing flows shared by the CLI and the TUI, printing progress as it goes.
package actions

import (
	"errors"
	"fmt"
	"os/exec"
	"runtime"

	"github.com/AirmanBugs/textercism/internal/config"
	"github.com/AirmanBugs/textercism/internal/exercism"
	"github.com/AirmanBugs/textercism/internal/render"
	"github.com/AirmanBugs/textercism/internal/sync"
	"github.com/charmbracelet/lipgloss"
)

var (
	okStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	infoStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("4"))
	errStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
)

func ok(msg string)   { fmt.Println(okStyle.Render("✔ ") + msg) }
func info(msg string) { fmt.Println(infoStyle.Render("• ") + msg) }
func fail(msg string) { fmt.Println(errStyle.Render("✘ ") + msg) }

// ConfirmFunc asks a yes/no question and returns the answer.
type ConfirmFunc func(question string) bool

// AutoYes is the default confirmer for non-interactive paths.
func AutoYes(string) bool { return true }

// Start downloads (if needed) and opens an exercise in VS Code. force re-downloads stubs.
func Start(cfg *config.Config, backend sync.Backend, track, exercise string, force bool) {
	if exercism.Downloaded(cfg, track, exercise) && !force {
		info("Already downloaded; continuing.")
		openEditor(cfg, track, exercise)
		return
	}
	downloadAndOpen(cfg, track, exercise)
}

// Open continues an exercise: if it's not on this machine, try the sync backend
// to recover a draft from another device, otherwise download a fresh stub.
func Open(cfg *config.Config, backend sync.Backend, track, exercise string) {
	if exercism.Downloaded(cfg, track, exercise) {
		openEditor(cfg, track, exercise)
		return
	}

	ref := sync.DraftRef{Track: track, Exercise: exercise}
	err := backend.Pull(ref, exercism.ExerciseDir(cfg, track, exercise))
	switch {
	case err == nil:
		ok("Recovered draft from " + backend.Name() + " sync.")
		openEditor(cfg, track, exercise)
	case errors.Is(err, sync.ErrNoDraft):
		info("Nothing to continue — downloading the stub.")
		downloadAndOpen(cfg, track, exercise)
	default:
		fail("Sync pull failed: " + err.Error())
	}
}

func downloadAndOpen(cfg *config.Config, track, exercise string) {
	info(fmt.Sprintf("Downloading %s/%s ...", track, exercise))
	dir, err := exercism.Download(cfg, track, exercise)
	if err != nil {
		fail(err.Error())
		return
	}
	ok("Downloaded to " + dir)
	openEditor(cfg, track, exercise)
}

// Pause saves the exercise's current draft to the sync backend so it can be
// resumed on another device. With the local-only backend there's nothing to
// sync to, so callers should hide this action (see backend.SyncsAcrossDevices).
func Pause(cfg *config.Config, backend sync.Backend, track, exercise string) {
	if !backend.SyncsAcrossDevices() {
		info("No sync backend configured — drafts stay on this machine.")
		return
	}
	if !exercism.Downloaded(cfg, track, exercise) {
		fail("Exercise not downloaded; nothing to sync.")
		return
	}
	info("Syncing draft via " + backend.Name() + " ...")
	ref := sync.DraftRef{Track: track, Exercise: exercise}
	if err := backend.Push(ref, exercism.ExerciseDir(cfg, track, exercise)); err != nil {
		fail("Sync push failed: " + err.Error())
		return
	}
	ok(fmt.Sprintf("Draft synced: %s/%s", track, exercise))
}

func openEditor(cfg *config.Config, track, exercise string) {
	err := exercism.OpenVSCode(cfg, track, exercise)
	switch {
	case err == nil:
		ok(fmt.Sprintf("Opened %s/%s in VS Code.", track, exercise))
	case err == exercism.ErrCodeNotFound:
		info("`code` not on PATH. Open manually: " + exercism.ExerciseDir(cfg, track, exercise))
	default:
		fail(err.Error())
	}

	// Instructions are read in-terminal now (the TUI's Instructions screen, or the
	// `read` command), so we no longer auto-open the browser. The `web` command
	// still opens exercism.org explicitly.
}

// Read renders the exercise's instructions (README) to the terminal. Use it
// non-interactively, e.g. `textercism read elixir two-fer | less`.
func Read(cfg *config.Config, track, exercise string, width int) {
	text, ok := render.Instructions(cfg, track, exercise, width)
	if !ok {
		fail("Exercise not downloaded — start it first to get the instructions.")
		return
	}
	fmt.Print(text)
}

// Test runs the exercise's tests, streaming output.
func Test(cfg *config.Config, track, exercise string) {
	if !exercism.Downloaded(cfg, track, exercise) {
		fail("Exercise not downloaded. Start it first.")
		return
	}
	info(fmt.Sprintf("Running tests for %s/%s ...", track, exercise))
	if err := exercism.Test(cfg, track, exercise); err != nil {
		fail("Tests failed.")
		return
	}
	ok("Tests passed.")
}

// Submit runs tests then submits to Exercism. Completed work persists on
// Exercism (the source of truth) — textercism keeps no separate copy.
func Submit(cfg *config.Config, track, exercise string, confirm ConfirmFunc) {
	if !exercism.Downloaded(cfg, track, exercise) {
		fail("Exercise not downloaded. Start it first.")
		return
	}

	info("Running tests before submit ...")
	if err := exercism.Test(cfg, track, exercise); err != nil {
		fail("Tests failed. Fix before submitting.")
		return
	}
	ok("Tests passed. Submitting ...")

	out, err := exercism.Submit(cfg, track, exercise)
	if err != nil {
		fail(err.Error())
		return
	}
	if out != "" {
		ok(out)
	}
	ok(fmt.Sprintf("Submitted %s/%s to Exercism.", track, exercise))
}

// Web opens the exercise's web page in the browser.
func Web(cfg *config.Config, track, exercise string) {
	client := exercism.NewClient(cfg)
	exs, err := client.Exercises(track)
	if err != nil {
		fail("Could not fetch exercise: " + err.Error())
		return
	}
	for _, e := range exs {
		if e.Slug == exercise {
			OpenURL(e.WebURL)
			return
		}
	}
	fail(fmt.Sprintf("Unknown exercise %s in %s.", exercise, track))
}

// OpenURL opens a URL in the default browser.
func OpenURL(url string) {
	var cmd string
	switch runtime.GOOS {
	case "darwin":
		cmd = "open"
	default:
		cmd = "xdg-open"
	}
	if _, err := exec.LookPath(cmd); err != nil {
		info("Open manually: " + url)
		return
	}
	_ = exec.Command(cmd, url).Run()
	ok("Opened " + url)
}
