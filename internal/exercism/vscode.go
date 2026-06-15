package exercism

import (
	"errors"
	"os/exec"
	"path/filepath"

	"github.com/AirmanBugs/exercism/xrc/internal/config"
)

// ErrCodeNotFound means the `code` CLI isn't on PATH.
var ErrCodeNotFound = errors.New("`code` not found on PATH")

// OpenVSCode opens the exercise as a single-folder window (one language server,
// avoiding the multi-root crash) with the solution and README in panes, and the
// README shown as a rendered preview to the side.
func OpenVSCode(cfg *config.Config, track, exercise string) error {
	code, err := exec.LookPath("code")
	if err != nil {
		return ErrCodeNotFound
	}

	dir := ExerciseDir(cfg, track, exercise)
	args := []string{"--new-window", dir}

	if sol := SolutionFiles(cfg, track, exercise); len(sol) > 0 {
		args = append(args, filepath.Join(dir, sol[0]))
	}
	readme := Readme(cfg, track, exercise)
	if readme != "" {
		args = append(args, readme)
	}

	if err := exec.Command(code, args...).Run(); err != nil {
		return err
	}

	if readme != "" {
		// Best-effort: show the README as a preview beside the solution.
		_ = exec.Command(code, "--reuse-window", "--command", "markdown.showPreviewToSide").Run()
	}
	return nil
}
