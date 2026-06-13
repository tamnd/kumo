package extract

import (
	"bytes"
	"net/url"
	"strings"
)

// RawMarkdownURL returns the URL of the Markdown sibling a documentation site
// may serve next to an HTML page, "<page>.md". Many modern docs frameworks
// (Mintlify, fumadocs, and others) publish a canonical Markdown source at this
// address. It returns "" when the page already points at a .md file or carries
// a query string, where the convention does not apply.
func RawMarkdownURL(pageURL string) string {
	u, err := url.Parse(pageURL)
	if err != nil || u.RawQuery != "" {
		return ""
	}
	p := strings.TrimSuffix(u.Path, "/")
	if p == "" || strings.HasSuffix(p, ".md") {
		return ""
	}
	u.Path = p + ".md"
	u.Fragment = ""
	return u.String()
}

// LooksLikeMarkdown reports whether a fetched body is plausibly Markdown rather
// than an HTML page served with a soft 200. It accepts a text/markdown or
// text/plain content type and rejects a body that opens with an HTML tag.
func LooksLikeMarkdown(body []byte, contentType string) bool {
	ct := strings.ToLower(contentType)
	if strings.Contains(ct, "html") {
		return false
	}
	trimmed := bytes.TrimSpace(body)
	if len(trimmed) == 0 {
		return false
	}
	if trimmed[0] == '<' {
		return false
	}
	if strings.Contains(ct, "markdown") || strings.Contains(ct, "text/plain") || ct == "" {
		return true
	}
	return false
}
