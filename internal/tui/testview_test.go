package tui

import (
	"strings"
	"testing"

	"github.com/AirmanBugs/textercism/internal/testresult"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

func TestRenderTestViewColorsAndStructure(t *testing.T) {
	// Force color rendering even though tests run without a TTY.
	lipgloss.SetColorProfile(termenv.ANSI)

	raw := strings.Join([]string{
		"  * test alpha works (0.1ms) [L#3]",
		"  * test beta works (0.1ms) [L#7]",
		"",
		"  1) test beta works (DemoTest)",
		"     test/demo_test.exs:7",
		"     Assertion with == failed",
		"     code:  assert Demo.beta() == 2",
		"     left:  1",
		"     right: 2",
		"     stacktrace:",
		"       test/demo_test.exs:7: (test)",
		"",
		"2 tests, 1 failure",
	}, "\n")

	res := testresult.Parse(raw)
	out := renderTestView(res, 60, false)

	// Banner reports passed count, not failed.
	if !strings.Contains(out, "1 of 2 passed") {
		t.Fatalf("banner missing 'passed' phrasing:\n%s", out)
	}
	// Numbered, with ✓ and ✗ glyphs and the ┃ bar.
	for _, want := range []string{"1. ", "2. ", "✓", "✗", "┃"} {
		if !strings.Contains(out, want) {
			t.Fatalf("output missing %q:\n%s", want, out)
		}
	}
	// Color escapes are present (red and green) now that color is forced on.
	if !strings.Contains(out, "\x1b[") {
		t.Fatalf("expected ANSI color escapes in output:\n%q", out)
	}
}
