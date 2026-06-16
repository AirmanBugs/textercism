package testresult

import (
	"os"
	"path/filepath"
	"testing"
)

func load(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestParseTrace(t *testing.T) {
	res := Parse(load(t, "bird-count-trace.txt"))

	if res.Total != 11 || res.Failures != 10 {
		t.Fatalf("summary = %d tests, %d failures; want 11, 10", res.Total, res.Failures)
	}
	if res.AllPassed {
		t.Fatal("expected AllPassed=false")
	}

	// All 11 tests listed in run order (passed and failed).
	if len(res.Tests) != 11 {
		t.Fatalf("parsed %d tests; want 11", len(res.Tests))
	}

	// Exactly one passed; it should carry no failure detail.
	passed := 0
	for _, tt := range res.Tests {
		if tt.Passed {
			passed++
		}
	}
	if passed != 1 {
		t.Fatalf("got %d passing tests; want 1", passed)
	}

	// Find a known failing test and check its assertion detail.
	var f *Test
	for i := range res.Tests {
		if res.Tests[i].Name == "today/1 returns today's bird count" {
			f = &res.Tests[i]
		}
	}
	if f == nil {
		t.Fatal("did not find the 'today/1 returns today's bird count' test")
	}
	if f.Passed {
		t.Fatal("expected that test to be failed")
	}
	if f.Failure.Code != "assert BirdCount.today([5]) == 5" {
		t.Errorf("code = %q", f.Failure.Code)
	}
	if f.Failure.Left != "nil" || f.Failure.Right != "5" {
		t.Errorf("left/right = %q / %q; want nil / 5", f.Failure.Left, f.Failure.Right)
	}
}

func TestParsePassing(t *testing.T) {
	res := Parse(load(t, "pass.txt"))
	if !res.AllPassed {
		t.Fatalf("expected AllPassed=true, got failures=%d", res.Failures)
	}
	if res.Total != 8 || res.Failures != 0 {
		t.Fatalf("summary = %d, %d; want 8, 0", res.Total, res.Failures)
	}
}

func TestMarkdownAssertionsToggle(t *testing.T) {
	res := Parse(load(t, "bird-count-trace.txt"))

	compact := res.Markdown(false)
	full := res.Markdown(true)

	// Compact lists numbered tests but no assertion code.
	if !contains(compact, "1.") || !contains(compact, "✗") {
		t.Fatalf("compact view missing numbered failures:\n%s", compact)
	}
	if contains(compact, "assert BirdCount.today") {
		t.Fatal("compact view should not include assertion code")
	}
	// Full view includes the assertion code.
	if !contains(full, "assert BirdCount.today([5]) == 5") {
		t.Fatal("full view should include assertion code")
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (func() bool {
		for i := 0; i+len(sub) <= len(s); i++ {
			if s[i:i+len(sub)] == sub {
				return true
			}
		}
		return false
	})()
}
