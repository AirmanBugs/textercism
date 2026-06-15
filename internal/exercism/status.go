package exercism

import "github.com/charmbracelet/lipgloss"

// Status is the derived state of an exercise.
type Status int

const (
	NotStarted Status = iota
	InProgress
	Completed
	Published
	Locked
)

// DeriveStatus maps an exercise + optional solution to a Status. The v2 schema
// is unofficial, so unknown non-empty solution statuses fall back to InProgress.
func DeriveStatus(isUnlocked bool, solutionStatus string, hasSolution bool) Status {
	if !hasSolution {
		if isUnlocked {
			return NotStarted
		}
		return Locked
	}
	switch solutionStatus {
	case "completed":
		return Completed
	case "published":
		return Published
	case "started", "iterated":
		return InProgress
	default:
		return InProgress
	}
}

type badge struct {
	glyph string
	color lipgloss.Color
	label string
}

var badges = map[Status]badge{
	NotStarted: {"●", lipgloss.Color("8"), "not started"},
	InProgress: {"◐", lipgloss.Color("3"), "in progress"},
	Completed:  {"✓", lipgloss.Color("2"), "completed"},
	Published:  {"★", lipgloss.Color("6"), "published"},
	Locked:     {"🔒", lipgloss.Color("1"), "locked"},
}

// Badge returns the colored glyph for a status.
func (s Status) Badge() string {
	b := badges[s]
	return lipgloss.NewStyle().Foreground(b.color).Render(b.glyph)
}

// Label returns the human-readable name for a status.
func (s Status) Label() string {
	return badges[s].label
}

// Legend renders the status legend line for list footers.
func Legend() string {
	order := []Status{NotStarted, InProgress, Completed, Published, Locked}
	out := ""
	for i, s := range order {
		if i > 0 {
			out += "   "
		}
		out += s.Badge() + " " + s.Label()
	}
	return out
}
