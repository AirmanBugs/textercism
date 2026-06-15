package exercism

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/AirmanBugs/exercism/xrc/internal/config"
)

// Download fetches an exercise via the official CLI, then moves it from the
// exercism workspace into the repo at <repo>/<track>/<exercise>. Returns the
// repo dir. Mirrors the old start-exercise.sh move logic.
func Download(cfg *config.Config, track, exercise string) (string, error) {
	out, err := exec.Command("exercism", "download",
		"--track="+track, "--exercise="+exercise, "--force").CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("exercism download failed: %s", strings.TrimSpace(string(out)))
	}
	return moveIntoRepo(cfg, track, exercise, string(out))
}

// The CLI prints "Downloaded to\n<path>"; move that dir into the repo.
func moveIntoRepo(cfg *config.Config, track, exercise, output string) (string, error) {
	target := ExerciseDir(cfg, track, exercise)
	downloaded := parseDownloadedPath(output)

	switch {
	case downloaded == "" && dirExists(target):
		return target, nil
	case downloaded != "" && downloaded == target:
		return target, nil
	case downloaded != "" && dirExists(downloaded):
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return "", err
		}
		_ = os.RemoveAll(target)
		if err := os.Rename(downloaded, target); err != nil {
			return "", err
		}
		return target, nil
	case dirExists(target):
		return target, nil
	default:
		return "", fmt.Errorf("could not locate downloaded exercise from CLI output:\n%s", output)
	}
}

func parseDownloadedPath(output string) string {
	lines := strings.Split(output, "\n")
	for i, line := range lines {
		if strings.Contains(line, "Downloaded to") && i+1 < len(lines) {
			return strings.TrimSpace(lines[i+1])
		}
	}
	return ""
}

// Test runs the exercise's tests, streaming output to stdout/stderr. Elixir uses
// `mix test`; other tracks use the CLI's `exercism test` runner.
func Test(cfg *config.Config, track, exercise string) error {
	dir := ExerciseDir(cfg, track, exercise)
	name, args := testCommand(track)

	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

func testCommand(track string) (string, []string) {
	if track == "elixir" {
		return "mix", []string{"test"}
	}
	return "exercism", []string{"test"}
}

// Submit copies the exercise into the exercism workspace and runs `exercism
// submit` on its solution files, then removes the copy. Mirrors end-exercise.sh.
func Submit(cfg *config.Config, track, exercise string) (string, error) {
	repoDir := ExerciseDir(cfg, track, exercise)
	if !dirExists(repoDir) {
		return "", fmt.Errorf("exercise not downloaded: %s", repoDir)
	}
	files := SolutionFiles(cfg, track, exercise)
	if len(files) == 0 {
		return "", fmt.Errorf("no solution files listed in .exercism/config.json for %s", exercise)
	}

	wsDir := filepath.Join(cfg.Workspace, track, exercise)
	if err := os.MkdirAll(filepath.Dir(wsDir), 0o755); err != nil {
		return "", err
	}
	_ = os.RemoveAll(wsDir)
	if err := copyDir(repoDir, wsDir); err != nil {
		return "", err
	}
	defer os.RemoveAll(wsDir)

	args := append([]string{"submit"}, files...)
	cmd := exec.Command("exercism", args...)
	cmd.Dir = wsDir
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

// copyDir recursively copies src to dst, preserving file modes.
func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, info.Mode())
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, info.Mode())
	})
}
