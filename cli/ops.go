package cli

import (
	"context"
	"net/url"
	"strings"

	"github.com/tamnd/any-cli/kit"
	"github.com/tamnd/kumo/kumo"
)

// registerOps registers every kumo command as a kit operation. Each declares
// its inputs and emits typed records; kit renders them in every format, applies
// --limit, tees them into --db, and exposes them over serve and mcp.
func registerOps(app *kit.App) {
	registerScrape(app)
	registerPage(app)
	registerSitemap(app)
	registerLinks(app)
	registerPages(app)
}

// scrapeIn is the whole-host crawl. Out is a stream of *kumo.Page records.
type scrapeIn struct {
	Engine        *kumo.Engine `kit:"inject"`
	Host          string       `kit:"arg" help:"host to crawl, e.g. example.com"`
	MaxPages      int          `kit:"flag,name=max-pages" help:"stop after N pages (0 = unlimited)"`
	MaxDepth      int          `kit:"flag,name=max-depth" help:"link-depth cap from the seed (0 = unlimited)"`
	Traversal     string       `kit:"flag" enum:"bfs,dfs" default:"bfs" help:"frontier order: bfs or dfs"`
	Subdomains    bool         `kit:"flag,name=include-subdomains" help:"also crawl subdomains of the host"`
	ScopePrefix   string       `kit:"flag,name=scope-prefix" help:"restrict to URLs whose path starts with this"`
	Exclude       []string     `kit:"flag,name=exclude-path" help:"skip URLs whose path contains this (repeatable)"`
	IncludeSearch bool         `kit:"flag,name=include-search" help:"crawl site-search routes too"`
	NoRobots      bool         `kit:"flag,name=no-robots" help:"do not fetch or honor robots.txt"`
	NoSitemap     bool         `kit:"flag,name=no-sitemap" help:"do not seed the frontier from sitemaps"`
	RawMD         bool         `kit:"flag,name=raw-md" help:"probe for a <page>.md Markdown sibling on docs sites"`
}

func (in scrapeIn) options() kumo.ScrapeOptions {
	return kumo.ScrapeOptions{
		MaxPages:      in.MaxPages,
		MaxDepth:      in.MaxDepth,
		Traversal:     in.Traversal,
		IncludeSubs:   in.Subdomains,
		ScopePrefix:   in.ScopePrefix,
		Exclude:       in.Exclude,
		IncludeSearch: in.IncludeSearch,
		NoRobots:      in.NoRobots,
		NoSitemap:     in.NoSitemap,
		RawMD:         in.RawMD,
	}
}

func registerScrape(app *kit.App) {
	kit.Handle(app, kit.OpMeta{
		Name:    "scrape",
		Group:   "crawl",
		Aliases: []string{"domain", "site", "crawl"},
		Summary: "Crawl a whole host into structured page records",
		Long: `Crawl every page of a host, convert each to a structured record, and write
the result into the data tree as pages://<host>/<path>.

The crawl is scoped to one host. It seeds from the host root, robots.txt, and
the sitemaps, then follows in-scope links breadth-first. robots.txt and its
crawl-delay are honored by default; bound the crawl with --max-pages, --max-depth,
--scope-prefix, and --exclude-path.

Examples:
  kumo scrape example.com
  kumo scrape example.com --max-pages 500 --scope-prefix /docs
  kumo scrape example.com --include-subdomains -o jsonl
  kumo scrape example.com --dry-run            list the seed frontier, fetch nothing`,
		Args: []kit.Arg{{Name: "host", Help: "host to crawl, e.g. example.com"}},
	}, func(ctx context.Context, in scrapeIn, emit func(*kumo.Page) error) error {
		host := normalizeHost(in.Host)
		if host == "" {
			return usageErr("a host is required, e.g. example.com")
		}
		_, err := in.Engine.Scrape(ctx, host, in.options(), emit)
		return err
	})
}

// pageIn fetches and structures a single URL.
type pageIn struct {
	Engine *kumo.Engine `kit:"inject"`
	URL    string       `kit:"arg" help:"the page URL to fetch"`
}

func registerPage(app *kit.App) {
	kit.Handle(app, kit.OpMeta{
		Name:    "page",
		Group:   "read",
		Single:  true,
		Summary: "Fetch and structure a single page",
		Long: `Fetch one URL and emit it as a structured record: title, description,
canonical, language, dates, OpenGraph, JSON-LD, outbound links, and the main
content as Markdown under "content". This is the pointwise form of scrape and
writes no tree.

Examples:
  kumo page https://example.com/docs/intro
  kumo page https://example.com/docs/intro -o jsonl | jq -r .content`,
		Args: []kit.Arg{{Name: "url", Help: "the page URL to fetch"}},
	}, func(ctx context.Context, in pageIn, emit func(*kumo.Page) error) error {
		if strings.TrimSpace(in.URL) == "" {
			return usageErr("a page URL is required")
		}
		p, err := in.Engine.Page(ctx, in.URL)
		if err != nil {
			return err
		}
		return emit(p)
	})
}

