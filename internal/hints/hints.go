// Package hints parses an Exercism HINTS.md into an ordered list of hint bullets
// (grouped by section) with their reference-style links resolved to URLs, so the
// TUI can reveal hints one at a time and offer the doc links.
package hints

import (
	"bufio"
	"os"
	"regexp"
	"strings"
)

// Hint is a single bullet, tagged with the section it came from.
type Hint struct {
	Section string   // e.g. "General" or "1. Check how many birds visited today"
	Text    string   // the bullet text, with [a][b]/[a](url) links turned into "a (url)"
	Links   []string // resolved URLs referenced by this bullet
}

// Doc is a named documentation link collected from the hints.
type Doc struct {
	Label string
	URL   string
}

// Hints is the parsed result.
type Hints struct {
	Items []Hint
	Docs  []Doc // all unique links, for an "open docs" action
}

var (
	sectionRe = regexp.MustCompile(`^#{1,6}\s+(.*\S)\s*$`)
	bulletRe  = regexp.MustCompile(`^\s*[-*]\s+(.*\S)\s*$`)
	// reference definition: "[getting-started-recursion]: https://..."
	refDefRe = regexp.MustCompile(`^\s*\[([^\]]+)\]:\s+(\S+)`)
	// inline link "[text](url)"
	inlineLinkRe = regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)
	// reference link "[text][ref]" or shortcut "[ref][]"
	refLinkRe = regexp.MustCompile(`\[([^\]]+)\]\[([^\]]*)\]`)
)

// ParseFile reads and parses a HINTS.md file. Returns ok=false if it's missing.
func ParseFile(path string) (Hints, bool) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Hints{}, false
	}
	h := Parse(string(raw))
	return h, len(h.Items) > 0
}

// Parse parses HINTS.md content.
func Parse(src string) Hints {
	// First pass: collect reference definitions.
	refs := map[string]string{}
	scanner := bufio.NewScanner(strings.NewReader(src))
	for scanner.Scan() {
		if m := refDefRe.FindStringSubmatch(scanner.Text()); m != nil {
			refs[strings.ToLower(m[1])] = m[2]
		}
	}

	// Second pass: sections + bullets.
	var out Hints
	seenDoc := map[string]bool{}
	section := ""

	scanner = bufio.NewScanner(strings.NewReader(src))
	for scanner.Scan() {
		line := scanner.Text()

		if m := sectionRe.FindStringSubmatch(line); m != nil {
			title := m[1]
			// Skip the top-level "Hints" heading.
			if !strings.EqualFold(title, "hints") {
				section = title
			}
			continue
		}

		if m := bulletRe.FindStringSubmatch(line); m != nil {
			text, links := resolveLinks(m[1], refs)
			out.Items = append(out.Items, Hint{Section: section, Text: text, Links: links})
			for _, u := range links {
				if !seenDoc[u] {
					seenDoc[u] = true
					out.Docs = append(out.Docs, Doc{Label: u, URL: u})
				}
			}
		}
	}
	return out
}

// resolveLinks turns inline and reference links in a bullet into readable text
// ("label (url)") and returns the list of URLs found.
func resolveLinks(s string, refs map[string]string) (string, []string) {
	var urls []string

	s = inlineLinkRe.ReplaceAllStringFunc(s, func(m string) string {
		sub := inlineLinkRe.FindStringSubmatch(m)
		urls = append(urls, sub[2])
		return sub[1] + " (" + sub[2] + ")"
	})

	s = refLinkRe.ReplaceAllStringFunc(s, func(m string) string {
		sub := refLinkRe.FindStringSubmatch(m)
		label, ref := sub[1], sub[2]
		if ref == "" {
			ref = label // shortcut form [ref][]
		}
		if url, ok := refs[strings.ToLower(ref)]; ok {
			urls = append(urls, url)
			return label + " (" + url + ")"
		}
		return label
	})

	return s, urls
}
