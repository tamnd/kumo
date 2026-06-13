package kumo

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/url"
	"strings"
	"time"

	"github.com/tamnd/kumo/crawl"
	"github.com/tamnd/kumo/extract"
)

// Page is one crawled page as a URI-addressed structured record. It is the unit
// every kumo surface emits and the unit written into the data tree. The kit:"id"
// tag makes the URI the store key; the table tags shape the default terminal
// output. The converted Markdown rides along as Content, so json and jsonl
// output carry the page text; the store writes it as the document body instead
// of repeating it in the front-matter.
type Page struct {
	URI         string            `json:"@id" table:"-" kit:"id"`
	Type        string            `json:"@type" table:"-"`
	FetchedAt   time.Time         `json:"@fetched" table:"fetched,time"`
	URL         string            `json:"url" table:"url,url"`
	Host        string            `json:"host" table:"-"`
	Status      int               `json:"status" table:"status"`
	Title       string            `json:"title,omitempty" table:"title,truncate"`
	Description string            `json:"description,omitempty" table:"-"`
	Language    string            `json:"lang,omitempty" table:"lang"`
	Canonical   string            `json:"canonical,omitempty" table:"-"`
	Author      string            `json:"author,omitempty" table:"-"`
	SiteName    string            `json:"site_name,omitempty" table:"-"`
	Published   string            `json:"published,omitempty" table:"-"`
	Modified    string            `json:"modified,omitempty" table:"-"`
	OpenGraph   map[string]string `json:"og,omitempty" table:"-"`
	JSONLD      []json.RawMessage `json:"jsonld,omitempty" table:"-"`
	Links       []Link            `json:"links,omitempty" table:"links"`
	Hash        string            `json:"hash,omitempty" table:"-"`
	Source      string            `json:"source,omitempty" table:"-"`
	Error       string            `json:"error,omitempty" table:"-"`
	Content     string            `json:"content,omitempty" table:"-"`
}

// Link is one outbound reference from a page. On-host targets are minted to a
// pages:// URI so the data tree is a real graph; off-host targets keep their
// absolute URL.
type Link struct {
	URI    string `json:"uri"`
	Anchor string `json:"anchor,omitempty"`
	Rel    string `json:"rel,omitempty"`
}

// newPage maps an engine-level crawl.Page onto the URI-addressed record,
// minting the page URI, resolving links, and hashing the body.
func newPage(host string, cp *crawl.Page) *Page {
	p := &Page{
		Type:      "pages/page",
		FetchedAt: cp.FetchedAt,
		URL:       cp.URL,
		Host:      host,
		Status:    cp.Status,
		Error:     cp.Error,
	}
	p.URI = MintURI(host, cp.URL)
	if cp.NotModified {
		p.Type = "pages/unchanged"
	}
	if cp.Extract != nil {
		e := cp.Extract
		p.Title = e.Title
		p.Description = e.Description
		p.Language = e.Language
		p.Canonical = e.Canonical
		p.Author = e.Author
		p.SiteName = e.SiteName
		p.Published = e.Published
		p.Modified = e.Modified
		if len(e.OpenGraph) > 0 {
			p.OpenGraph = e.OpenGraph
		}
		p.JSONLD = e.JSONLD
		p.Links = mintLinks(host, e.Links)
		p.Source = e.Source
		p.Content = e.Markdown
	}
	if cp.RawMarkdown != "" {
		// The site's own canonical Markdown wins over the extracted body.
		p.Content = cp.RawMarkdown
		p.Source = "raw-md"
	}
	if p.Content != "" {
		sum := sha256.Sum256([]byte(p.Content))
		p.Hash = hex.EncodeToString(sum[:])
	}
	return p
}

// mintLinks converts discovered links to records, minting on-host targets to
// pages:// URIs and leaving off-host targets as absolute URLs.
func mintLinks(host string, links []extract.Link) []Link {
	if len(links) == 0 {
		return nil
	}
	out := make([]Link, 0, len(links))
	seen := map[string]bool{}
	for _, l := range links {
		uri := l.URL
		if u, err := url.Parse(l.URL); err == nil && sameHost(u.Hostname(), host) {
			uri = MintURI(host, l.URL)
		}
		if seen[uri] {
			continue
		}
		seen[uri] = true
		out = append(out, Link{URI: uri, Anchor: l.Anchor, Rel: l.Rel})
	}
	return out
}

func sameHost(a, want string) bool {
	a = strings.TrimPrefix(strings.ToLower(a), "www.")
	want = strings.TrimPrefix(strings.ToLower(want), "www.")
	return a == want || strings.HasSuffix(a, "."+want)
}
