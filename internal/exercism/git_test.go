package exercism

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/AirmanBugs/exercism/xrc/internal/config"
)

func gitInit(t *testing.T) *config.Config {
	t.Helper()
	root := t.TempDir()
	run := func(args ...string) {
		cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
		}
	}
	run("init", "-q")
	run("config", "user.email", "t@example.com")
	run("config", "user.name", "Test")
	run("commit", "--allow-empty", "-q", "-m", "init")
	return &config.Config{RepoRoot: root}
}

func commitFile(t *testing.T, cfg *config.Config, rel, content, msg string) {
	t.Helper()
	mustWrite(t, filepath.Join(cfg.RepoRoot, rel), content)
	if _, _, err := git(cfg, "add", "-A"); err != nil {
		t.Fatal(err)
	}
	if out, _, err := git(cfg, "commit", "-q", "-m", msg); err != nil {
		t.Fatalf("commit: %v\n%s", err, out)
	}
}

func subjects(t *testing.T, cfg *config.Config) []string {
	out, _, _ := git(cfg, "log", "--format=%s")
	var s []string
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		if line != "" {
			s = append(s, line)
		}
	}
	return s
}

func TestSquashWipIntoComplete(t *testing.T) {
	cfg := gitInit(t)
	track, ex := "elixir", "demo"

	// Two WIP commits for the exercise sit at HEAD.
	commitFile(t, cfg, filepath.Join(track, ex, "lib/demo.ex"), "v1", track+": wip "+ex)
	commitFile(t, cfg, filepath.Join(track, ex, "lib/demo.ex"), "v2", track+": wip "+ex)

	// CommitAndPush (no upstream -> no push) should squash both WIPs into one
	// complete commit.
	res, err := CommitAndPush(cfg, track, ex, func(string) bool { return true })
	if err != nil {
		t.Fatal(err)
	}
	if res != CommitOK {
		t.Fatalf("result = %v, want CommitOK", res)
	}

	got := subjects(t, cfg)
	want := []string{track + ": complete " + ex, "init"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("history = %v, want %v", got, want)
	}

	// The final tree keeps the latest content.
	out, _, _ := git(cfg, "show", "HEAD:"+filepath.ToSlash(filepath.Join(track, ex, "lib/demo.ex")))
	if strings.TrimSpace(out) != "v2" {
		t.Fatalf("final content = %q, want v2", strings.TrimSpace(out))
	}
}

func TestWipCommitNoChanges(t *testing.T) {
	cfg := gitInit(t)
	res, err := WipCommitAndPush(cfg, "elixir", "demo")
	if err != nil {
		t.Fatal(err)
	}
	if res != CommitNoChanges {
		t.Fatalf("result = %v, want CommitNoChanges", res)
	}
}

func TestAlreadyCompleted(t *testing.T) {
	cfg := gitInit(t)
	commitFile(t, cfg, "elixir/demo/lib/demo.ex", "x", "elixir: complete demo")
	if !AlreadyCompleted(cfg, "elixir", "demo") {
		t.Fatal("expected AlreadyCompleted to be true")
	}
	if AlreadyCompleted(cfg, "elixir", "other") {
		t.Fatal("expected AlreadyCompleted false for other exercise")
	}
}
