package exercism

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/AirmanBugs/exercism/xrc/internal/config"
)

// setupExercise creates a minimal downloaded exercise with a solution file and
// the .exercism config that lists it.
func setupExercise(t *testing.T, solution string) (*config.Config, string, string) {
	t.Helper()
	root := t.TempDir()
	cfg := &config.Config{RepoRoot: root}
	track, ex := "elixir", "demo"
	dir := filepath.Join(root, track, ex)

	mustWrite(t, filepath.Join(dir, ".exercism", "config.json"),
		`{"files":{"solution":["lib/demo.ex"]}}`)
	mustWrite(t, filepath.Join(dir, "lib", "demo.ex"), solution)
	return cfg, track, ex
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestLocalStateEditDetection(t *testing.T) {
	cfg, track, ex := setupExercise(t, "def stub, do: :todo\n")

	// No stub captured yet -> treated as possibly-edited (don't hide local work).
	if got := LocalStateOf(cfg, track, ex); got != OnDiskEdited {
		t.Fatalf("no stub: got %v, want OnDiskEdited", got)
	}

	// Capture the stub (simulates post-download). Now solution == stub.
	if err := captureStub(cfg, track, ex); err != nil {
		t.Fatal(err)
	}
	if got := LocalStateOf(cfg, track, ex); got != OnDiskUnedited {
		t.Fatalf("post-capture: got %v, want OnDiskUnedited", got)
	}

	// Edit the solution -> detected as in-progress.
	mustWrite(t, filepath.Join(ExerciseDir(cfg, track, ex), "lib", "demo.ex"),
		"def real, do: 42\n")
	if got := LocalStateOf(cfg, track, ex); got != OnDiskEdited {
		t.Fatalf("post-edit: got %v, want OnDiskEdited", got)
	}
}

func TestLocalStateNotOnDisk(t *testing.T) {
	cfg := &config.Config{RepoRoot: t.TempDir()}
	if got := LocalStateOf(cfg, "elixir", "missing"); got != NotOnDisk {
		t.Fatalf("missing exercise: got %v, want NotOnDisk", got)
	}
}
