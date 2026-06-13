---
title: "Guides"
linkTitle: "Guides"
description: "Task-oriented walkthroughs for the things people do with kumo."
weight: 20
featured: true
---

Each guide is built around a job rather than a command. They assume you have run
the [quick start](/getting-started/quick-start/).

## Mirror a documentation site

Crawl just the docs and keep the converted Markdown:

```bash
kumo scrape example.com --scope-prefix /docs --raw-md
kumo pages example.com -o table
```

`--scope-prefix` keeps the crawl under one path, and `--raw-md` prefers a site's
own `<page>.md` source where it publishes one.

## Take a polite, bounded sample

When you only need a feel for a site, cap the crawl and slow it down:

```bash
kumo scrape example.com --max-pages 50 --max-depth 2 --rate 1s
```

## Plan before you fetch

See what a crawl would touch without making a single page request:

```bash
kumo sitemap example.com -o url | head
kumo scrape example.com --dry-run -o url
```
