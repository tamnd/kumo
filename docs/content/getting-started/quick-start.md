---
title: "Quick start"
description: "Run your first kumo command."
weight: 30
---

Once `kumo` is on your `PATH`, crawl a host into the data tree:

```bash
kumo scrape example.com --max-pages 20
```

Each page is written as `pages/<host>/<path>.md` under your data directory: a
JSON front-matter block with every structured field, followed by the page
content as Markdown. Read back what you crawled, offline:

```bash
kumo pages example.com -o table
```

Before a full crawl, look at the frontier without fetching anything:

```bash
kumo scrape example.com --dry-run -o url     # the URLs a crawl would start from
kumo sitemap example.com                     # the same, from robots.txt and sitemaps
```

Work with a single page when you do not want a whole crawl:

```bash
kumo page https://example.com/ -o json | jq .title
kumo links https://example.com/              # its outbound links, as URIs
```

Add `-o jsonl` to stream records into the rest of your tools, and `--limit` to
cap any command. See the [CLI reference](/reference/cli/) for the full surface.
