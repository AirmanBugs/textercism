package exercism

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/AirmanBugs/textercism/internal/config"
)

// Download fetches an exercise via the official CLI into the Exercism workspace
// (which is also textercism's storage), captures the pristine stub for later
// edit detection, and returns the exercise dir.
func Download(cfg *config.Config, track, exercise string) (string, error) {
	out, err := exec.Command("exercism", "download",
		"--track="+track, "--exercise="+exercise, "--force").CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("exercism download failed: %s", strings.TrimSpace(string(out)))
	}

	dir := ExerciseDir(cfg, track, exercise)
	if !dirExists(dir) {
		return "", fmt.Errorf("exercise not found at %s after download:\n%s", dir, strings.TrimSpace(string(out)))
	}

	// Capture the pristine stub (solution == stub right after download) so we can
	// later tell whether the user has edited the solution. Non-fatal on failure.
	_ = captureStub(cfg, track, exercise)
	return dir, nil
}

// Test runs the exercise's tests, streaming output to stdout/stderr. Elixir uses
// `mix test`; other tracks use the CLI's `exercism test` runner.
func Test(cfg *config.Config, track, exercise string) error {
	cmd := TestCmd(cfg, track, exercise)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

// TestCmd returns the (unstarted) test command for an exercise, so callers like
// the TUI can run it via tea.ExecProcess (which wires up the terminal itself).
func TestCmd(cfg *config.Config, track, exercise string) *exec.Cmd {
	name, args := testCommand(track)
	cmd := exec.Command(name, args...)
	cmd.Dir = ExerciseDir(cfg, track, exercise)
	return cmd
}

func testCommand(track string) (string, []string) {
	if track == "elixir" {
		return "mix", []string{"test"}
	}
	return "exercism", []string{"test"}
}

// Submit runs `exercism submit` on the exercise's solution files in place. The
// exercise already lives in the Exercism workspace, so no copy is needed.
func Submit(cfg *config.Config, track, exercise string) (string, error) {
	dir := ExerciseDir(cfg, track, exercise)
	if !dirExists(dir) {
		return "", fmt.Errorf("exercise not downloaded: %s", dir)
	}
	files := SolutionFiles(cfg, track, exercise)
	if len(files) == 0 {
		return "", fmt.Errorf("no solution files listed in .exercism/config.json for %s", exercise)
	}

	args := append([]string{"submit"}, files...)
	cmd := exec.Command("exercism", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("exercism submit failed: %s", strings.TrimSpace(string(out)))
	}
	return strings.TrimSpace(string(out)), nil
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
