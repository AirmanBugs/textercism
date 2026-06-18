// Package testresult parses `mix test` (ExUnit) output into a clean, structured
// result. Elixir has no JSON formatter, so we parse the standard text output,
// extracting the summary and per-failure detail and discarding compile warnings,
// the ExUnit preamble, progress dots, and stacktraces.
package testresult

import (
	"bufio"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Failure is the assertion detail for a failed test.
type Failure struct {
	Location string // e.g. "test/bird_count_test.exs:11"
	Code     string // the failing assertion, e.g. "assert BirdCount.today([5]) == 5"
	Left     string // left value (if the failure reports one)
	Right    string // right value
	Message  string // a non-assertion message (e.g. "Assertion with == failed")
	Error    string // a raised exception, e.g. "(MatchError) no match of right hand side value: []"
}

// Test is one test in run order, with its outcome and (if failed) detail.
type Test struct {
	Name    string // e.g. "today/1 returns today's bird count"
	Line    string // source line, e.g. "6" (from [L#6])
	Passed  bool
	Failure Failure // populated when Passed is false
}

// Result is the parsed outcome of a test run.
type Result struct {
	Total     int
	Failures  int
	Tests     []Test // every test, in run order (from --trace)
	AllPassed bool
}

var (
	// "  1) test today/1 returns today's bird count (BirdCountTest)"
	failureHeader = regexp.MustCompile(`^\s*\d+\)\s+test\s+(.*?)\s+\([^)]*\)\s*$`)
	// "11 tests, 10 failures" (also handles excluded/skipped suffixes)
	summaryLine = regexp.MustCompile(`(\d+)\s+tests?,\s+(\d+)\s+failures?`)
	// "test/bird_count_test.exs:11"
	locationLine = regexp.MustCompile(`^\s*((?:test|lib)/\S+\.exs?:\d+)\s*$`)
	fieldLine    = regexp.MustCompile(`^\s*(code|left|right):\s*(.*)$`)
	// "     ** (MatchError) no match of right hand side value:"
	errorRe = regexp.MustCompile(`^\s*\*\*\s+(\(.*)$`)
	// "  * test today/1 returns today's bird count (0.4ms) [L#11]" (--trace)
	traceLine = regexp.MustCompile(`^\s*\* test (.*) \([0-9.]+m?s\) \[L#(\d+)\]\s*$`)
)

// Parse turns raw `mix test --trace` output into a Result.
func Parse(raw string) Result {
	var res Result

	// First, pull the summary (it's near the end and unambiguous).
	if m := summaryLine.FindStringSubmatch(raw); m != nil {
		res.Total, _ = strconv.Atoi(m[1])
		res.Failures, _ = strconv.Atoi(m[2])
	}
	res.AllPassed = res.Failures == 0

	// Walk the failure blocks into a map keyed by test name. A block starts at a
	// "N) test ..." header and ends at the stacktrace.
	failures := map[string]Failure{}
	var curName string
	var cur *Failure
	flush := func() {
		if cur != nil {
			cur.Error = collapseBlanks(strings.TrimSpace(cur.Error))
			failures[curName] = *cur
			cur, curName = nil, ""
		}
	}

	scanner := bufio.NewScanner(strings.NewReader(raw))
	scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024)

	var order []Test // tests in run order, from --trace lines
	inError := false // collecting a multi-line exception block
	for scanner.Scan() {
		line := scanner.Text()

		// --trace prints each test name twice on one physical line, joined by a
		// carriage return (first without timing, then with). Take the last \r
		// segment so we match the completed "(N ms) [L#N]" form cleanly.
		if i := strings.LastIndex(line, "\r"); i >= 0 {
			line = line[i+1:]
		}

		// --trace lists every test (passed or failed) with its line number.
		if m := traceLine.FindStringSubmatch(line); m != nil {
			order = append(order, Test{Name: strings.TrimSpace(m[1]), Line: m[2]})
			continue
		}

		if m := failureHeader.FindStringSubmatch(line); m != nil {
			flush()
			inError = false
			curName = strings.TrimSpace(m[1])
			cur = &Failure{}
			continue
		}
		if cur == nil {
			continue
		}
		if strings.TrimSpace(line) == "stacktrace:" {
			flush()
			inError = false
			continue
		}
		if cur.Location == "" {
			if m := locationLine.FindStringSubmatch(line); m != nil {
				cur.Location = m[1]
				continue
			}
		}
		// A raised exception: "** (MatchError) no match of right hand side value:".
		// Start collecting the multi-line exception block (cleaned of ExUnit's
		// underline/diff "-...-" markers) until we reach code:/stacktrace:.
		if m := errorRe.FindStringSubmatch(line); m != nil {
			inError = true
			cur.Error = cleanArtifacts(strings.TrimSpace(m[1]))
			continue
		}
		if m := fieldLine.FindStringSubmatch(line); m != nil {
			inError = false
			switch m[1] {
			case "code":
				cur.Code = strings.TrimSpace(m[2])
			case "left":
				cur.Left = strings.TrimSpace(m[2])
			case "right":
				cur.Right = strings.TrimSpace(m[2])
			}
			continue
		}
		if inError {
			// Preserve the block's line structure; dedent by the common 5-space
			// indent ExUnit uses, and clean the marker artifacts.
			cur.Error += "\n" + cleanArtifacts(dedent(line))
			continue
		}
		t := strings.TrimSpace(line)
		if strings.HasSuffix(t, "failed") && cur.Message == "" {
			cur.Message = t
		}
	}
	flush()

	// Build the ordered test list, marking each pass/fail by name.
	for _, t := range order {
		if f, failed := failures[t.Name]; failed {
			t.Passed = false
			t.Failure = f
		} else {
			t.Passed = true
		}
		res.Tests = append(res.Tests, t)
	}

	return res
}

