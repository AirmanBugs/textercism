package tui

import (
	"fmt"
	"strings"

	"github.com/AirmanBugs/textercism/internal/render"
	"github.com/AirmanBugs/textercism/internal/testresult"
	"github.com/charmbracelet/lipgloss"
)

var (
	passStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("2")) // green
	failStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("1")) // red
	bannerOK  = lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Bold(true)
	bannerBad = lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Bold(true)
	dimText   = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
)

// renderTestView builds the clean test-results pane with lipgloss (not Glamour,
// which strips inline color): a banner, then each test as a colored ┃ bar + ✓/✗
// glyph + name, wrapped to width. When showAssertions is true, failed tests also
// show their assertion detail (rendered via Glamour for the code blocks).
func renderTestView(res testresult.Result, width int, showAssertions bool) string {
	if width < 10 {
		width = 10
	}
	var b strings.Builder

	banner := bannerBad
	if res.AllPassed {
		banner = bannerOK
	}
	b.WriteString(banner.Render(res.Banner()))
	b.WriteString("\n\n")

	for i, t := range res.Tests {
		bar, glyph := failStyle, "✗"
		if t.Passed {
			bar, glyph = passStyle, "✓"
		}

		// "┃ 1. ✓ " prefix; wrap the name to the remaining width with a hanging
		// indent so continuation lines align under the name.
		num := fmt.Sprintf("%d. ", i+1)
		prefix := bar.Render("┃") + " " + dimText.Render(num) + bar.Render(glyph) + " "
		indent := strings.Repeat(" ", lipgloss.Width(prefix))
		nameWidth := width - lipgloss.Width(prefix)
		if nameWidth < 8 {
			nameWidth = 8
		}
		wrapped := lipgloss.NewStyle().Width(nameWidth).Render(t.Name)
		b.WriteString(joinHanging(prefix, indent, wrapped))
		b.WriteString("\n")

		if !t.Passed && showAssertions {
			md := t.Failure.AssertionMarkdown()
			if md != "" {
				b.WriteString(render.Markdown(md, width-2))
			}
		}
	}

	return b.String()
}

// joinHanging prefixes the first line of body with prefix and every later line
// with indent, so wrapped text hangs under the first line's content.
func joinHanging(prefix, indent, body string) string {
	lines := strings.Split(body, "\n")
	for i, ln := range lines {
		if i == 0 {
			lines[i] = prefix + ln
		} else {
			lines[i] = indent + ln
		}
	}
	return strings.Join(lines, "\n")
}
