package exercism

import "testing"

func TestDisplayMerge(t *testing.T) {
	cases := []struct {
		name   string
		server Status
		local  LocalState
		want   DisplayStatus
	}{
		{"nothing anywhere", NotStarted, NotOnDisk, DNotStarted},
		{"server started, not local", InProgress, NotOnDisk, DStartedServer},
		{"downloaded, untouched stub", InProgress, OnDiskUnedited, DStarted},
		{"downloaded, untouched, server not-started", NotStarted, OnDiskUnedited, DStarted},
		{"downloaded with edits", InProgress, OnDiskEdited, DInProgress},
		{"edits even if server not-started", NotStarted, OnDiskEdited, DInProgress},
		{"completed is authoritative regardless of disk", Completed, NotOnDisk, DCompleted},
		{"published is authoritative", Published, OnDiskEdited, DPublished},
		{"locked is authoritative", Locked, NotOnDisk, DLocked},
	}
	for _, c := range cases {
		if got := Display(c.server, c.local); got != c.want {
			t.Errorf("%s: Display(%v,%v) = %v, want %v", c.name, c.server, c.local, got, c.want)
		}
	}
}

func TestDeriveStatus(t *testing.T) {
	if got := DeriveStatus(true, "", false); got != NotStarted {
		t.Errorf("no solution, unlocked = %v, want NotStarted", got)
	}
	if got := DeriveStatus(false, "", false); got != Locked {
		t.Errorf("no solution, locked = %v, want Locked", got)
	}
	if got := DeriveStatus(true, "iterated", true); got != InProgress {
		t.Errorf("iterated = %v, want InProgress", got)
	}
	if got := DeriveStatus(true, "weird-future-value", true); got != InProgress {
		t.Errorf("unknown status with solution = %v, want InProgress (lenient)", got)
	}
}
