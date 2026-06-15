// Package config reads the Exercism CLI configuration (user.json) and resolves
// the local git repo root. The CLI's API token also authenticates against the
// unofficial v2 website API, so it's the single source of credentials for xrc.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// V2Base is the fixed base URL for the unofficial v2 website API (not the CLI's v1 base).
const V2Base = "https://exercism.org/api/v2"

// Config holds the resolved CLI credentials and the repo root.
type Config struct {
	Token      string
	Workspace  string
	APIBaseURL string
	RepoRoot   string
}

type userJSON struct {
	Token      string `json:"token"`
	Workspace  string `json:"workspace"`
	APIBaseURL string `json:"apibaseurl"`
}

// Load reads user.json and resolves the repo root, or returns a friendly error
// if the Exercism CLI isn't configured.
func Load() (*Config, error) {
	path := userJSONPath()

	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("Exercism CLI config not found at %s.\nRun `exercism configure --token=<your-token>` first", path)
	}

	var u userJSON
	if err := json.Unmarshal(raw, &u); err != nil {
		return nil, fmt.Errorf("could not parse %s: %w", path, err)
	}
	if u.Token == "" {
		return nil, fmt.Errorf("missing %q in %s; re-run `exercism configure`", "token", path)
	}
	if u.Workspace == "" {
		return nil, fmt.Errorf("missing %q in %s; re-run `exercism configure`", "workspace", path)
	}

	apiBase := u.APIBaseURL
	if apiBase == "" {
		apiBase = "https://api.exercism.org/v1"
	}

	return &Config{
		Token:      u.Token,
		Workspace:  u.Workspace,
		APIBaseURL: apiBase,
		RepoRoot:   repoRoot(),
	}, nil
}

// userJSONPath honors the same env vars as the official CLI: EXERCISM_CONFIG_HOME,
// then XDG_CONFIG_HOME/exercism, then ~/.config/exercism.
func userJSONPath() string {
	var dir string
	switch {
	case os.Getenv("EXERCISM_CONFIG_HOME") != "":
		dir = os.Getenv("EXERCISM_CONFIG_HOME")
	case os.Getenv("XDG_CONFIG_HOME") != "":
		dir = filepath.Join(os.Getenv("XDG_CONFIG_HOME"), "exercism")
	default:
		home, _ := os.UserHomeDir()
		dir = filepath.Join(home, ".config", "exercism")
	}
	return filepath.Join(dir, "user.json")
}

// repoRoot resolves the repo holding the per-track exercise folders. It prefers
// the binary's own location (<repo>/tooling/xrc) so xrc works from any directory
// via a PATH symlink; honors XRC_REPO_ROOT; falls back to git, then CWD.
func repoRoot() string {
	if dir := os.Getenv("XRC_REPO_ROOT"); dir != "" {
		return dir
	}
	if dir := repoRootFromBinary(); dir != "" {
		return dir
	}
	if dir := repoRootFromGit(); dir != "" {
		return dir
	}
	cwd, _ := os.Getwd()
	return cwd
}

func repoRootFromBinary() string {
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	// Resolve symlinks so a ~/.local/bin/xrc symlink points back to <repo>/tooling/xrc.
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}
	dir := filepath.Dir(exe) // <repo>/tooling
	if filepath.Base(dir) == "tooling" {
		return filepath.Dir(dir)
	}
	return ""
}

func repoRootFromGit() string {
	out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
