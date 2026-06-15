package tui

import (
	"fmt"

	"github.com/AirmanBugs/exercism/xrc/internal/exercism"
)

// trackItem implements bubbles/list DefaultItem.
type trackItem struct {
	slug   string
	title  string
	joined bool
	done   int
	total  int
}

func (t trackItem) Title() string {
	marker := "·"
	if t.joined {
		marker = "✔"
	}
	return fmt.Sprintf("%s %s", marker, t.slug)
}

func (t trackItem) Description() string {
	return fmt.Sprintf("%d/%d completed", t.done, t.total)
}

func (t trackItem) FilterValue() string { return t.slug + " " + t.title }

// exerciseItem implements DefaultItem.
type exerciseItem struct {
	ex         exercism.Exercise
	downloaded bool
}

func (e exerciseItem) Title() string {
	local := " "
	if e.downloaded {
		local = "⬇"
	}
	rec := ""
	if e.ex.IsRecommended {
		rec = "  ★rec"
	}
	return fmt.Sprintf("%s %s %s%s", e.ex.Status.Badge(), local, e.ex.Title, rec)
}

func (e exerciseItem) Description() string {
	diff := e.ex.Difficulty
	if diff == "" {
		diff = "—"
	}
	return fmt.Sprintf("[%s] %s", diff, e.ex.Status.Label())
}

func (e exerciseItem) FilterValue() string { return e.ex.Title + " " + e.ex.Slug }

// actionItem implements DefaultItem.
type actionItem struct {
	label string
	desc  string
	kind  ActionKind
}

func (a actionItem) Title() string       { return a.label }
func (a actionItem) Description() string { return a.desc }
func (a actionItem) FilterValue() string { return a.label }
