// Package extract turns a raw HTML response into a structured record: the page
// metadata (title, description, canonical, language, dates, OpenGraph, JSON-LD),
// the main readable content converted to Markdown, and the outbound links.
//
// It parses the document twice from the same decoded bytes: once to walk the
// whole tree for metadata and links, and once through go-readability, which
// mutates its own copy of the tree to isolate the main content. The cleaned
// content node is handed straight to html-to-markdown, so there is no lossy
// HTML-string round trip in the middle.
package extract

import (
	"bytes"
	"encoding/json"
	"io"
	"net/url"
	"strings"

	readability "codeberg.org/readeck/go-readability/v2"
	htmltomarkdown "github.com/JohannesKaufmann/html-to-markdown/v2"
	"golang.org/x/net/html"
	"golang.org/x/net/html/charset"
)

// Link is one outbound hyperlink discovered on a page, with its URL resolved to
// an absolute form.
type Link struct {
	URL    string `json:"url"`
	Anchor string `json:"anchor,omitempty"`
	Rel    string `json:"rel,omitempty"`
}

// Result is the structured form of one HTML page.
type Result struct {
	Title       string
	Description string
	Language    string
	Canonical   string
	Author      string
	SiteName    string
	Published   string
	Modified    string
	OpenGraph   map[string]string
	JSONLD      []json.RawMessage
	Links       []Link
	Markdown    string
	Source      string // how Markdown was produced: "extracted"
}

// FromHTML parses a raw HTML response into a Result. contentType is the HTTP
// Content-Type header (used for charset detection) and pageURL is the absolute
// URL of the page (used to resolve relative links and guide readability).
func FromHTML(raw []byte, contentType, pageURL string) (*Result, error) {
	base, _ := url.Parse(pageURL)

	body, err := decodeUTF8(raw, contentType)
	if err != nil {
		body = raw
	}

	doc, err := html.Parse(bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	res := &Result{OpenGraph: map[string]string{}, Source: "extracted"}
	walk(doc, base, res)

	if art, err := readability.FromReader(bytes.NewReader(body), base); err == nil {
		if art.Node != nil {
			if md, err := htmltomarkdown.ConvertNode(art.Node); err == nil {
				res.Markdown = strings.TrimSpace(string(md))
			}
		}
		fillFromArticle(res, art)
	}
	return res, nil
}

// decodeUTF8 reads raw bytes through a charset-aware reader so a non-UTF-8 page
// (declared in the Content-Type or a <meta charset>) becomes valid UTF-8.
func decodeUTF8(raw []byte, contentType string) ([]byte, error) {
	r, err := charset.NewReader(bytes.NewReader(raw), contentType)
	if err != nil {
		return nil, err
	}
	return io.ReadAll(r)
}

// fillFromArticle backfills fields readability recovers when the raw tags were
// missing or thin: the title, language, byline, site name, dates, and a content
// excerpt used as a last-resort description.
func fillFromArticle(res *Result, art readability.Article) {
	if res.Title == "" {
		res.Title = strings.TrimSpace(art.Title())
	}
	if res.Language == "" {
		res.Language = art.Language()
	}
	if res.Author == "" {
		res.Author = strings.TrimSpace(art.Byline())
	}
	if res.SiteName == "" {
		res.SiteName = strings.TrimSpace(art.SiteName())
	}
	if res.Description == "" {
		res.Description = strings.TrimSpace(art.Excerpt())
	}
	if res.Published == "" {
		if t, err := art.PublishedTime(); err == nil && !t.IsZero() {
			res.Published = t.UTC().Format("2006-01-02T15:04:05Z")
		}
	}
	if res.Modified == "" {
		if t, err := art.ModifiedTime(); err == nil && !t.IsZero() {
			res.Modified = t.UTC().Format("2006-01-02T15:04:05Z")
		}
	}
}

// walk traverses the whole parsed document collecting metadata and links.
func walk(n *html.Node, base *url.URL, res *Result) {
	if n.Type == html.ElementNode {
		switch n.Data {
		case "html":
			if v := attr(n, "lang"); v != "" && res.Language == "" {
				res.Language = v
			}
		case "title":
			if res.Title == "" {
				res.Title = strings.TrimSpace(text(n))
			}
		case "meta":
			meta(n, res)
		case "link":
			if relHas(attr(n, "rel"), "canonical") {
				res.Canonical = resolve(base, attr(n, "href"))
			}
		case "script":
			if strings.EqualFold(attr(n, "type"), "application/ld+json") {
				if raw := strings.TrimSpace(text(n)); raw != "" && json.Valid([]byte(raw)) {
					res.JSONLD = append(res.JSONLD, json.RawMessage(raw))
				}
			}
		case "a":
			if u := resolve(base, attr(n, "href")); u != "" {
				res.Links = append(res.Links, Link{
					URL:    u,
					Anchor: strings.TrimSpace(text(n)),
					Rel:    attr(n, "rel"),
				})
			}
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		walk(c, base, res)
	}
}

// meta reads one <meta> tag into the result: standard names, OpenGraph and
// Twitter properties, and the article published/modified timestamps.
func meta(n *html.Node, res *Result) {
	content := attr(n, "content")
	if content == "" {
		return
	}
	name := strings.ToLower(attr(n, "name"))
	prop := strings.ToLower(attr(n, "property"))

	switch name {
	case "description":
		if res.Description == "" {
			res.Description = content
		}
	case "author":
		if res.Author == "" {
			res.Author = content
		}
	}
	switch prop {
	case "og:title":
		if res.Title == "" {
			res.Title = content
		}
	case "og:description":
		if res.Description == "" {
			res.Description = content
		}
	case "og:site_name":
		res.SiteName = content
	case "article:published_time":
		res.Published = content
	case "article:modified_time":
		res.Modified = content
	}
	if strings.HasPrefix(prop, "og:") {
		res.OpenGraph[prop] = content
	}
}

func attr(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if strings.EqualFold(a.Key, key) {
			return a.Val
		}
	}
	return ""
}

// text returns the concatenated text content of a node and its descendants,
// collapsed to single spaces.
func text(n *html.Node) string {
	var b strings.Builder
	var rec func(*html.Node)
	rec = func(n *html.Node) {
		if n.Type == html.TextNode {
			b.WriteString(n.Data)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			rec(c)
		}
	}
	rec(n)
	return strings.Join(strings.Fields(b.String()), " ")
}

// resolve turns a possibly relative href into an absolute http(s) URL against
// the page base, returning "" for empty, fragment-only, or non-http links.
func resolve(base *url.URL, href string) string {
	href = strings.TrimSpace(href)
	if href == "" || strings.HasPrefix(href, "#") {
		return ""
	}
	switch {
	case strings.HasPrefix(href, "javascript:"),
		strings.HasPrefix(href, "mailto:"),
		strings.HasPrefix(href, "tel:"),
		strings.HasPrefix(href, "data:"):
		return ""
	}
	u, err := url.Parse(href)
	if err != nil {
		return ""
	}
	if base != nil {
		u = base.ResolveReference(u)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return ""
	}
	u.Fragment = ""
	return u.String()
}

func relHas(rel, want string) bool {
	for f := range strings.FieldsSeq(rel) {
		if strings.EqualFold(f, want) {
			return true
		}
	}
	return false
}
