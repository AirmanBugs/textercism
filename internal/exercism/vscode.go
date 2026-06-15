package exercism

import (
	"errors"
	"os/exec"
	"path/filepath"

	"github.com/AirmanBugs/textercism/internal/config"
)

// ErrCodeNotFound means the `code` CLI isn't on PATH.
var ErrCodeNotFound = errors.New("`code` not found on PATH")

// OpenVSCode opens the exercise as a single-folder window (one language server,
// avoiding the multi-root crash) with the solution file active.
//
// We deliberately don't try to arrange a README pane here: VS Code 1.124's `code`
// CLI has no `--command` flag, so pane splits / preview can't be driven from the
// outside. Instructions are shown in the browser instead (see actions.Open), which
// renders them perfectly and can sit beside the editor window.
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
	return exec.Command(code, args...).Run()
}
