package exercism

import (
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
