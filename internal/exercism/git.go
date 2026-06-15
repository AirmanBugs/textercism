package exercism

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/AirmanBugs/exercism/xrc/internal/config"
)

// GitClean reports whether the working tree (tracked files) is clean.
func GitClean(cfg *config.Config) bool {
	return gitOK(cfg, "diff", "--quiet") && gitOK(cfg, "diff", "--cached", "--quiet")
}

// GitPullFF fast-forwards to sync progress from other devices. Succeeds (no-op)
// when the branch has no upstream; errors only on a real failure.
func GitPullFF(cfg *config.Config) error {
	if !gitOK(cfg, "rev-parse", "--abbrev-ref", "--symbolic-full-name", "@{u}") {
		return nil
	}
	if out, _, err := git(cfg, "pull", "--ff-only"); err != nil {
		return fmt.Errorf("%s", strings.TrimSpace(out))
	}
	return nil
}

// AlreadyCompleted reports whether a prior "<track>: complete <exercise>" commit exists.
func AlreadyCompleted(cfg *config.Config, track, exercise string) bool {
	out, _, _ := git(cfg, "log", "--oneline", "--grep="+track+": complete "+exercise)
	return strings.TrimSpace(out) != ""
}

// CommitResult is the outcome of CommitAndPush.
type CommitResult int

const (
	CommitOK CommitResult = iota
	CommitNoChanges
	CommitAborted
)

// CommitAndPush stages the exercise dir, shows the stat diff via confirm, then
// commits "<track>: complete <exercise>" and pushes. confirm receives the staged
// stat and returns whether to proceed.
func CommitAndPush(cfg *config.Config, track, exercise string, confirm func(stat string) bool) (CommitResult, error) {
	rel := filepath.Join(track, exercise)

	if !gitHasChanges(cfg, rel) {
		return CommitNoChanges, nil
	}

	if _, _, err := git(cfg, "add", rel+"/"); err != nil {
		return CommitOK, err
	}
	stat, _, _ := git(cfg, "diff", "--cached", "--stat")

	if !confirm(stat) {
		_, _, _ = git(cfg, "reset", "HEAD")
		return CommitAborted, nil
	}

	if out, _, err := git(cfg, "commit", "-m", track+": complete "+exercise); err != nil {
		return CommitOK, fmt.Errorf("commit failed:\n%s", out)
	}
	if out, _, err := git(cfg, "push"); err != nil {
		return CommitOK, fmt.Errorf("push failed:\n%s", out)
	}
	return CommitOK, nil
}

func gitHasChanges(cfg *config.Config, rel string) bool {
	out, _, _ := git(cfg, "status", "--porcelain", "--", rel)
	return strings.TrimSpace(out) != ""
}

func gitOK(cfg *config.Config, args ...string) bool {
	_, code, _ := git(cfg, args...)
	return code == 0
}

// git runs a git command in the repo root and returns combined output, exit code, error.
func git(cfg *config.Config, args ...string) (string, int, error) {
	full := append([]string{"-C", cfg.RepoRoot}, args...)
	cmd := exec.Command("git", full...)
	out, err := cmd.CombinedOutput()
	code := cmd.ProcessState.ExitCode()
	return string(out), code, err
}
