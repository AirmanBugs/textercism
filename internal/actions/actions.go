// Package actions composes the lower-level exercism operations into the
// user-facing flows shared by the CLI and the TUI, printing progress as it goes.
package actions

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"

	"github.com/AirmanBugs/exercism/xrc/internal/config"
	"github.com/AirmanBugs/exercism/xrc/internal/exercism"
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
func Start(cfg *config.Config, track, exercise string, force bool) {
	if !syncBeforeWork(cfg) {
		return
	}

	if exercism.Downloaded(cfg, track, exercise) && !force {
		info("Already downloaded; continuing.")
		openEditor(cfg, track, exercise)
		return
	}

	info(fmt.Sprintf("Downloading %s/%s ...", track, exercise))
	dir, err := exercism.Download(cfg, track, exercise)
	if err != nil {
		fail(err.Error())
		return
	}
	ok("Downloaded to " + dir)
	openEditor(cfg, track, exercise)
}

// Open continues an exercise: pull first (recover WIP synced from another
// device), open it if present, otherwise download the stub. This is what makes
// "continue" work for a server-started exercise that isn't on this machine.
func Open(cfg *config.Config, track, exercise string) {
	if exercism.Downloaded(cfg, track, exercise) {
		openEditor(cfg, track, exercise)
		return
	}
	// Not on disk: it may be a WIP committed on another device, or a fresh
	// server-started exercise. Sync, then re-check before falling back to download.
	if !syncBeforeWork(cfg) {
		return
	}
	if exercism.Downloaded(cfg, track, exercise) {
		ok("Recovered work-in-progress from sync.")
		openEditor(cfg, track, exercise)
		return
	}
	info("Nothing local to continue — downloading the stub.")
	dir, err := exercism.Download(cfg, track, exercise)
	if err != nil {
		fail(err.Error())
		return
	}
	ok("Downloaded to " + dir)
	openEditor(cfg, track, exercise)
}

// Pause commits the exercise's work-in-progress and pushes it, so it can be
// resumed on another device.
func Pause(cfg *config.Config, track, exercise string) {
	if !exercism.Downloaded(cfg, track, exercise) {
		fail("Exercise not downloaded; nothing to pause.")
		return
	}
	info("Saving work-in-progress ...")
	res, err := exercism.WipCommitAndPush(cfg, track, exercise)
	if err != nil {
		fail(err.Error())
		return
	}
	switch res {
	case exercism.CommitOK:
		ok(fmt.Sprintf("Synced WIP: %s: wip %s", track, exercise))
	case exercism.CommitNoChanges:
		info("No changes to sync.")
	}
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

	// Show the instructions in the browser, where they render properly and can sit
	// beside the editor window (VS Code's CLI can't arrange a README pane).
	OpenURL(instructionsURL(track, exercise))
}

// instructionsURL is the exercise's instructions page on exercism.org. The URL
// is deterministic, so no API call is needed.
func instructionsURL(track, exercise string) string {
	return fmt.Sprintf("https://exercism.org/tracks/%s/exercises/%s", track, exercise)
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

// Submit tests, submits, then commits + pushes. confirm gates resubmit/commit.
func Submit(cfg *config.Config, track, exercise string, confirm ConfirmFunc) {
	if !exercism.Downloaded(cfg, track, exercise) {
		fail("Exercise not downloaded. Start it first.")
		return
	}
	if exercism.AlreadyCompleted(cfg, track, exercise) &&
		!confirm("This exercise was completed before. Resubmit?") {
		info("Aborted.")
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
	commit(cfg, track, exercise, confirm)
}

func commit(cfg *config.Config, track, exercise string, confirm ConfirmFunc) {
	info("Committing to git ...")
	res, err := exercism.CommitAndPush(cfg, track, exercise, func(stat string) bool {
		fmt.Println("\nChanges to commit:\n" + stat)
		return confirm("Commit and push these changes?")
	})
	if err != nil {
		fail(err.Error())
		return
	}
	switch res {
	case exercism.CommitOK:
		ok(fmt.Sprintf("Committed and pushed: %s: complete %s", track, exercise))
	case exercism.CommitNoChanges:
		info("No file changes to commit (submission still sent).")
	case exercism.CommitAborted:
		info("Commit aborted; nothing pushed.")
	}
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

func syncBeforeWork(cfg *config.Config) bool {
	if !exercism.GitClean(cfg) {
		fail("Working tree has uncommitted changes. Commit or stash first.")
		return false
	}
	if err := exercism.GitPullFF(cfg); err != nil {
		fail("git pull --ff-only failed:\n" + strings.TrimSpace(err.Error()))
		return false
	}
	return true
}
