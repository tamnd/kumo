// Package crawl is the single-host crawl engine behind kumo. It owns the
// frontier, the polite fetcher, robots and sitemap handling, URL normalization
// and scoping, and the worker loop that turns a host into a stream of structured
// pages. It depends on the extract package for HTML-to-structured conversion and
// on nothing in the kumo package, so the engine stays reusable on its own.
package crawl

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/tamnd/kumo/extract"
)

// Options configures one crawl. Only Host is required; the zero value of every
// other field is a sensible default applied by New.
type Options struct {
	Host          string
	Seeds         []string // extra seed URLs beyond the host root
	Workers       int
	MaxPages      int    // 0 = unlimited
	MaxDepth      int    // 0 = unlimited
	Traversal     string // "bfs" (default) or "dfs"
	IncludeSubs   bool
	ScopePrefix   string
	Exclude       []string
	IncludeSearch bool
	RespectRobots bool
	FollowSitemap bool
	ProbeRawMD    bool
	UserAgent     string
	Rate          time.Duration
	Retries       int
	HTTP          *http.Client
	DryRun        bool

	// Known returns the stored validators for a URL so a re-crawl can issue a
	// conditional GET. It may be nil, in which case every fetch is unconditional.
	Known func(url string) (etag, lastModified string)
}

// Page is one crawled page as the engine sees it: the fetch outcome plus the
// extracted structured content. The kumo package maps this onto its own URI
// addressed record.
type Page struct {
	URL          string
	Status       int
	Depth        int
	FetchedAt    time.Time
	ContentType  string
	ETag         string
	LastModified string
	NotModified  bool
	Extract      *extract.Result
	RawMarkdown  string
	Error        string
}

// Stats summarizes a finished crawl.
type Stats struct {
	Fetched int
	Pages   int
	RawMD   int
	Errors  int
	Skipped int
	Elapsed time.Duration
}

// Crawler runs one crawl. Build it with New and drive it with Run.
type Crawler struct {
	opt    Options
	scheme string
	scope  Scope
	fr     *frontier
	f      *fetcher
	robots *robotsPolicy

	emitMu  sync.Mutex
	emit    func(*Page) error
	count   int
	capped  bool
	emitErr error

	statsMu sync.Mutex
	stats   Stats
}

// New builds a Crawler from options, applying defaults.
func New(opt Options) *Crawler {
	if opt.Workers <= 0 {
		opt.Workers = 16
	}
	if opt.Traversal == "" {
		opt.Traversal = "bfs"
	}
	if opt.UserAgent == "" {
		opt.UserAgent = "kumo/dev (+https://github.com/tamnd/kumo)"
	}
	if opt.HTTP == nil {
		opt.HTTP = &http.Client{Timeout: 30 * time.Second}
	}
	host := strings.ToLower(strings.TrimSpace(opt.Host))
	return &Crawler{
		opt:    opt,
		scheme: "https",
		scope: Scope{
			Host:           host,
			IncludeSubs:    opt.IncludeSubs,
			Prefix:         opt.ScopePrefix,
			Exclude:        opt.Exclude,
			IncludeSearch:  opt.IncludeSearch,
			CollapseLocale: true,
		},
	}
}

// Run crawls the host, calling emit for each page. emit is serialized, so it
// need not be safe for concurrent use. It returns when the frontier drains, the
// page cap is reached, or the context is cancelled.
func (c *Crawler) Run(ctx context.Context, emit func(*Page) error) (Stats, error) {
	start := time.Now()
	c.emit = emit
	c.f = newFetcher(c.opt.HTTP, c.opt.UserAgent, c.opt.Retries, c.opt.Rate)

	if c.opt.RespectRobots {
		c.robots = fetchRobots(ctx, c.f, c.scheme, c.scope.Host)
		if c.robots.delay > c.opt.Rate {
			// Re-pace the fetcher to honor a stricter robots crawl-delay.
			c.f = newFetcher(c.opt.HTTP, c.opt.UserAgent, c.opt.Retries, c.robots.delay)
		}
	}

	c.fr = newFrontier(c.opt.Traversal == "dfs", c.scope.DedupKey)
	c.seed(ctx)

	if c.opt.DryRun {
		c.drainDry()
		c.stats.Elapsed = time.Since(start)
		return c.stats, c.emitErr
	}

	var wg sync.WaitGroup
	for range c.opt.Workers {
		wg.Go(func() { c.worker(ctx) })
	}
	wg.Wait()

	c.stats.Elapsed = time.Since(start)
	return c.stats, c.emitErr
}

// seed primes the frontier with the host root, any extra seeds, and the sitemap
// URLs in scope.
func (c *Crawler) seed(ctx context.Context) {
	root := c.scheme + "://" + c.scope.Host + "/"
	c.addURL(root, 0)
	for _, s := range c.opt.Seeds {
		c.addURL(s, 0)
	}
	if c.opt.FollowSitemap {
		var robotsSitemaps []string
		if c.robots != nil {
			robotsSitemaps = c.robots.sitemaps
		}
		for _, u := range discoverSitemaps(ctx, c.f, c.scheme, c.scope.Host, robotsSitemaps) {
			c.addURL(u, 1)
		}
	}
}

// addURL normalizes, scopes, and enqueues a candidate URL.
func (c *Crawler) addURL(raw string, depth int) {
	norm, ok := Normalize(raw)
	if !ok || !c.scope.InScope(norm) {
		return
	}
	c.fr.Add(norm, depth)
}

// worker pulls the frontier until it closes.
func (c *Crawler) worker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			c.fr.Close()
			return
		default:
		}
		it, ok := c.fr.Next()
		if !ok {
			return
		}
		c.process(ctx, it)
		c.fr.Done()
	}
}

