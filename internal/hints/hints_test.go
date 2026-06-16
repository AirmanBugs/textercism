package hints

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseHintsFile(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "bird-count-hints.md"))
	if err != nil {
		t.Fatal(err)
	}
	h := Parse(string(raw))

	if len(h.Items) == 0 {
		t.Fatal("no hints parsed")
	}

	// First hint is in the General section and resolves its reference link.
	first := h.Items[0]
	if first.Section != "General" {
		t.Errorf("first section = %q, want General", first.Section)
	}
	if !strings.Contains(first.Text, "https://hexdocs.pm/elixir/recursion.html") {
		t.Errorf("first hint did not resolve its doc link:\n%s", first.Text)
	}
	if len(first.Links) == 0 {
		t.Error("first hint has no links")
	}

	// A later section is captured (the numbered tasks).
	foundTask := false
	for _, it := range h.Items {
		if strings.HasPrefix(it.Section, "1.") {
			foundTask = true
		}
	}
	if !foundTask {
		t.Error("did not capture a numbered task section")
	}

	// Docs are de-duplicated and non-empty.
	if len(h.Docs) == 0 {
		t.Error("expected collected docs")
	}
	seen := map[string]bool{}
	for _, d := range h.Docs {
		if seen[d.URL] {
			t.Errorf("duplicate doc URL: %s", d.URL)
		}
		seen[d.URL] = true
	}
}

func TestResolveInlineLink(t *testing.T) {
	h := Parse("## General\n- See [the docs](https://example.com/x) for more.\n")
	if len(h.Items) != 1 {
		t.Fatalf("got %d items", len(h.Items))
	}
	if !strings.Contains(h.Items[0].Text, "the docs (https://example.com/x)") {
		t.Errorf("inline link not resolved: %q", h.Items[0].Text)
	}
}
