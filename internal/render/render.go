// Package render turns exercise instructions (markdown) into styled terminal
// output using Glamour, so the README can be read inside the TUI instead of a
// browser.
package render

import (
	"os"

	"github.com/AirmanBugs/textercism/internal/config"
	"github.com/AirmanBugs/textercism/internal/exercism"
	"github.com/charmbracelet/glamour"
)

// Markdown renders markdown source to ANSI-styled text wrapped to width. It
// auto-detects light/dark terminal styling. On error it falls back to the raw
// source so the caller always has something to show.
func Markdown(src string, width int) string {
	if width <= 0 {
		width = 80
	}
	r, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(width),
	)
	if err != nil {
		return src
	}
	out, err := r.Render(src)
	if err != nil {
		return src
	}
	return out
}

// Instructions returns the rendered README for an exercise. If the exercise
// isn't downloaded (no local README), ok is false so the caller can offer to
// download it first.
func Instructions(cfg *config.Config, track, exercise string, width int) (text string, ok bool) {
	readme := exercism.Readme(cfg, track, exercise)
	if readme == "" {
		return "", false
	}
	raw, err := os.ReadFile(readme)
	if err != nil {
		return "", false
	}
	return Markdown(string(raw), width), true
}
