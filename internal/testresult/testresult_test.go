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

func TestParseFailing(t *testing.T) {
	res := Parse(load(t, "bird-count-fail.txt"))

	if res.Total != 11 || res.Failures != 10 {
		t.Fatalf("summary = %d tests, %d failures; want 11, 10", res.Total, res.Failures)
	}
	if res.Passed {
		t.Fatal("expected Passed=false")
	}
	if len(res.Failed) != 10 {
		t.Fatalf("parsed %d failure blocks; want 10", len(res.Failed))
	}

	// First failure should be fully populated.
	f := res.Failed[0]
	if f.Name != "today/1 returns today's bird count" {
		t.Errorf("name = %q", f.Name)
	}
	if f.Location != "test/bird_count_test.exs:11" {
		t.Errorf("location = %q", f.Location)
	}
	if f.Code != "assert BirdCount.today([5]) == 5" {
		t.Errorf("code = %q", f.Code)
	}
	if f.Left != "nil" || f.Right != "5" {
		t.Errorf("left/right = %q / %q; want nil / 5", f.Left, f.Right)
	}
}

func TestParsePassing(t *testing.T) {
	res := Parse(load(t, "pass.txt"))
	if !res.Passed {
		t.Fatalf("expected Passed=true, got failures=%d", res.Failures)
	}
	if res.Total != 8 || res.Failures != 0 {
		t.Fatalf("summary = %d, %d; want 8, 0", res.Total, res.Failures)
	}
	if len(res.Failed) != 0 {
		t.Fatalf("expected no failure blocks, got %d", len(res.Failed))
	}
}