// frontierURL is one discoverable URL emitted by the sitemap command.
type frontierURL struct {
	URL string `json:"url" table:"url,url" kit:"id"`
}

type sitemapIn struct {
	Engine     *kumo.Engine `kit:"inject"`
	Host       string       `kit:"arg" help:"host whose frontier to discover"`
	Subdomains bool         `kit:"flag,name=include-subdomains" help:"include subdomains of the host"`
	NoRobots   bool         `kit:"flag,name=no-robots" help:"do not read robots.txt for sitemap directives"`
}

func registerSitemap(app *kit.App) {
	kit.Handle(app, kit.OpMeta{
		Name:    "sitemap",
		Group:   "read",
		Summary: "Discover the crawlable URL frontier without fetching pages",
		Long: `Discover the URLs a crawl would start from, by reading robots.txt and the
host's sitemaps. No page content is fetched.

Examples:
  kumo sitemap example.com
  kumo sitemap example.com -o url | head`,
		Args: []kit.Arg{{Name: "host", Help: "host whose frontier to discover"}},
	}, func(ctx context.Context, in sitemapIn, emit func(frontierURL) error) error {
		host := normalizeHost(in.Host)
		if host == "" {
			return usageErr("a host is required, e.g. example.com")
		}
		urls, err := in.Engine.Sitemap(ctx, host, kumo.ScrapeOptions{
			IncludeSubs: in.Subdomains,
			NoRobots:    in.NoRobots,
		})
		if err != nil {
			return err
		}
		for _, u := range urls {
			if err := emit(frontierURL{URL: u}); err != nil {
				return err
			}
		}
		return nil
	})
}

type linksIn struct {
	Engine *kumo.Engine `kit:"inject"`
	URL    string       `kit:"arg" help:"the page URL whose links to list"`
}

func registerLinks(app *kit.App) {
	kit.Handle(app, kit.OpMeta{
		Name:    "links",
		Group:   "read",
		Summary: "List the outbound links of a page, as URIs",
		Args:    []kit.Arg{{Name: "url", Help: "the page URL whose links to list"}},
	}, func(ctx context.Context, in linksIn, emit func(kumo.Link) error) error {
		if strings.TrimSpace(in.URL) == "" {
			return usageErr("a page URL is required")
		}
		p, err := in.Engine.Page(ctx, in.URL)
		if err != nil {
			return err
		}
		for _, l := range p.Links {
			if err := emit(l); err != nil {
				return err
			}
		}
		return nil
	})
}

type pagesIn struct {
	Engine *kumo.Engine `kit:"inject"`
	Host   string       `kit:"arg" help:"host whose stored pages to list"`
}

func registerPages(app *kit.App) {
	kit.Handle(app, kit.OpMeta{
		Name:    "pages",
		Group:   "data",
		Summary: "List the pages already stored on disk for a host",
		Long: `Read the data tree for a host and emit the page records already crawled,
offline. This makes no network request.

Examples:
  kumo pages example.com
  kumo pages example.com -o jsonl`,
		Args: []kit.Arg{{Name: "host", Help: "host whose stored pages to list"}},
	}, func(_ context.Context, in pagesIn, emit func(*kumo.Page) error) error {
		host := normalizeHost(in.Host)
		if host == "" {
			return usageErr("a host is required, e.g. example.com")
		}
		pages, err := in.Engine.StoredPages(host)
		if err != nil {
			return err
		}
		for _, p := range pages {
			if err := emit(p); err != nil {
				return err
			}
		}
		return nil
	})
}

// normalizeHost accepts a bare host or a full URL and returns the lower-cased
// host with any scheme, path, and trailing dot removed.
func normalizeHost(arg string) string {
	arg = strings.TrimSpace(arg)
	if arg == "" {
		return ""
	}
	if strings.Contains(arg, "://") {
		if u, err := url.Parse(arg); err == nil && u.Hostname() != "" {
			return strings.ToLower(u.Hostname())
		}
	}
	// Strip any path the user may have appended to a bare host.
	if i := strings.IndexByte(arg, '/'); i >= 0 {
		arg = arg[:i]
	}
	return strings.ToLower(strings.TrimSuffix(arg, "."))
}
