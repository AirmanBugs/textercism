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

// DisplayStatus merges the server-side Status with the local on-disk state into
// the single state shown to the user. This is what the badge/label reflect.
type DisplayStatus int

const (
	DNotStarted    DisplayStatus = iota // no solution anywhere
	DStartedServer                      // server says started/iterated, nothing local to continue
	DStarted                            // downloaded, stub untouched
	DInProgress                         // downloaded with local edits (real WIP)
	DCompleted                          // marked complete
	DPublished                          // completed + published
	DLocked                             // not yet unlocked
)

// Display merges an exercise's server Status with its LocalState. The local
// state only refines the "in between" states; completed/published/locked are
// authoritative from the server regardless of disk.
func Display(s Status, local LocalState) DisplayStatus {
	switch s {
	case Completed:
		return DCompleted
	case Published:
		return DPublished
	case Locked:
		return DLocked
	}
	// NotStarted / InProgress (server started|iterated) refined by local state.
	switch local {
	case OnDiskEdited:
		return DInProgress
	case OnDiskUnedited:
		return DStarted
	default: // NotOnDisk
		if s == InProgress {
			return DStartedServer
		}
		return DNotStarted
	}
}

type badge struct {
	glyph string
	color lipgloss.Color
	label string
}

var badges = map[DisplayStatus]badge{
	DNotStarted:    {"●", lipgloss.Color("8"), "not started"},
	DStartedServer: {"◌", lipgloss.Color("3"), "started (server)"},
	DStarted:       {"◔", lipgloss.Color("3"), "started"},
	DInProgress:    {"◐", lipgloss.Color("3"), "in progress"},
	DCompleted:     {"✓", lipgloss.Color("2"), "completed"},
	DPublished:     {"★", lipgloss.Color("6"), "published"},
	DLocked:        {"🔒", lipgloss.Color("1"), "locked"},
}

// Badge returns the colored glyph for a display status.
func (d DisplayStatus) Badge() string {
	b := badges[d]
	return lipgloss.NewStyle().Foreground(b.color).Render(b.glyph)
}

// Label returns the human-readable name for a display status.
func (d DisplayStatus) Label() string {
	return badges[d].label
}

// Legend renders the status legend line for list footers.
func Legend() string {
	order := []DisplayStatus{DNotStarted, DStartedServer, DStarted, DInProgress, DCompleted, DPublished, DLocked}
	out := ""
	for i, d := range order {
		if i > 0 {
			out += "  "
		}
		out += d.Badge() + " " + d.Label()
	}
	return out
}
