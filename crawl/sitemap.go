package crawl

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/xml"
	"io"
	"strings"
)

// sitemapDoc models both flavors of sitemap XML: a urlset of page locations and
// a sitemapindex of nested sitemaps. Decoding into one struct lets a single
// parser handle either without sniffing the root element first.
type sitemapDoc struct {
	URLs     []sitemapEntry `xml:"url"`
	Sitemaps []sitemapEntry `xml:"sitemap"`
}

type sitemapEntry struct {
	Loc string `xml:"loc"`
}

// discoverSitemaps collects the page URLs reachable from a host's sitemaps. It
// starts from the sitemaps robots.txt advertises plus the conventional
// /sitemap.xml, follows sitemap-index entries (bounded in depth), and returns
// the de-duplicated page locations found. Fetch errors are skipped quietly so a
// missing or malformed sitemap never fails the crawl.
func discoverSitemaps(ctx context.Context, f *fetcher, scheme, host string, robotsSitemaps []string) []string {
	seeds := append([]string{}, robotsSitemaps...)
	seeds = append(seeds, scheme+"://"+host+"/sitemap.xml")

	seen := map[string]bool{}
	var pages []string
	pageSeen := map[string]bool{}

	var visit func(url string, depth int)
	visit = func(url string, depth int) {
		if depth > 5 || url == "" || seen[url] {
			return
		}
		seen[url] = true
		res, err := f.get(ctx, url, conds{})
		if err != nil || res == nil || res.Status != 200 || len(res.Body) == 0 {
			return
		}
		doc, err := parseSitemap(res.Body, url)
		if err != nil {
			return
		}
		for _, sm := range doc.Sitemaps {
			visit(strings.TrimSpace(sm.Loc), depth+1)
		}
		for _, u := range doc.URLs {
			loc := strings.TrimSpace(u.Loc)
			if loc != "" && !pageSeen[loc] {
				pageSeen[loc] = true
				pages = append(pages, loc)
			}
		}
	}
	for _, s := range seeds {
		visit(strings.TrimSpace(s), 0)
	}
	return pages
}

// parseSitemap decodes sitemap XML, transparently decompressing a gzipped body
// (sitemaps are often served as .xml.gz).
func parseSitemap(body []byte, url string) (*sitemapDoc, error) {
	if strings.HasSuffix(url, ".gz") || isGzip(body) {
		if zr, err := gzip.NewReader(bytes.NewReader(body)); err == nil {
			if dec, err := io.ReadAll(zr); err == nil {
				body = dec
			}
			_ = zr.Close()
		}
	}
	var doc sitemapDoc
	if err := xml.Unmarshal(body, &doc); err != nil {
		return nil, err
	}
	return &doc, nil
}

func isGzip(b []byte) bool {
	return len(b) >= 2 && b[0] == 0x1f && b[1] == 0x8b
}

// Sitemap discovers the in-scope URL frontier for a host from robots.txt and
// its sitemaps, fetching no page content. It backs the "kumo sitemap" command.
func Sitemap(ctx context.Context, opt Options) ([]string, error) {
	c := New(opt)
	c.f = newFetcher(opt.HTTP, c.opt.UserAgent, opt.Retries, opt.Rate)
	var robotsSitemaps []string
	if opt.RespectRobots {
		c.robots = fetchRobots(ctx, c.f, c.scheme, c.scope.Host)
		robotsSitemaps = c.robots.sitemaps
	}
	found := discoverSitemaps(ctx, c.f, c.scheme, c.scope.Host, robotsSitemaps)
	out := make([]string, 0, len(found))
	seen := map[string]bool{}
	for _, u := range found {
		n, ok := Normalize(u)
		if !ok || !c.scope.InScope(n) || seen[n] {
			continue
		}
		seen[n] = true
		out = append(out, n)
	}
	return out, nil
}
