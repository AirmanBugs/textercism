package tui

import (
	"fmt"
	"strings"

	"github.com/AirmanBugs/textercism/internal/hints"
	"github.com/AirmanBugs/textercism/internal/render"
)

// renderHintsView shows the first `shown` hint bullets (grouped by section),
// with a counter and a prompt to reveal more. Rendered via Glamour as markdown.
func renderHintsView(h hints.Hints, shown, width int) string {
	if len(h.Items) == 0 {
		return render.Markdown("# Hints\n\n_No hints for this exercise._", width)
	}
	if shown > len(h.Items) {
		shown = len(h.Items)
	}

	var b strings.Builder
	fmt.Fprintf(&b, "# Hints  (%d of %d shown)\n\n", shown, len(h.Items))

	lastSection := ""
	for i := 0; i < shown; i++ {
		hint := h.Items[i]
		if hint.Section != "" && hint.Section != lastSection {
			fmt.Fprintf(&b, "## %s\n\n", hint.Section)
			lastSection = hint.Section
		}
		fmt.Fprintf(&b, "- %s\n", hint.Text)
	}

	b.WriteString("\n")
	if shown < len(h.Items) {
		b.WriteString("_Press **n** to reveal the next hint._\n")
	} else {
		b.WriteString("_That's all the hints._\n")
	}
	if len(h.Docs) > 0 {
		b.WriteString("_Press **o** to open the referenced docs in your browser._\n")
	}

	return render.Markdown(b.String(), width)
}
