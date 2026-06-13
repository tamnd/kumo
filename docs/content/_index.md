---
title: "kumo"
description: "kumo takes a host, crawls every page, converts each to clean structured Markdown and JSON, and saves the result as a navigable URI tree under your data directory."
heroTitle: "kumo, from the command line"
heroLead: "Crawl a whole host into structured data. One pure-Go binary, no API key, output that pipes into the rest of your tools."
heroPrimaryURL: "/getting-started/quick-start/"
heroPrimaryText: "Get started"
---

Crawl a whole host into structured data.

```bash
kumo scrape example.com --max-pages 20   # crawl a host into $HOME/data
kumo pages example.com -o table          # read back what you crawled
kumo page https://example.com/ -o json   # structure a single page
```

Each page is written as `pages/<host>/<path>.md`: a JSON front-matter block with
the title, description, canonical, language, dates, OpenGraph, JSON-LD, and
outbound links, followed by the main content as Markdown.

## Where to go next

- New here? Read the [introduction](/getting-started/introduction/), then the
  [quick start](/getting-started/quick-start/).
- Installing? See [installation](/getting-started/installation/).
- Need every flag? The [CLI reference](/reference/cli/) is the full surface.