// process fetches one URL, extracts it, enqueues its in-scope links, and emits
// the page.
func (c *Crawler) process(ctx context.Context, it item) {
	if c.robots != nil && !c.robots.Allowed(it.URL) {
		c.bumpSkipped()
		return
	}

	var cd conds
	if c.opt.Known != nil {
		cd.ETag, cd.LastModified = c.opt.Known(it.URL)
	}

	res, err := c.f.get(ctx, it.URL, cd)
	c.bumpFetched()
	if err != nil {
		c.bumpErrors()
		c.send(&Page{URL: it.URL, Depth: it.Depth, FetchedAt: time.Now(), Error: err.Error()})
		return
	}

	page := &Page{
		URL:          res.URL,
		Status:       res.Status,
		Depth:        it.Depth,
		FetchedAt:    time.Now(),
		ContentType:  res.ContentType,
		ETag:         res.ETag,
		LastModified: res.LastModified,
		NotModified:  res.NotModified,
	}
	if res.NotModified {
		c.send(page)
		return
	}
	if res.Status != 200 {
		c.bumpSkipped()
		c.send(page)
		return
	}
	if !isHTML(res.ContentType, res.Body) {
		c.bumpSkipped()
		return
	}

	if r, err := extract.FromHTML(res.Body, res.ContentType, res.URL); err == nil {
		page.Extract = r
		c.bumpPages()
		c.enqueueLinks(it.Depth, r.Links)
	}
	if c.opt.ProbeRawMD {
		c.probeRawMD(ctx, page)
	}
	c.send(page)
}

// enqueueLinks adds the in-scope links discovered on a page, respecting the
// depth cap.
func (c *Crawler) enqueueLinks(depth int, links []extract.Link) {
	if c.opt.MaxDepth > 0 && depth >= c.opt.MaxDepth {
		return
	}
	for _, l := range links {
		c.addURL(l.URL, depth+1)
	}
}

// probeRawMD attempts the "<page>.md" sibling fetch and attaches it when the
// body is plausibly Markdown.
func (c *Crawler) probeRawMD(ctx context.Context, page *Page) {
	cand := extract.RawMarkdownURL(page.URL)
	if cand == "" {
		return
	}
	rr, err := c.f.get(ctx, cand, conds{})
	if err != nil || rr == nil || rr.Status != 200 {
		return
	}
	if extract.LooksLikeMarkdown(rr.Body, rr.ContentType) {
		page.RawMarkdown = string(rr.Body)
		c.bumpRawMD()
	}
}

// drainDry empties the frontier without fetching, emitting one stub page per
// URL so a dry run reports exactly what a real crawl would seed.
func (c *Crawler) drainDry() {
	for {
		it, ok := c.fr.Next()
		if !ok {
			return
		}
		c.send(&Page{URL: it.URL, Depth: it.Depth})
		c.fr.Done()
	}
}

// send serializes emit, enforces the page cap, and closes the frontier once the
// cap is hit.
func (c *Crawler) send(p *Page) {
	c.emitMu.Lock()
	defer c.emitMu.Unlock()
	if c.capped {
		return
	}
	if err := c.emit(p); err != nil {
		c.emitErr = err
		c.capped = true
		c.fr.Close()
		return
	}
	c.count++
	if c.opt.MaxPages > 0 && c.count >= c.opt.MaxPages {
		c.capped = true
		c.fr.Close()
	}
}

func (c *Crawler) bumpFetched() { c.statsMu.Lock(); c.stats.Fetched++; c.statsMu.Unlock() }
func (c *Crawler) bumpPages()   { c.statsMu.Lock(); c.stats.Pages++; c.statsMu.Unlock() }
func (c *Crawler) bumpRawMD()   { c.statsMu.Lock(); c.stats.RawMD++; c.statsMu.Unlock() }
func (c *Crawler) bumpErrors()  { c.statsMu.Lock(); c.stats.Errors++; c.statsMu.Unlock() }
func (c *Crawler) bumpSkipped() { c.statsMu.Lock(); c.stats.Skipped++; c.statsMu.Unlock() }

// isHTML decides whether a response is HTML worth extracting, from the
// Content-Type and, when that is missing, a sniff of the body.
func isHTML(contentType string, body []byte) bool {
	ct := strings.ToLower(contentType)
	if strings.Contains(ct, "html") {
		return true
	}
	if ct != "" && !strings.Contains(ct, "xml") {
		return false
	}
	head := strings.ToLower(strings.TrimSpace(string(body[:min(len(body), 512)])))
	return strings.Contains(head, "<html") || strings.Contains(head, "<!doctype html")
}

// One fetches and structures a single URL outside a crawl, the pointwise case
// behind "kumo page". It applies the same fetch and extraction path.
func One(ctx context.Context, opt Options, target string) (*Page, error) {
	norm, ok := Normalize(target)
	if !ok {
		return nil, &url.Error{Op: "parse", URL: target, Err: errBadURL}
	}
	c := New(opt)
	c.f = newFetcher(opt.HTTP, c.opt.UserAgent, opt.Retries, opt.Rate)
	res, err := c.f.get(ctx, norm, conds{})
	if err != nil {
		return nil, err
	}
	page := &Page{
		URL: res.URL, Status: res.Status, FetchedAt: time.Now(),
		ContentType: res.ContentType, ETag: res.ETag, LastModified: res.LastModified,
	}
	if res.Status == 200 && isHTML(res.ContentType, res.Body) {
		if r, err := extract.FromHTML(res.Body, res.ContentType, res.URL); err == nil {
			page.Extract = r
		}
		if opt.ProbeRawMD {
			c.probeRawMD(ctx, page)
		}
	}
	return page, nil
}

type urlErr string

func (e urlErr) Error() string { return string(e) }

const errBadURL = urlErr("not an absolute http(s) URL")
