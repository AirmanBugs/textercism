// Package sync defines how in-progress drafts are persisted for cross-device use.
// The only thing textercism needs to store is the latest draft of an exercise
// (Exercism itself holds completed/submitted work), so the interface is tiny and
// carries no version history — last write wins.
package sync

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/AirmanBugs/textercism/internal/config"
)

// ErrNoDraft is returned by Pull when no stored draft exists for an exercise.
var ErrNoDraft = errors.New("no stored draft")

// DraftRef identifies a stored draft.
type DraftRef struct {
	Track    string
	Exercise string
}

// Backend persists in-progress drafts. Semantics are "latest only, last write
// wins"; implementations need not version anything. Future backends (a synced
// folder, ssh/scp to a personal server, git) implement this same interface and
// are selected by config.
type Backend interface {
	// Name is shown in status/config (e.g. "local", "folder", "ssh", "git").
	Name() string

	// SyncsAcrossDevices reports whether this backend actually moves drafts
	// between machines. The local backend returns false, which lets the UI hide
	// the "Pause & sync" action that would otherwise be a no-op.
	SyncsAcrossDevices() bool

	// Push saves the exercise's current draft (the files under dir).
	Push(ref DraftRef, dir string) error

	// Pull fetches the latest stored draft into dir (overwriting). Returns
	// ErrNoDraft if none exists.
	Pull(ref DraftRef, dir string) error

	// List returns the drafts the backend knows about.
	List() ([]DraftRef, error)
}

// Local is the default backend: drafts live only in the Exercism workspace on
// this machine. Push/Pull are no-ops (there's nowhere else to sync to); List
// walks the workspace for downloaded exercises.
type Local struct {
	workspace string
}

// NewLocal returns the local-only backend rooted at the Exercism workspace.
func NewLocal(cfg *config.Config) *Local {
	return &Local{workspace: cfg.Workspace}
}

func (l *Local) Name() string             { return "local" }
func (l *Local) SyncsAcrossDevices() bool { return false }

// Push is a no-op: the draft already lives in the workspace.
func (l *Local) Push(DraftRef, string) error { return nil }

// Pull is a no-op: there is no remote to pull from. Callers treat "not present
// locally" as the signal to download a fresh stub instead.
func (l *Local) Pull(DraftRef, string) error { return ErrNoDraft }

// List walks the workspace for downloaded exercises (dirs containing .exercism).
func (l *Local) List() ([]DraftRef, error) {
	var refs []DraftRef

	tracks, err := os.ReadDir(l.workspace)
	if err != nil {
		if os.IsNotExist(err) {
			return refs, nil
		}
		return nil, err
	}

	for _, t := range tracks {
		if !t.IsDir() {
			continue
		}
		trackDir := filepath.Join(l.workspace, t.Name())
		exercises, err := os.ReadDir(trackDir)
		if err != nil {
			continue
		}
		for _, e := range exercises {
			if !e.IsDir() {
				continue
			}
			marker := filepath.Join(trackDir, e.Name(), ".exercism")
			if info, err := os.Stat(marker); err == nil && info.IsDir() {
				refs = append(refs, DraftRef{Track: t.Name(), Exercise: e.Name()})
			}
		}
	}
	return refs, nil
}
