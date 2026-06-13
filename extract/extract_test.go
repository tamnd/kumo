package extract

import (
	"strings"
	"testing"
)

const sampleHTML = `<!doctype html>
<html lang="en">
<head>
  <title>The Sample Page</title>
  <meta name="description" content="A page used to exercise the extractor.">
  <meta name="author" content="Ada Lovelace">
  <meta property="og:title" content="Sample (OG)">
  <meta property="og:site_name" content="Example Docs">
  <meta property="article:published_time" content="2024-01-02T03:04:05Z">
  <link rel="canonical" href="https://example.com/guide">
  <script type="application/ld+json">{"@type":"Article","headline":"Sample"}</script>
</head>
<body>
  <nav><a href="/menu">Menu</a></nav>
  <article>
    <h1>The Sample Page</h1>
    <p>This is the main content with a <a href="/docs/next">relative link</a>
    and an <a href="https://other.example/page">absolute link</a>.</p>
    <p>A second paragraph so readability keeps the article body.</p>
  </article>
</body>
</html>`

func TestFromHTMLMetadata(t *testing.T) {
	res, err := FromHTML([]byte(sampleHTML), "text/html; charset=utf-8", "https://example.com/guide")
	if err != nil {
		t.Fatalf("FromHTML: %v", err)
	}

	checks := []struct {
		field, got, want string
	}{
		{"Title", res.Title, "The Sample Page"},
		{"Description", res.Description, "A page used to exercise the extractor."},
		{"Language", res.Language, "en"},
		{"Canonical", res.Canonical, "https://example.com/guide"},
		{"Author", res.Author, "Ada Lovelace"},
		{"SiteName", res.SiteName, "Example Docs"},
		{"Published", res.Published, "2024-01-02T03:04:05Z"},
	}
	for _, c := range checks {
		if c.got != c.want {
			t.Errorf("%s = %q, want %q", c.field, c.got, c.want)
		}
	}

	if res.OpenGraph["og:title"] != "Sample (OG)" {
		t.Errorf("OpenGraph[og:title] = %q, want %q", res.OpenGraph["og:title"], "Sample (OG)")
	}
	if len(res.JSONLD) != 1 {
		t.Errorf("JSONLD blocks = %d, want 1", len(res.JSONLD))
	}
}

func TestFromHTMLResolvesLinks(t *testing.T) {
	res, err := FromHTML([]byte(sampleHTML), "text/html", "https://example.com/guide")
	if err != nil {
		t.Fatalf("FromHTML: %v", err)
	}
	want := map[string]bool{
		"https://example.com/menu":      false,
		"https://example.com/docs/next": false,
		"https://other.example/page":    false,
	}
	for _, l := range res.Links {
		if _, ok := want[l.URL]; ok {
			want[l.URL] = true
		}
	}
	for u, seen := range want {
		if !seen {
			t.Errorf("expected resolved link %q among %v", u, res.Links)
		}
	}
}

func TestFromHTMLMarkdown(t *testing.T) {
	res, err := FromHTML([]byte(sampleHTML), "text/html", "https://example.com/guide")
	if err != nil {
		t.Fatalf("FromHTML: %v", err)
	}
	if !strings.Contains(res.Markdown, "main content") {
		t.Errorf("Markdown missing article body, got:\n%s", res.Markdown)
	}
	// The nav link sits outside the article, so readability should drop it.
	if strings.Contains(res.Markdown, "Menu") {
		t.Errorf("Markdown leaked chrome (nav), got:\n%s", res.Markdown)
	}
	if res.Source != "extracted" {
		t.Errorf("Source = %q, want %q", res.Source, "extracted")
	}
}

func TestRawMarkdownURL(t *testing.T) {
	cases := []struct{ in, want string }{
		{"https://example.com/docs/intro", "https://example.com/docs/intro.md"},
		{"https://example.com/docs/intro/", "https://example.com/docs/intro.md"},
		{"https://example.com/docs/intro.md", ""},
		{"https://example.com/", ""},
		{"https://example.com/docs?x=1", ""},
	}
	for _, c := range cases {
		if got := RawMarkdownURL(c.in); got != c.want {
			t.Errorf("RawMarkdownURL(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestLooksLikeMarkdown(t *testing.T) {
	cases := []struct {
		name string
		body string
		ct   string
		want bool
	}{
		{"markdown content type", "# Title", "text/markdown", true},
		{"plain text", "# Title", "text/plain", true},
		{"html body rejected", "<!doctype html>", "text/markdown", false},
		{"html content type rejected", "# Title", "text/html", false},
		{"empty rejected", "", "text/markdown", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := LooksLikeMarkdown([]byte(c.body), c.ct); got != c.want {
				t.Errorf("LooksLikeMarkdown(%q, %q) = %v, want %v", c.body, c.ct, got, c.want)
			}
		})
	}
}
