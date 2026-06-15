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

// CommitAndPush finalizes an exercise: it folds any trailing "wip <exercise>"
// commits back into the working tree, stages the exercise dir, shows the stat
// diff via confirm, then commits "<track>: complete <exercise>" and pushes.
// When WIP commits were squashed (history rewritten), the push uses
// --force-with-lease. confirm receives the staged stat and returns whether to
// proceed.
func CommitAndPush(cfg *config.Config, track, exercise string, confirm func(stat string) bool) (CommitResult, error) {
	rel := filepath.Join(track, exercise)

	// Count trailing WIP commits before squashing so we know whether the upstream
	// push must be a force (history was rewritten).
	squashed := countTrailingWip(cfg, track, exercise) > 0
	if err := SquashWipInto(cfg, track, exercise); err != nil {
		return CommitOK, err
	}

	if !gitHasChanges(cfg, rel) && !squashed {
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

	if !gitOK(cfg, "rev-parse", "--abbrev-ref", "--symbolic-full-name", "@{u}") {
		return CommitOK, nil
	}
	pushArgs := []string{"push"}
	if squashed {
		// Squashing rewrote history past already-pushed WIP commits.
		pushArgs = []string{"push", "--force-with-lease"}
	}
	if out, _, err := git(cfg, pushArgs...); err != nil {
		return CommitOK, fmt.Errorf("push failed:\n%s", out)
	}
	return CommitOK, nil
}

// countTrailingWip returns how many consecutive HEAD commits are this exercise's
// "wip" snapshots.
func countTrailingWip(cfg *config.Config, track, exercise string) int {
	want := track + ": wip " + exercise
	n := 0
	for {
		subject, _, err := git(cfg, "log", "-1", "--format=%s", fmt.Sprintf("HEAD~%d", n))
		if err != nil || strings.TrimSpace(subject) != want {
			break
		}
		n++
	}
	return n
}

// WipCommitAndPush stages the exercise dir and commits a "<track>: wip
// <exercise>" snapshot, then pushes (best-effort). Used by Pause & sync to carry
// work-in-progress across devices. Returns CommitNoChanges when nothing changed.
func WipCommitAndPush(cfg *config.Config, track, exercise string) (CommitResult, error) {
	rel := filepath.Join(track, exercise)
	if !gitHasChanges(cfg, rel) {
		return CommitNoChanges, nil
	}
	if _, _, err := git(cfg, "add", rel+"/"); err != nil {
		return CommitOK, err
	}
	if out, _, err := git(cfg, "commit", "-m", track+": wip "+exercise); err != nil {
		return CommitOK, fmt.Errorf("commit failed:\n%s", out)
	}
	// Push is best-effort: a missing upstream shouldn't fail a local snapshot.
	if !gitOK(cfg, "rev-parse", "--abbrev-ref", "--symbolic-full-name", "@{u}") {
		return CommitOK, nil
	}
	if out, _, err := git(cfg, "push"); err != nil {
		return CommitOK, fmt.Errorf("push failed:\n%s", out)
	}
	return CommitOK, nil
}

// SquashWipInto folds any consecutive trailing "<track>: wip <exercise>" commits
// into the working tree so a single "complete" commit can replace them. It does
// a soft reset back past those commits (keeping their changes staged-or-unstaged
// in the tree). Only resets WIP commits for THIS exercise that sit at HEAD; if
// the most recent commit isn't such a WIP, it does nothing.
func SquashWipInto(cfg *config.Config, track, exercise string) error {
	n := countTrailingWip(cfg, track, exercise)
	if n == 0 {
		return nil
	}
	// Soft reset preserves the WIP changes in the index/tree; the caller then
	// stages + commits them as the single "complete" commit.
	if out, _, err := git(cfg, "reset", "--soft", fmt.Sprintf("HEAD~%d", n)); err != nil {
		return fmt.Errorf("squash reset failed:\n%s", out)
	}
	return nil
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
