// Package render turns exercise instructions (markdown) into styled terminal
// output using Glamour, so the README can be read inside the TUI instead of a
// browser.
package render

import (
	"os"
	"sync"

	"github.com/AirmanBugs/textercism/internal/config"
	"github.com/AirmanBugs/textercism/internal/exercism"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/glamour/ansi"
	"github.com/charmbracelet/glamour/styles"
)

// Building a Glamour renderer is expensive (it sets up a syntax highlighter), so
// we cache one renderer per word-wrap width and reuse it across calls. Without
// this, rendering on every selection/resize caused a visible lag.
var (
	mu        sync.Mutex
	renderers = map[int]*glamour.TermRenderer{}
)

func rendererFor(width int) (*glamour.TermRenderer, error) {
	mu.Lock()
	defer mu.Unlock()
	if r, ok := renderers[width]; ok {
		return r, nil
	}
	r, err := glamour.NewTermRenderer(
		glamour.WithStyles(headingStyle()),
		glamour.WithWordWrap(width),
	)
	if err != nil {
		return nil, err
	}
	renderers[width] = r
	return r, nil
}

// headingStyle is the dark style with the literal "#"/"##" heading prefixes
// removed (Glamour's default keeps them). Headings stay bold/colored, just
// without the markdown markers showing.
func headingStyle() ansi.StyleConfig {
	s := styles.DarkStyleConfig
	s.H1.Prefix = ""
	s.H1.Suffix = ""
	s.H2.Prefix = ""
	s.H3.Prefix = ""
	s.H4.Prefix = ""
	s.H5.Prefix = ""
	s.H6.Prefix = ""
	return s
}

// Markdown renders markdown source to ANSI-styled text wrapped to width. On
// error it falls back to the raw source so the caller always has something.
func Markdown(src string, width int) string {
	if width <= 0 {
		width = 80
	}
	r, err := rendererFor(width)
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
