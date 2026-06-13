package kumo

import (
	"strings"
	"testing"
	"time"
)

func TestStoreWriteAndList(t *testing.T) {
	root := t.TempDir()
	s := NewStore(root)
	p := &Page{
		URI:       "pages://example.com/docs/intro",
		Type:      "pages/page",
		FetchedAt: time.Unix(1_700_000_000, 0).UTC(),
		URL:       "https://example.com/docs/intro",
		Host:      "example.com",
		Status:    200,
		Title:     "Intro",
		Content:   "# Intro\n\nThe body text.",
	}
	if err := s.Write(p); err != nil {
		t.Fatalf("Write: %v", err)
	}

	got, err := s.List("example.com")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("List returned %d pages, want 1", len(got))
	}
	r := got[0]
	if r.URI != p.URI || r.Title != p.Title || r.Status != p.Status {
		t.Errorf("round-trip mismatch: got %+v", r)
	}
	// Stored files are newline-terminated, so the body round-trips modulo a
	// trailing newline.
	if strings.TrimRight(r.Content, "\n") != p.Content {
		t.Errorf("Content round-trip = %q, want %q", r.Content, p.Content)
	}
}

func TestStoreFrontMatterExcludesContent(t *testing.T) {
	s := NewStore(t.TempDir())
	p := &Page{
		URI:     "pages://example.com/",
		URL:     "https://example.com/",
		Host:    "example.com",
		Status:  200,
		Content: "UNIQUE_BODY_MARKER body text",
	}
	doc, err := s.render(p)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	front, body, found := strings.Cut(string(doc), "\n---\n")
	if !found {
		t.Fatal("rendered doc has no front-matter fence")
	}
	if strings.Contains(front, "UNIQUE_BODY_MARKER") {
		t.Error("front-matter should not repeat the page content")
	}
	if !strings.Contains(body, "UNIQUE_BODY_MARKER") {
		t.Error("body should carry the page content")
	}
}

func TestStoreSkipsEmptyBody(t *testing.T) {
	root := t.TempDir()
	s := NewStore(root)
	// A page with no content (an error, a redirect, a 304) writes nothing, so a
	// re-crawl stays free of churn.
	if err := s.Write(&Page{URI: "pages://example.com/x", Host: "example.com", URL: "https://example.com/x"}); err != nil {
		t.Fatalf("Write: %v", err)
	}
	got, err := s.List("example.com")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected no stored pages for an empty body, got %d", len(got))
	}
}
