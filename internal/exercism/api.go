// Package exercism wraps the official `exercism` CLI and the unofficial v2
// website API. The API is read-only here (status/listing); writes go through the
// supported CLI surface (download/submit) only.
package exercism

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"time"

	"github.com/AirmanBugs/exercism/xrc/internal/config"
)

// Track is a v2 track summary.
type Track struct {
	Slug                  string `json:"slug"`
	Title                 string `json:"title"`
	IsJoined              bool   `json:"is_joined"`
	NumExercises          int    `json:"num_exercises"`
	NumCompletedExercises int    `json:"num_completed_exercises"`
	WebURL                string `json:"web_url"`
}

// Exercise is the merged exercise + solution view used by the UI.
type Exercise struct {
	Slug          string
	Title         string
	Difficulty    string
	Blurb         string
	IsUnlocked    bool
	IsRecommended bool
	Status        Status
	WebURL        string
}

type apiExercise struct {
	Slug          string `json:"slug"`
	Title         string `json:"title"`
	Difficulty    string `json:"difficulty"`
	Blurb         string `json:"blurb"`
	IsUnlocked    bool   `json:"is_unlocked"`
	IsRecommended bool   `json:"is_recommended"`
	Links         struct {
		Self string `json:"self"`
	} `json:"links"`
}

type apiSolution struct {
	UUID       string `json:"uuid"`
	Status     string `json:"status"`
	PublicURL  string `json:"public_url"`
	PrivateURL string `json:"private_url"`
	Exercise   struct {
		Slug string `json:"slug"`
	} `json:"exercise"`
}

// Client calls the v2 API with the CLI's bearer token.
type Client struct {
	cfg  *config.Config
	http *http.Client
}

func NewClient(cfg *config.Config) *Client {
	return &Client{cfg: cfg, http: &http.Client{Timeout: 20 * time.Second}}
}

// Tracks lists all tracks, joined ones first.
func (c *Client) Tracks() ([]Track, error) {
	var body struct {
		Tracks []Track `json:"tracks"`
	}
	if err := c.get("/tracks", &body); err != nil {
		return nil, err
	}
	sort.SliceStable(body.Tracks, func(i, j int) bool {
		return body.Tracks[i].IsJoined && !body.Tracks[j].IsJoined
	})
	return body.Tracks, nil
}

// Exercises lists a track's exercises merged with the user's solutions (status).
func (c *Client) Exercises(track string) ([]Exercise, error) {
	var body struct {
		Exercises []apiExercise `json:"exercises"`
		Solutions []apiSolution `json:"solutions"`
	}
	path := fmt.Sprintf("/tracks/%s/exercises?sideload=solutions", track)
	if err := c.get(path, &body); err != nil {
		return nil, err
	}

	bySlug := make(map[string]apiSolution, len(body.Solutions))
	for _, s := range body.Solutions {
		bySlug[s.Exercise.Slug] = s
	}

	out := make([]Exercise, 0, len(body.Exercises))
	for _, e := range body.Exercises {
		sol, has := bySlug[e.Slug]
		title := e.Title
		if title == "" {
			title = e.Slug
		}
		out = append(out, Exercise{
			Slug:          e.Slug,
			Title:         title,
			Difficulty:    e.Difficulty,
			Blurb:         e.Blurb,
			IsUnlocked:    e.IsUnlocked,
			IsRecommended: e.IsRecommended,
			Status:        DeriveStatus(e.IsUnlocked, sol.Status, has),
			WebURL:        webURL(track, e, sol, has),
		})
	}
	return out, nil
}

func webURL(track string, e apiExercise, sol apiSolution, hasSol bool) string {
	switch {
	case hasSol && sol.PublicURL != "":
		return sol.PublicURL
	case hasSol && sol.PrivateURL != "":
		return sol.PrivateURL
	case e.Links.Self != "":
		if len(e.Links.Self) > 0 && e.Links.Self[0] == '/' {
			return "https://exercism.org" + e.Links.Self
		}
		return e.Links.Self
	default:
		return fmt.Sprintf("https://exercism.org/tracks/%s/exercises/%s", track, e.Slug)
	}
}

func (c *Client) get(path string, out any) error {
	req, err := http.NewRequest(http.MethodGet, config.V2Base+path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.cfg.Token)
	req.Header.Set("User-Agent", "xrc (exercism personal tool)")

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("API %s returned %d: %s", path, resp.StatusCode, string(b))
	}
	return json.NewDecoder(resp.Body).Decode(out)
}
