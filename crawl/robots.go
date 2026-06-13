package crawl

import (
	"context"
	"net/url"
	"time"

	"github.com/temoto/robotstxt"
)

// robotsPolicy is the crawl's view of a host's robots.txt: whether a path may be
// fetched, the crawl delay to honor, and any sitemaps the file advertises.
type robotsPolicy struct {
	allowAll bool
	group    *robotstxt.Group
	delay    time.Duration
	sitemaps []string
}

// robotsAgent is the product token kumo presents to robots.txt for group
// matching. It is the stable name, independent of the full User-Agent string.
const robotsAgent = "kumo"

// fetchRobots loads and parses robots.txt for a host. A missing file, a network
// error, or a 4xx is treated as "allow everything", which is the conventional
// reading of an absent robots.txt.
func fetchRobots(ctx context.Context, f *fetcher, scheme, host string) *robotsPolicy {
	robotsURL := scheme + "://" + host + "/robots.txt"
	res, err := f.get(ctx, robotsURL, conds{})
	if err != nil || res == nil || res.Status != 200 || len(res.Body) == 0 {
		return &robotsPolicy{allowAll: true}
	}
	data, err := robotstxt.FromStatusAndBytes(res.Status, res.Body)
	if err != nil {
		return &robotsPolicy{allowAll: true}
	}
	g := data.FindGroup(robotsAgent)
	p := &robotsPolicy{group: g, sitemaps: data.Sitemaps}
	if g != nil {
		p.delay = g.CrawlDelay
	}
	return p
}

// Allowed reports whether a URL's path may be fetched under this policy.
func (p *robotsPolicy) Allowed(raw string) bool {
	if p == nil || p.allowAll || p.group == nil {
		return true
	}
	u, err := url.Parse(raw)
	if err != nil {
		return true
	}
	path := u.Path
	if path == "" {
		path = "/"
	}
	if u.RawQuery != "" {
		path += "?" + u.RawQuery
	}
	return p.group.Test(path)
}
