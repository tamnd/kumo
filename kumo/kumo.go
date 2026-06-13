// Package kumo is the library behind the kumo command line. It crawls a whole
// host, converts every page to a structured record, and writes those records
// into a navigable URI tree under the data directory.
//
// The Engine is the spine every command shares: it holds the polite HTTP client
// and the crawl defaults, builds a crawl for a host, and maps each crawled page
// onto a URI-addressed Page record. The crawl mechanics live in the crawl
// package and the HTML-to-structured conversion in the extract package; this
// package is the domain glue and the store.
package kumo

import (
	"context"
	"net/http"
	"time"

	"github.com/tamnd/kumo/crawl"
)

// DefaultUserAgent identifies the crawler. An honest User-Agent is both polite
// and the thing most likely to keep a crawl unblocked.
const DefaultUserAgent = "kumo/dev (+https://github.com/tamnd/kumo)"

// Engine carries the resolved configuration and the shared HTTP client. One is
// built per run from the kit config and injected into the operation handlers.
type Engine struct {
	HTTP      *http.Client
	UserAgent string
	Rate      time.Duration
	Retries   int
	Workers   int
	DataRoot  string // root of the URI tree, e.g. $HOME/data
	DryRun    bool
}

// NewEngine returns an Engine with library defaults, for use outside kit.
func NewEngine() *Engine {
	return &Engine{
		HTTP:      &http.Client{Timeout: 30 * time.Second},
		UserAgent: DefaultUserAgent,
		Rate:      time.Second,
		Retries:   3,
		Workers:   16,
		DataRoot:  DefaultDataRoot(),
	}
}

// ScrapeOptions are the per-invocation knobs for a crawl, layered on top of the
// engine defaults.
type ScrapeOptions struct {
	MaxPages      int
	MaxDepth      int
	Traversal     string
	IncludeSubs   bool
	ScopePrefix   string
	Exclude       []string
	IncludeSearch bool
	NoRobots      bool
	NoSitemap     bool
	RawMD         bool
}

// crawlOptions folds the engine defaults and the per-call options into the
// crawl.Options the engine package consumes.
func (e *Engine) crawlOptions(host string, o ScrapeOptions) crawl.Options {
	return crawl.Options{
		Host:          host,
		Workers:       e.Workers,
		MaxPages:      o.MaxPages,
		MaxDepth:      o.MaxDepth,
		Traversal:     o.Traversal,
		IncludeSubs:   o.IncludeSubs,
		ScopePrefix:   o.ScopePrefix,
		Exclude:       o.Exclude,
		IncludeSearch: o.IncludeSearch,
		RespectRobots: !o.NoRobots,
		FollowSitemap: !o.NoSitemap,
		ProbeRawMD:    o.RawMD,
		UserAgent:     e.UserAgent,
		Rate:          e.Rate,
		Retries:       e.Retries,
		HTTP:          e.HTTP,
		DryRun:        e.DryRun,
	}
}

// Scrape crawls the host and calls emit for each page record as it lands. The
// records are also written into the URI tree under the engine data root. emit
// is serialized by the crawler, so it need not be safe for concurrent use.
func (e *Engine) Scrape(ctx context.Context, host string, o ScrapeOptions, emit func(*Page) error) (crawl.Stats, error) {
	store := NewStore(e.DataRoot)
	c := crawl.New(e.crawlOptions(host, o))
	return c.Run(ctx, func(cp *crawl.Page) error {
		p := newPage(host, cp)
		if !e.DryRun {
			if err := store.Write(p); err != nil {
				return err
			}
		}
		return emit(p)
	})
}

// Page fetches and structures a single URL, the pointwise case behind the
// "page" command. It does not write the tree.
func (e *Engine) Page(ctx context.Context, target string) (*Page, error) {
	cp, err := crawl.One(ctx, e.crawlOptions("", ScrapeOptions{RawMD: true}), target)
	if err != nil {
		return nil, err
	}
	return newPage(hostOf(target), cp), nil
}

// Sitemap discovers the in-scope URL frontier for a host without fetching pages.
func (e *Engine) Sitemap(ctx context.Context, host string, o ScrapeOptions) ([]string, error) {
	return crawl.Sitemap(ctx, e.crawlOptions(host, o))
}

// StoredPages reads the data tree for a host and returns the page records
// already crawled, offline.
func (e *Engine) StoredPages(host string) ([]*Page, error) {
	return NewStore(e.DataRoot).List(host)
}
