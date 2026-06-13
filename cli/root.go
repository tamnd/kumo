// Package cli builds the kumo command tree on top of the kumo library and the
// any-cli/kit framework. Every command is a kit operation: declared once and
// exposed as a CLI subcommand, an HTTP route, and an MCP tool, with --limit,
// the --db store tee, and the output formats handled by the framework.
package cli

import (
	"context"
	"net/http"
	"time"

	"github.com/tamnd/any-cli/kit"
	"github.com/tamnd/any-cli/kit/errs"
	"github.com/tamnd/kumo/kumo"
)

// Build metadata, set via -ldflags at release time.
var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

// domainGlobals holds the kumo-specific persistent flags that are not part of
// the kit baseline.
type domainGlobals struct {
	workers int
	out     string
}

// builder wires the kumo globals and defaults into a kit.App.
type builder struct {
	dom *domainGlobals
}

// NewApp assembles the kit application: identity, defaults, the kumo global
// flags, the client factory that builds the shared engine, and the operations.
func NewApp() *kit.App {
	b := &builder{dom: &domainGlobals{}}

	app := kit.New(kit.Identity{
		Binary:  "kumo",
		Version: Version,
		Short:   "Crawl a whole host into structured data",
		Long: `kumo takes a host, crawls every page, converts each to clean structured
Markdown and JSON, and saves the result as a navigable URI tree under your data
directory.

Quick start:
  kumo scrape example.com                  crawl the whole host
  kumo scrape example.com --max-pages 200  bound the crawl
  kumo page https://example.com/docs/x     one page as a structured record
  kumo sitemap example.com                 just the discoverable URL frontier`,
		Site: "the open web",
		Repo: "https://github.com/tamnd/kumo",
	}, kit.WithDefaults(b.defaults))

	app.GlobalFlags(b.globals)
	app.SetClient(b.client)

	registerOps(app)
	app.AddCommand(newVersionCmd())
	return app
}

// defaults seeds the framework baseline with kumo's politer values, so an unset
// --rate/--retries/--timeout keeps these.
func (b *builder) defaults(c *kit.Config) {
	c.Rate = time.Second
	c.Retries = 3
	c.Timeout = 30 * time.Second
	c.Workers = 16
	c.UserAgent = kumo.DefaultUserAgent
}

// globals registers the kumo-specific persistent flags on top of the kit
// baseline.
func (b *builder) globals(f *kit.FlagSet) {
	f.IntVarP(&b.dom.workers, "workers", "j", 16, "number of concurrent fetchers")
	f.StringVar(&b.dom.out, "out", kumo.DefaultDataRoot(), "root of the data tree pages are written into")
}

// client builds the shared engine from the resolved config and the kumo
// globals. kit injects it into the handler fields tagged kit:"inject".
func (b *builder) client(_ context.Context, c kit.Config) (any, error) {
	return &kumo.Engine{
		HTTP:      &http.Client{Timeout: c.Timeout},
		UserAgent: c.UserAgent,
		Rate:      c.Rate,
		Retries:   c.Retries,
		Workers:   b.dom.workers,
		DataRoot:  b.dom.out,
		DryRun:    c.DryRun,
	}, nil
}

func usageErr(format string, args ...any) error { return errs.Usage(format, args...) }