// artifactRe matches ExUnit's diff/underline markers around code in exception
// messages, e.g. "def today(-[]-)" -> "def today([])". They come from stripped
// ANSI underline sequences.
var artifactRe = regexp.MustCompile(`-(\[[^\]]*\]|\([^)]*\)|[^-\s]+)-`)

// cleanArtifacts removes the "-...-" marker artifacts from a line.
func cleanArtifacts(s string) string {
	return artifactRe.ReplaceAllString(s, "$1")
}

// dedent removes up to 5 leading spaces (ExUnit's failure-block indent), keeping
// any deeper indentation (e.g. nested clause lists) relative.
func dedent(s string) string {
	for i := 0; i < 5; i++ {
		if strings.HasPrefix(s, " ") {
			s = s[1:]
		} else {
			break
		}
	}
	return s
}

// collapseBlanks collapses runs of blank lines into a single blank line.
func collapseBlanks(s string) string {
	var out []string
	prevBlank := false
	for _, ln := range strings.Split(s, "\n") {
		blank := strings.TrimSpace(ln) == ""
		if blank && prevBlank {
			continue
		}
		out = append(out, ln)
		prevBlank = blank
	}
	return strings.Join(out, "\n")
}

// Passed returns the number of passing tests.
func (r Result) Passed() int { return r.Total - r.Failures }

// Banner is the headline result, e.g. "✓ 11 of 11 passed" or "✗ 1 of 11 passed".
func (r Result) Banner() string {
	mark := "✓"
	if !r.AllPassed {
		mark = "✗"
	}
	return fmt.Sprintf("%s %d of %d passed", mark, r.Passed(), r.Total)
}

// Summary is a short status-line form, e.g. "1 of 11 passed".
func (r Result) Summary() string {
	return fmt.Sprintf("%d of %d passed", r.Passed(), r.Total)
}

// AssertionMarkdown renders a failed test's assertion detail (code, left, right)
// as markdown, for Glamour. Empty when there's nothing to show.
func (f Failure) AssertionMarkdown() string {
	var b strings.Builder
	if f.Location != "" {
		fmt.Fprintf(&b, "`%s`\n\n", f.Location)
	}
	if f.Code != "" {
		fmt.Fprintf(&b, "```elixir\n%s\n```\n", f.Code)
	}
	switch {
	case f.Error != "":
		// A raised exception — the code crashed rather than returning a value.
		// Render the multi-line message in a code block so its structure (the
		// attempted clauses, the given arguments) is preserved, not flattened.
		fmt.Fprintf(&b, "**error:**\n```\n%s\n```\n", f.Error)
	case f.Left != "" || f.Right != "":
		fmt.Fprintf(&b, "- your result: `%s`\n- expected:    `%s`\n", f.Left, f.Right)
	case f.Message != "" && f.Code == "":
		fmt.Fprintf(&b, "%s\n", f.Message)
	}
	return b.String()
}
