package exercism

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/AirmanBugs/exercism/xrc/internal/config"
)

// ExerciseDir returns the absolute path to an exercise in the repo.
func ExerciseDir(cfg *config.Config, track, exercise string) string {
	return filepath.Join(cfg.RepoRoot, track, exercise)
}

// Downloaded reports whether the exercise exists locally (has a .exercism dir).
func Downloaded(cfg *config.Config, track, exercise string) bool {
	info, err := os.Stat(filepath.Join(ExerciseDir(cfg, track, exercise), ".exercism"))
	return err == nil && info.IsDir()
}

type exerciseConfig struct {
	Files struct {
		Solution []string `json:"solution"`
	} `json:"files"`
}

// SolutionFiles returns the solution file paths (relative to the exercise dir)
// from .exercism/config.json, or nil if unavailable.
func SolutionFiles(cfg *config.Config, track, exercise string) []string {
	path := filepath.Join(ExerciseDir(cfg, track, exercise), ".exercism", "config.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var ec exerciseConfig
	if err := json.Unmarshal(raw, &ec); err != nil {
		return nil
	}
	return ec.Files.Solution
}

// Readme returns the exercise's README.md path if it exists.
func Readme(cfg *config.Config, track, exercise string) string {
	path := filepath.Join(ExerciseDir(cfg, track, exercise), "README.md")
	if _, err := os.Stat(path); err == nil {
		return path
	}
	return ""
}

// stubDir is where the pristine solution stub is saved at download time.
const stubSubdir = ".exercism/stub"

// LocalState describes the on-disk state of an exercise's solution.
type LocalState int

const (
	// NotOnDisk: the exercise isn't downloaded into the repo.
	NotOnDisk LocalState = iota
	// OnDiskUnedited: downloaded, but the solution still matches the stub.
	OnDiskUnedited
	// OnDiskEdited: downloaded and the solution differs from the stub (real WIP).
	OnDiskEdited
)

// captureStub copies the current solution files into .exercism/stub so later
// edits can be detected. Called right after download (solution == stub).
func captureStub(cfg *config.Config, track, exercise string) error {
	dir := ExerciseDir(cfg, track, exercise)
	stubRoot := filepath.Join(dir, stubSubdir)
	if err := os.RemoveAll(stubRoot); err != nil {
		return err
	}
	for _, rel := range SolutionFiles(cfg, track, exercise) {
		src := filepath.Join(dir, rel)
		data, err := os.ReadFile(src)
		if err != nil {
			return err
		}
		dst := filepath.Join(stubRoot, rel)
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(dst, data, 0o644); err != nil {
			return err
		}
	}
	return nil
}

// LocalStateOf reports whether the exercise is on disk and, if so, whether its
// solution has been edited away from the captured stub. If no stub was captured
// (e.g. an exercise downloaded before this feature), a downloaded exercise is
// treated as edited so the user never loses sight of existing local work.
func LocalStateOf(cfg *config.Config, track, exercise string) LocalState {
	if !Downloaded(cfg, track, exercise) {
		return NotOnDisk
	}

	dir := ExerciseDir(cfg, track, exercise)
	stubRoot := filepath.Join(dir, stubSubdir)
	files := SolutionFiles(cfg, track, exercise)

	if _, err := os.Stat(stubRoot); err != nil || len(files) == 0 {
		// No stub to compare against — assume there may be local work.
		return OnDiskEdited
	}

	for _, rel := range files {
		cur, err := os.ReadFile(filepath.Join(dir, rel))
		if err != nil {
			return OnDiskEdited
		}
		stub, err := os.ReadFile(filepath.Join(stubRoot, rel))
		if err != nil || !bytes.Equal(cur, stub) {
			return OnDiskEdited
		}
	}
	return OnDiskUnedited
}
