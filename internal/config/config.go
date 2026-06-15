// Package config reads the Exercism CLI configuration (user.json). The CLI's API
// token authenticates against the unofficial v2 website API, and its workspace is
// where exercises are downloaded and where textercism reads/writes drafts — so
// user.json is the single source of both credentials and storage location.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// V2Base is the fixed base URL for the unofficial v2 website API (not the CLI's v1 base).
const V2Base = "https://exercism.org/api/v2"

// Config holds the resolved Exercism credentials and the workspace (storage root).
type Config struct {
	Token      string
	Workspace  string // exercises live at <Workspace>/<track>/<exercise>
	APIBaseURL string
}

type userJSON struct {
	Token      string `json:"token"`
	Workspace  string `json:"workspace"`
	APIBaseURL string `json:"apibaseurl"`
}

// Load reads user.json, or returns a friendly error if the Exercism CLI isn't
// configured.
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
		Workspace:  expandHome(u.Workspace),
		APIBaseURL: apiBase,
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

// expandHome resolves a leading ~ in a path to the user's home directory.
func expandHome(path string) string {
	if path == "~" || (len(path) >= 2 && path[:2] == "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, path[1:])
		}
	}
	return path
}
