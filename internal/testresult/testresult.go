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

// Failure is one failed test, with the high-value detail.
type Failure struct {
	Name     string // e.g. "today/1 returns today's bird count"
	Location string // e.g. "test/bird_count_test.exs:11"
	Code     string // the failing assertion, e.g. "assert BirdCount.today([5]) == 5"
	Left     string // left value (if the failure reports one)
	Right    string // right value
	Message  string // a non-assertion message, when there's no left/right
}

// Result is the parsed outcome of a test run.
type Result struct {
	Total    int
	Failures int
	Failed   []Failure
	Passed   bool
}

var (
	// "  1) test today/1 returns today's bird count (BirdCountTest)"
	failureHeader = regexp.MustCompile(`^\s*\d+\)\s+test\s+(.*?)\s+\([^)]*\)\s*$`)
	// "11 tests, 10 failures" (also handles "1 test, 0 failures", excluded/skipped suffixes)
	summaryLine = regexp.MustCompile(`(\d+)\s+tests?,\s+(\d+)\s+failures?`)
	// "test/bird_count_test.exs:11"
	locationLine = regexp.MustCompile(`^\s*((?:test|lib)/\S+\.exs?:\d+)\s*$`)
	fieldLine    = regexp.MustCompile(`^\s*(code|left|right):\s*(.*)$`)
)

// Parse turns raw `mix test` output into a Result.
func Parse(raw string) Result {
	var res Result

	// First, pull the summary (it's near the end and unambiguous).
	if m := summaryLine.FindStringSubmatch(raw); m != nil {
		res.Total, _ = strconv.Atoi(m[1])
		res.Failures, _ = strconv.Atoi(m[2])
	}
	res.Passed = res.Failures == 0

	// Then walk the failure blocks. A block starts at a "N) test ..." header and
	// runs until the next header, a summary, or a blank-line gap followed by
	// progress output.
	scanner := bufio.NewScanner(strings.NewReader(raw))
	scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024)

	var cur *Failure
	flush := func() {
		if cur != nil {
			res.Failed = append(res.Failed, *cur)
			cur = nil
		}
	}

	for scanner.Scan() {
		line := scanner.Text()

		if m := failureHeader.FindStringSubmatch(line); m != nil {
			flush()
			cur = &Failure{Name: strings.TrimSpace(m[1])}
			continue
		}
		if cur == nil {
			continue
		}

		// Stacktrace ends the useful part of a block.
		if strings.TrimSpace(line) == "stacktrace:" {
			flush()
			continue
		}
		if cur.Location == "" {
			if m := locationLine.FindStringSubmatch(line); m != nil {
				cur.Location = m[1]
				continue
			}
		}
		if m := fieldLine.FindStringSubmatch(line); m != nil {
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
		// A line like "Assertion with == failed" or "match (=) failed" is a
		// message we keep only if there's no structured code/left/right.
		if t := strings.TrimSpace(line); strings.HasSuffix(t, "failed") && cur.Message == "" {
			cur.Message = t
		}
	}
	flush()

	return res
}

// Summary is a one-line result for the footer/status, e.g. "✓ 8 passed" or
// "✗ 10 of 11 failed".
func (r Result) Summary() string {
	if r.Passed {
		return fmt.Sprintf("✓ %d passed", r.Total)
	}
	return fmt.Sprintf("✗ %d of %d failed", r.Failures, r.Total)
}

// Markdown renders the result as clean markdown for the instructions pane: a
// banner, then each failed test with its location and assertion detail.
func (r Result) Markdown() string {
	var b strings.Builder

	if r.Passed {
		fmt.Fprintf(&b, "# ✓ All %d tests passed\n", r.Total)
		return b.String()
	}

	fmt.Fprintf(&b, "# ✗ %d of %d tests failed\n\n", r.Failures, r.Total)

	for _, f := range r.Failed {
		fmt.Fprintf(&b, "### %s\n", f.Name)
		if f.Location != "" {
			fmt.Fprintf(&b, "`%s`\n\n", f.Location)
		}
		switch {
		case f.Code != "":
			fmt.Fprintf(&b, "```elixir\n%s\n```\n", f.Code)
			if f.Left != "" || f.Right != "" {
				fmt.Fprintf(&b, "- left:  `%s`\n- right: `%s`\n", f.Left, f.Right)
			}
		case f.Message != "":
			fmt.Fprintf(&b, "%s\n", f.Message)
		}
		b.WriteString("\n")
	}

	return b.String()
}
